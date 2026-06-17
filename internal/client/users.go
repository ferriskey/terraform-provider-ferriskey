package client

import (
	"context"
	"fmt"
	"net/url"
)

func userBase(realm string) string {
	return fmt.Sprintf("/realms/%s/users", url.PathEscape(realm))
}

// GetUser fetches a user by UUID. Returns ErrNotFound if absent. The response
// is wrapped in a { "data": User } envelope.
func (c *Client) GetUser(ctx context.Context, realm, id string) (*User, error) {
	var env userEnvelope
	if err := c.do(ctx, "GET", userBase(realm)+"/"+url.PathEscape(id), nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// CreateUser creates a user. The response is an { "data": User } envelope.
func (c *Client) CreateUser(ctx context.Context, realm string, req CreateUserRequest) (*User, error) {
	var env userEnvelope
	if err := c.do(ctx, "POST", userBase(realm), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// UpdateUser updates a user. The response is an { "data": User } envelope.
func (c *Client) UpdateUser(ctx context.Context, realm, id string, req UpdateUserRequest) (*User, error) {
	var env userEnvelope
	if err := c.do(ctx, "PUT", userBase(realm)+"/"+url.PathEscape(id), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// DeleteUser deletes a user by UUID.
func (c *Client) DeleteUser(ctx context.Context, realm, id string) error {
	if err := c.do(ctx, "DELETE", userBase(realm)+"/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("deleting user %q: %w", id, err)
	}
	return nil
}
