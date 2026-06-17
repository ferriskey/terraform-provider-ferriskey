package client

import (
	"context"
	"fmt"
	"net/url"
)

func idpBase(realm string) string {
	return fmt.Sprintf("/realms/%s/identity-providers", url.PathEscape(realm))
}

// GetIdentityProvider fetches an identity provider by alias. ErrNotFound if absent.
func (c *Client) GetIdentityProvider(ctx context.Context, realm, alias string) (*IdentityProvider, error) {
	var idp IdentityProvider
	if err := c.do(ctx, "GET", idpBase(realm)+"/"+url.PathEscape(alias), nil, &idp); err != nil {
		return nil, err
	}
	return &idp, nil
}

// CreateIdentityProvider creates an identity provider (bare response).
func (c *Client) CreateIdentityProvider(ctx context.Context, realm string, req CreateIdentityProviderRequest) (*IdentityProvider, error) {
	var idp IdentityProvider
	if err := c.do(ctx, "POST", idpBase(realm), req, &idp); err != nil {
		return nil, err
	}
	return &idp, nil
}

// UpdateIdentityProvider updates an identity provider by alias (bare response).
func (c *Client) UpdateIdentityProvider(ctx context.Context, realm, alias string, req UpdateIdentityProviderRequest) (*IdentityProvider, error) {
	var idp IdentityProvider
	if err := c.do(ctx, "PUT", idpBase(realm)+"/"+url.PathEscape(alias), req, &idp); err != nil {
		return nil, err
	}
	return &idp, nil
}

// DeleteIdentityProvider deletes an identity provider by alias.
func (c *Client) DeleteIdentityProvider(ctx context.Context, realm, alias string) error {
	if err := c.do(ctx, "DELETE", idpBase(realm)+"/"+url.PathEscape(alias), nil, nil); err != nil {
		return fmt.Errorf("deleting identity provider %q: %w", alias, err)
	}
	return nil
}
