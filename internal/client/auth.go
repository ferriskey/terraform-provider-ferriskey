package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// GrantType enumerates the OAuth2 grant types the provider supports for
// authenticating against FerrisKey.
type GrantType string

const (
	// GrantPassword authenticates with a resource-owner password against a
	// public client (e.g. admin-cli). Used for the bootstrap phase with the
	// initial admin account created by the Helm chart.
	GrantPassword GrantType = "password"
	// GrantClientCredentials authenticates a confidential service account
	// (client_id + client_secret). Used for steady-state automation.
	GrantClientCredentials GrantType = "client_credentials"
)

// tokenExpiryMargin is subtracted from the advertised token lifetime so a token
// is proactively refreshed before it actually expires. A long `terraform apply`
// can otherwise outlive a short-lived access token mid-run — a real foot-gun.
const tokenExpiryMargin = 30 * time.Second

// AuthConfig holds everything needed to obtain access tokens from a FerrisKey
// realm's OIDC token endpoint.
type AuthConfig struct {
	// Realm is the authentication realm (often the admin/master realm), not
	// necessarily the realm the resources live in.
	Realm        string
	GrantType    GrantType
	Username     string
	Password     string
	ClientID     string
	ClientSecret string
	// Scope is optional; when empty no scope parameter is sent.
	Scope string
}

// tokenManager fetches and caches access tokens, refreshing them transparently
// when they expire. It is safe for concurrent use.
type tokenManager struct {
	baseURL    string
	httpClient *http.Client
	cfg        AuthConfig
	// now is injected for deterministic testing; defaults to time.Now.
	now func() time.Time

	mu               sync.Mutex
	accessToken      string
	accessExpiresAt  time.Time
	refreshToken     string
	refreshExpiresAt time.Time
}

func newTokenManager(baseURL string, httpClient *http.Client, cfg AuthConfig) *tokenManager {
	return &tokenManager{
		baseURL:    baseURL,
		httpClient: httpClient,
		cfg:        cfg,
		now:        time.Now,
	}
}

// token returns a valid access token, fetching or refreshing as needed.
func (m *tokenManager) token(ctx context.Context) (string, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := m.now()
	if m.accessToken != "" && now.Before(m.accessExpiresAt) {
		return m.accessToken, nil
	}

	// Prefer a refresh-token exchange when we still hold a usable refresh
	// token; it is cheaper and, for the password grant, avoids re-sending
	// credentials.
	if m.refreshToken != "" && now.Before(m.refreshExpiresAt) {
		if tok, err := m.requestToken(ctx, url.Values{
			"grant_type":    {"refresh_token"},
			"refresh_token": {m.refreshToken},
		}); err == nil {
			m.store(tok)
			tflog.Debug(ctx, "refreshed FerrisKey access token", map[string]any{"expires_in": tok.ExpiresIn})
			return m.accessToken, nil
		}
		// Fall through to a fresh grant if the refresh failed (e.g. the
		// refresh token was revoked).
		tflog.Debug(ctx, "refresh token exchange failed; falling back to a fresh grant", nil)
	}

	tok, err := m.requestToken(ctx, m.grantValues())
	if err != nil {
		return "", err
	}
	m.store(tok)
	tflog.Debug(ctx, "obtained FerrisKey access token",
		map[string]any{"grant_type": string(m.cfg.GrantType), "expires_in": tok.ExpiresIn})
	return m.accessToken, nil
}

// grantValues builds the form parameters for the configured primary grant.
func (m *tokenManager) grantValues() url.Values {
	v := url.Values{"grant_type": {string(m.cfg.GrantType)}}
	switch m.cfg.GrantType {
	case GrantPassword:
		v.Set("username", m.cfg.Username)
		v.Set("password", m.cfg.Password)
		v.Set("client_id", m.cfg.ClientID)
		if m.cfg.ClientSecret != "" {
			v.Set("client_secret", m.cfg.ClientSecret)
		}
	case GrantClientCredentials:
		v.Set("client_id", m.cfg.ClientID)
		v.Set("client_secret", m.cfg.ClientSecret)
	}
	if m.cfg.Scope != "" {
		v.Set("scope", m.cfg.Scope)
	}
	return v
}

func (m *tokenManager) store(tok *JwtToken) {
	now := m.now()
	m.accessToken = tok.AccessToken
	m.accessExpiresAt = now.Add(time.Duration(tok.ExpiresIn)*time.Second - tokenExpiryMargin)
	m.refreshToken = tok.RefreshToken
	if tok.RefreshExpiresIn > 0 {
		m.refreshExpiresAt = now.Add(time.Duration(tok.RefreshExpiresIn)*time.Second - tokenExpiryMargin)
	} else {
		m.refreshExpiresAt = time.Time{}
	}
}

// requestToken POSTs to the realm token endpoint. The body is
// application/x-www-form-urlencoded — see the note on JwtToken.
func (m *tokenManager) requestToken(ctx context.Context, form url.Values) (*JwtToken, error) {
	endpoint := fmt.Sprintf("%s/realms/%s/protocol/openid-connect/token", m.baseURL, url.PathEscape(m.cfg.Realm))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return nil, fmt.Errorf("building token request: %w", err)
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := m.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("calling token endpoint %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		var apiErr APIError
		if json.Unmarshal(body, &apiErr) == nil && apiErr.Message != "" {
			return nil, fmt.Errorf("token endpoint returned %d: %s (%s)", resp.StatusCode, apiErr.Message, apiErr.Code)
		}
		return nil, fmt.Errorf("token endpoint returned %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	var tok JwtToken
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, fmt.Errorf("decoding token response: %w", err)
	}
	if tok.AccessToken == "" {
		return nil, fmt.Errorf("token endpoint returned an empty access_token")
	}
	return &tok, nil
}

// invalidate clears the cached access token, forcing the next call to
// re-authenticate. Used when the API rejects a token with 401 despite our
// local clock believing it is still valid (e.g. server restart, clock skew).
func (m *tokenManager) invalidate() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.accessToken = ""
	m.accessExpiresAt = time.Time{}
}
