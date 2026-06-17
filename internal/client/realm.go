package client

import (
	"context"
	"fmt"
	"net/url"
)

// GetRealm fetches a realm by name. Returns ErrNotFound if absent.
func (c *Client) GetRealm(ctx context.Context, name string) (*Realm, error) {
	var realm Realm
	if err := c.do(ctx, "GET", "/realms/"+url.PathEscape(name), nil, &realm); err != nil {
		return nil, err
	}
	return &realm, nil
}

// CreateRealm creates a realm and returns the created object.
func (c *Client) CreateRealm(ctx context.Context, req CreateRealmRequest) (*Realm, error) {
	var realm Realm
	if err := c.do(ctx, "POST", "/realms", req, &realm); err != nil {
		return nil, err
	}
	return &realm, nil
}

// UpdateRealm renames a realm. The endpoint returns an { "data": Realm }
// envelope.
func (c *Client) UpdateRealm(ctx context.Context, name string, req UpdateRealmRequest) (*Realm, error) {
	var env realmEnvelope
	if err := c.do(ctx, "PUT", "/realms/"+url.PathEscape(name), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// DeleteRealm deletes a realm by name.
func (c *Client) DeleteRealm(ctx context.Context, name string) error {
	if err := c.do(ctx, "DELETE", "/realms/"+url.PathEscape(name), nil, nil); err != nil {
		return fmt.Errorf("deleting realm %q: %w", name, err)
	}
	return nil
}
