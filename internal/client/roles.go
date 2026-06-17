package client

import (
	"context"
	"fmt"
	"net/url"
)

func roleBase(realm string) string {
	return fmt.Sprintf("/realms/%s/roles", url.PathEscape(realm))
}

// GetRole fetches a role by UUID. Returns ErrNotFound if absent.
func (c *Client) GetRole(ctx context.Context, realm, id string) (*Role, error) {
	var env roleEnvelope
	if err := c.do(ctx, "GET", roleBase(realm)+"/"+url.PathEscape(id), nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// CreateRole creates a realm role. The response is an { "data": Role } envelope.
func (c *Client) CreateRole(ctx context.Context, realm string, req CreateRoleRequest) (*Role, error) {
	var env roleEnvelope
	if err := c.do(ctx, "POST", roleBase(realm), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// UpdateRole updates a role's name/description. Permissions are managed
// separately (see UpdateRolePermissions).
func (c *Client) UpdateRole(ctx context.Context, realm, id string, req UpdateRoleRequest) (*Role, error) {
	var env roleEnvelope
	if err := c.do(ctx, "PUT", roleBase(realm)+"/"+url.PathEscape(id), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// UpdateRolePermissions replaces a role's permission set via the dedicated
// PATCH endpoint.
func (c *Client) UpdateRolePermissions(ctx context.Context, realm, id string, permissions []string) (*Role, error) {
	var env roleEnvelope
	req := UpdateRolePermissionsRequest{Permissions: permissions}
	if err := c.do(ctx, "PATCH", roleBase(realm)+"/"+url.PathEscape(id)+"/permissions", req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// DeleteRole deletes a role by UUID.
func (c *Client) DeleteRole(ctx context.Context, realm, id string) error {
	if err := c.do(ctx, "DELETE", roleBase(realm)+"/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("deleting role %q: %w", id, err)
	}
	return nil
}
