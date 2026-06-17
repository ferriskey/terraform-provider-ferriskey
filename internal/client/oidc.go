package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
)

// OpenIDConfiguration mirrors the public OIDC discovery document exposed at
// /realms/{realm}/.well-known/openid-configuration.
type OpenIDConfiguration struct {
	Issuer                            string   `json:"issuer"`
	AuthorizationEndpoint             string   `json:"authorization_endpoint"`
	TokenEndpoint                     string   `json:"token_endpoint"`
	RevocationEndpoint                string   `json:"revocation_endpoint"`
	EndSessionEndpoint                string   `json:"end_session_endpoint"`
	IntrospectionEndpoint             string   `json:"introspection_endpoint"`
	UserinfoEndpoint                  string   `json:"userinfo_endpoint"`
	JwksURI                           string   `json:"jwks_uri"`
	GrantTypesSupported               []string `json:"grant_types_supported"`
	TokenEndpointAuthMethodsSupported []string `json:"token_endpoint_auth_methods_supported"`
}

// GetOpenIDConfiguration fetches the public discovery document for a realm. It
// is unauthenticated (the endpoint is public).
func (c *Client) GetOpenIDConfiguration(ctx context.Context, realm string) (*OpenIDConfiguration, error) {
	endpoint := fmt.Sprintf("%s/realms/%s/.well-known/openid-configuration", c.baseURL, url.PathEscape(realm))
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return nil, fmt.Errorf("building discovery request: %w", err)
	}
	req.Header.Set("Accept", "application/json")
	if c.userAgent != "" {
		req.Header.Set("User-Agent", c.userAgent)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetching discovery document: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)
	if resp.StatusCode == http.StatusNotFound {
		return nil, ErrNotFound
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, &APIErrorResponse{StatusCode: resp.StatusCode}
	}

	var oidc OpenIDConfiguration
	if err := json.Unmarshal(body, &oidc); err != nil {
		return nil, fmt.Errorf("decoding discovery document: %w", err)
	}
	return &oidc, nil
}
