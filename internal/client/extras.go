package client

import (
	"context"
	"fmt"
	"net/url"
)

// ---------------------------------------------------------------------------
// Password policy
// ---------------------------------------------------------------------------

// GetPasswordPolicy reads a realm's password policy (always exists).
func (c *Client) GetPasswordPolicy(ctx context.Context, realm string) (*PasswordPolicy, error) {
	var p PasswordPolicy
	if err := c.do(ctx, "GET", "/realms/"+url.PathEscape(realm)+"/password-policy", nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// UpdatePasswordPolicy updates a realm's password policy and returns the result.
func (c *Client) UpdatePasswordPolicy(ctx context.Context, realm string, req UpdatePasswordPolicyRequest) (*PasswordPolicy, error) {
	var p PasswordPolicy
	if err := c.do(ctx, "PUT", "/realms/"+url.PathEscape(realm)+"/password-policy", req, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

// ---------------------------------------------------------------------------
// SMTP config
// ---------------------------------------------------------------------------

// GetSmtpConfig reads a realm's SMTP config. Returns ErrNotFound when unset.
func (c *Client) GetSmtpConfig(ctx context.Context, realm string) (*SmtpConfig, error) {
	var s SmtpConfig
	if err := c.do(ctx, "GET", "/realms/"+url.PathEscape(realm)+"/smtp-config", nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// UpsertSmtpConfig creates or updates a realm's SMTP config.
func (c *Client) UpsertSmtpConfig(ctx context.Context, realm string, req UpsertSmtpConfigRequest) (*SmtpConfig, error) {
	var s SmtpConfig
	if err := c.do(ctx, "PUT", "/realms/"+url.PathEscape(realm)+"/smtp-config", req, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

// DeleteSmtpConfig removes a realm's SMTP config.
func (c *Client) DeleteSmtpConfig(ctx context.Context, realm string) error {
	if err := c.do(ctx, "DELETE", "/realms/"+url.PathEscape(realm)+"/smtp-config", nil, nil); err != nil {
		return fmt.Errorf("deleting SMTP config for realm %q: %w", realm, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Organization members
// ---------------------------------------------------------------------------

func orgMembersBase(realm, orgID string) string {
	return fmt.Sprintf("/realms/%s/organizations/%s/members", url.PathEscape(realm), url.PathEscape(orgID))
}

// ListOrganizationMembers returns every member of an organization, walking the
// paginated { "data": [...], "metadata": {...} } envelope.
func (c *Client) ListOrganizationMembers(ctx context.Context, realm, orgID string) ([]OrganizationMember, error) {
	var all []OrganizationMember
	for page := 1; ; page++ {
		var env struct {
			Data     []OrganizationMember `json:"data"`
			Metadata orgPageMeta          `json:"metadata"`
		}
		if err := c.do(ctx, "GET", orgExtPage(orgMembersBase(realm, orgID), page), nil, &env); err != nil {
			return nil, err
		}
		all = append(all, env.Data...)
		if page >= env.Metadata.TotalPages {
			return all, nil
		}
	}
}

// AddOrganizationMember adds a user to an organization.
func (c *Client) AddOrganizationMember(ctx context.Context, realm, orgID, userID string) (*OrganizationMember, error) {
	var m OrganizationMember
	req := AddOrganizationMemberRequest{UserID: userID}
	if err := c.do(ctx, "POST", orgMembersBase(realm, orgID), req, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

// RemoveOrganizationMember removes a user from an organization.
func (c *Client) RemoveOrganizationMember(ctx context.Context, realm, orgID, userID string) error {
	path := orgMembersBase(realm, orgID) + "/" + url.PathEscape(userID)
	if err := c.do(ctx, "DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("removing member %q from organization %q: %w", userID, orgID, err)
	}
	return nil
}
