package client

import (
	"context"
	"fmt"
	"net/url"
)

func orgBase(realm string) string {
	return fmt.Sprintf("/realms/%s/organizations", url.PathEscape(realm))
}

// GetOrganization fetches an organization by UUID. Returns ErrNotFound if
// absent.
func (c *Client) GetOrganization(ctx context.Context, realm, id string) (*Organization, error) {
	var org Organization
	if err := c.do(ctx, "GET", orgBase(realm)+"/"+url.PathEscape(id), nil, &org); err != nil {
		return nil, err
	}
	return &org, nil
}

// CreateOrganization creates an organization. The 201 response is the bare
// Organization object.
func (c *Client) CreateOrganization(ctx context.Context, realm string, req CreateOrganizationRequest) (*Organization, error) {
	var org Organization
	if err := c.do(ctx, "POST", orgBase(realm), req, &org); err != nil {
		return nil, err
	}
	return &org, nil
}

// UpdateOrganization updates an organization. The 200 response is the bare
// Organization object.
func (c *Client) UpdateOrganization(ctx context.Context, realm, id string, req UpdateOrganizationRequest) (*Organization, error) {
	var org Organization
	if err := c.do(ctx, "PUT", orgBase(realm)+"/"+url.PathEscape(id), req, &org); err != nil {
		return nil, err
	}
	return &org, nil
}

// DeleteOrganization deletes an organization by UUID.
func (c *Client) DeleteOrganization(ctx context.Context, realm, id string) error {
	if err := c.do(ctx, "DELETE", orgBase(realm)+"/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("deleting organization %q: %w", id, err)
	}
	return nil
}
