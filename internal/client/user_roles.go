package client

import (
	"context"
	"fmt"
	"net/url"
)

// ListUsers returns all users in a realm. The endpoint is not paginated.
func (c *Client) ListUsers(ctx context.Context, realm string) ([]User, error) {
	var env struct {
		Data []User `json:"data"`
	}
	if err := c.do(ctx, "GET", userBase(realm), nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// FindUserByUsername returns the user with the given username, or ErrNotFound.
func (c *Client) FindUserByUsername(ctx context.Context, realm, username string) (*User, error) {
	users, err := c.ListUsers(ctx, realm)
	if err != nil {
		return nil, err
	}
	for i := range users {
		if users[i].Username == username {
			return &users[i], nil
		}
	}
	return nil, ErrNotFound
}

// ServiceAccountUsername returns the deterministic username FerrisKey assigns to
// a client's service account user.
func ServiceAccountUsername(clientID string) string {
	return "service-account-" + clientID
}

// ListRoles returns all realm roles. The endpoint is not paginated.
func (c *Client) ListRoles(ctx context.Context, realm string) ([]Role, error) {
	var env struct {
		Data []Role `json:"data"`
	}
	if err := c.do(ctx, "GET", roleBase(realm), nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// FindRoleByName returns the realm role with the given name, or ErrNotFound.
func (c *Client) FindRoleByName(ctx context.Context, realm, name string) (*Role, error) {
	roles, err := c.ListRoles(ctx, realm)
	if err != nil {
		return nil, err
	}
	for i := range roles {
		if roles[i].Name == name {
			return &roles[i], nil
		}
	}
	return nil, ErrNotFound
}

// ListUserRoles returns the roles assigned to a user.
func (c *Client) ListUserRoles(ctx context.Context, realm, userID string) ([]Role, error) {
	var env struct {
		Data []Role `json:"data"`
	}
	path := userBase(realm) + "/" + url.PathEscape(userID) + "/roles"
	if err := c.do(ctx, "GET", path, nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// AssignRoleToUser assigns a role to a user (idempotent on the server side).
func (c *Client) AssignRoleToUser(ctx context.Context, realm, userID, roleID string) error {
	path := userBase(realm) + "/" + url.PathEscape(userID) + "/roles/" + url.PathEscape(roleID)
	if err := c.do(ctx, "POST", path, nil, nil); err != nil {
		return fmt.Errorf("assigning role %q to user %q: %w", roleID, userID, err)
	}
	return nil
}

// RemoveRoleFromUser removes a role from a user.
func (c *Client) RemoveRoleFromUser(ctx context.Context, realm, userID, roleID string) error {
	path := userBase(realm) + "/" + url.PathEscape(userID) + "/roles/" + url.PathEscape(roleID)
	if err := c.do(ctx, "DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("removing role %q from user %q: %w", roleID, userID, err)
	}
	return nil
}
