package client

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

type ClientWebOrigin struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	Value     string    `json:"value"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateClientWebOriginRequest struct {
	Value string `json:"value"`
}

type ClientPostLogoutRedirectURI struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	Value     string    `json:"value"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateClientPostLogoutRedirectURIRequest struct {
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

type UpdateClientPostLogoutRedirectURIRequest struct {
	Enabled bool `json:"enabled"`
}

type clientPostLogoutRedirectURIEnvelope struct {
	Data ClientPostLogoutRedirectURI `json:"data"`
}

type ClientTokenExchangePolicy struct {
	ID                 string    `json:"id"`
	ClientID           string    `json:"client_id"`
	TargetAudience     string    `json:"target_audience"`
	AllowedScopes      []string  `json:"allowed_scopes"`
	AllowImpersonation bool      `json:"allow_impersonation"`
	AllowDelegation    bool      `json:"allow_delegation"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type CreateClientTokenExchangePolicyRequest struct {
	TargetAudience     string   `json:"target_audience"`
	AllowedScopes      []string `json:"allowed_scopes"`
	AllowImpersonation bool     `json:"allow_impersonation"`
	AllowDelegation    bool     `json:"allow_delegation"`
}

func clientExtBase(realm, clientID, segment string) string {
	return fmt.Sprintf("/realms/%s/clients/%s/%s", url.PathEscape(realm), url.PathEscape(clientID), segment)
}

func (c *Client) CreateClientRole(ctx context.Context, realm, clientID string, req CreateRoleRequest) (*Role, error) {
	var out Role
	if err := c.do(ctx, "POST", clientExtBase(realm, clientID, "roles"), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) ListClientWebOrigins(ctx context.Context, realm, clientID string) ([]ClientWebOrigin, error) {
	var out []ClientWebOrigin
	if err := c.do(ctx, "GET", clientExtBase(realm, clientID, "web-origins"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetClientWebOrigin(ctx context.Context, realm, clientID, id string) (*ClientWebOrigin, error) {
	items, err := c.ListClientWebOrigins(ctx, realm, clientID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == id {
			return &items[i], nil
		}
	}
	return nil, ErrNotFound
}

func (c *Client) CreateClientWebOrigin(ctx context.Context, realm, clientID string, req CreateClientWebOriginRequest) (*ClientWebOrigin, error) {
	var out ClientWebOrigin
	if err := c.do(ctx, "POST", clientExtBase(realm, clientID, "web-origins"), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteClientWebOrigin(ctx context.Context, realm, clientID, id string) error {
	if err := c.do(ctx, "DELETE", clientExtBase(realm, clientID, "web-origins")+"/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("deleting web origin %q: %w", id, err)
	}
	return nil
}

func (c *Client) ListClientPostLogoutRedirectURIs(ctx context.Context, realm, clientID string) ([]ClientPostLogoutRedirectURI, error) {
	var out []ClientPostLogoutRedirectURI
	if err := c.do(ctx, "GET", clientExtBase(realm, clientID, "post-logout-redirects"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetClientPostLogoutRedirectURI(ctx context.Context, realm, clientID, id string) (*ClientPostLogoutRedirectURI, error) {
	items, err := c.ListClientPostLogoutRedirectURIs(ctx, realm, clientID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == id {
			return &items[i], nil
		}
	}
	return nil, ErrNotFound
}

func (c *Client) CreateClientPostLogoutRedirectURI(ctx context.Context, realm, clientID string, req CreateClientPostLogoutRedirectURIRequest) (*ClientPostLogoutRedirectURI, error) {
	var out ClientPostLogoutRedirectURI
	if err := c.do(ctx, "POST", clientExtBase(realm, clientID, "post-logout-redirects"), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateClientPostLogoutRedirectURI(ctx context.Context, realm, clientID, id string, req UpdateClientPostLogoutRedirectURIRequest) (*ClientPostLogoutRedirectURI, error) {
	var env clientPostLogoutRedirectURIEnvelope
	if err := c.do(ctx, "PUT", clientExtBase(realm, clientID, "post-logout-redirects")+"/"+url.PathEscape(id), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) DeleteClientPostLogoutRedirectURI(ctx context.Context, realm, clientID, id string) error {
	if err := c.do(ctx, "DELETE", clientExtBase(realm, clientID, "post-logout-redirects")+"/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("deleting post-logout redirect URI %q: %w", id, err)
	}
	return nil
}

func (c *Client) ListClientTokenExchangePolicies(ctx context.Context, realm, clientID string) ([]ClientTokenExchangePolicy, error) {
	var out []ClientTokenExchangePolicy
	if err := c.do(ctx, "GET", clientExtBase(realm, clientID, "token-exchange-policies"), nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetClientTokenExchangePolicy(ctx context.Context, realm, clientID, id string) (*ClientTokenExchangePolicy, error) {
	items, err := c.ListClientTokenExchangePolicies(ctx, realm, clientID)
	if err != nil {
		return nil, err
	}
	for i := range items {
		if items[i].ID == id {
			return &items[i], nil
		}
	}
	return nil, ErrNotFound
}

func (c *Client) CreateClientTokenExchangePolicy(ctx context.Context, realm, clientID string, req CreateClientTokenExchangePolicyRequest) (*ClientTokenExchangePolicy, error) {
	var out ClientTokenExchangePolicy
	if err := c.do(ctx, "POST", clientExtBase(realm, clientID, "token-exchange-policies"), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteClientTokenExchangePolicy(ctx context.Context, realm, clientID, id string) error {
	if err := c.do(ctx, "DELETE", clientExtBase(realm, clientID, "token-exchange-policies")+"/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("deleting token exchange policy %q: %w", id, err)
	}
	return nil
}
