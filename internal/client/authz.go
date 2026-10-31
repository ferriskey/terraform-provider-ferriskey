package client

import (
	"context"
	"fmt"
	"net/url"
)

type AuthzState struct {
	Enabled bool  `json:"enabled"`
	Version int64 `json:"version"`
}

type AuthzSchema struct {
	Namespace string `json:"namespace"`
	Source    string `json:"source"`
	UpdatedAt string `json:"updated_at"`
}

type AuthzBuiltinSchema struct {
	Namespace string `json:"namespace"`
	Source    string `json:"source"`
}

type ReplaceAuthzSchemaRequest struct {
	Namespace *string `json:"namespace,omitempty"`
	Source    string  `json:"source"`
}

type AuthzEntityRef struct {
	Type string `json:"type"`
	ID   string `json:"id"`
}

type AuthzPolicyBody struct {
	Type       string                    `json:"type"`
	Source     *string                   `json:"source,omitempty"`
	TemplateID *string                   `json:"template_id,omitempty"`
	Links      map[string]AuthzEntityRef `json:"links,omitempty"`
}

type AuthzPolicy struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description *string         `json:"description"`
	Origin      string          `json:"origin"`
	Enabled     bool            `json:"enabled"`
	ManagedBy   *string         `json:"managed_by"`
	Body        AuthzPolicyBody `json:"body"`
}

type AuthzPolicyRequest struct {
	Name        string          `json:"name"`
	Description *string         `json:"description,omitempty"`
	Enabled     bool            `json:"enabled"`
	Body        AuthzPolicyBody `json:"body"`
}

type AuthzTemplate struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description *string `json:"description"`
	Source      string  `json:"source"`
}

type AuthzTemplateRequest struct {
	Name        string  `json:"name"`
	Description *string `json:"description,omitempty"`
	Source      string  `json:"source"`
}

type ResourceOwner struct {
	URI      string `json:"uri"`
	ClientID string `json:"client_id"`
}

type resourceOwnersEnvelope struct {
	Data []ResourceOwner `json:"data"`
}

type replaceResourceOwnersRequest struct {
	Owners []ResourceOwner `json:"owners"`
}

func authzBase(realm string) string {
	return "/realms/" + url.PathEscape(realm) + "/authz"
}

func authzManagedQuery(managedBy string) string {
	if managedBy == "" {
		return ""
	}
	return "?managed_by=" + url.QueryEscape(managedBy)
}

func (c *Client) GetAuthzState(ctx context.Context, realm string) (*AuthzState, error) {
	var s AuthzState
	if err := c.do(ctx, "GET", authzBase(realm)+"/state", nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Client) SetAuthzState(ctx context.Context, realm string, enabled bool) (*AuthzState, error) {
	var s AuthzState
	body := struct {
		Enabled bool `json:"enabled"`
	}{Enabled: enabled}
	if err := c.do(ctx, "PUT", authzBase(realm)+"/state", body, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Client) GetAuthzSchema(ctx context.Context, realm string) (*AuthzSchema, error) {
	var s AuthzSchema
	if err := c.do(ctx, "GET", authzBase(realm)+"/schema", nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Client) PutAuthzSchema(ctx context.Context, realm string, req ReplaceAuthzSchemaRequest) (*AuthzSchema, error) {
	var s AuthzSchema
	if err := c.do(ctx, "PUT", authzBase(realm)+"/schema", req, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Client) GetAuthzBuiltinSchema(ctx context.Context, realm string) (*AuthzBuiltinSchema, error) {
	var s AuthzBuiltinSchema
	if err := c.do(ctx, "GET", authzBase(realm)+"/schema/builtin", nil, &s); err != nil {
		return nil, err
	}
	return &s, nil
}

func (c *Client) CreateAuthzPolicy(ctx context.Context, realm, managedBy string, req AuthzPolicyRequest) (*AuthzPolicy, error) {
	var p AuthzPolicy
	if err := c.do(ctx, "POST", authzBase(realm)+"/policies"+authzManagedQuery(managedBy), req, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) GetAuthzPolicy(ctx context.Context, realm, id string) (*AuthzPolicy, error) {
	var p AuthzPolicy
	if err := c.do(ctx, "GET", authzBase(realm)+"/policies/"+url.PathEscape(id), nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) UpdateAuthzPolicy(ctx context.Context, realm, id, managedBy string, req AuthzPolicyRequest) (*AuthzPolicy, error) {
	var p AuthzPolicy
	path := authzBase(realm) + "/policies/" + url.PathEscape(id) + authzManagedQuery(managedBy)
	if err := c.do(ctx, "PUT", path, req, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) DeleteAuthzPolicy(ctx context.Context, realm, id string) error {
	if err := c.do(ctx, "DELETE", authzBase(realm)+"/policies/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("deleting authorization policy %q: %w", id, err)
	}
	return nil
}

func (c *Client) CreateAuthzTemplate(ctx context.Context, realm string, req AuthzTemplateRequest) (*AuthzTemplate, error) {
	var t AuthzTemplate
	if err := c.do(ctx, "POST", authzBase(realm)+"/templates", req, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (c *Client) GetAuthzTemplate(ctx context.Context, realm, id string) (*AuthzTemplate, error) {
	var t AuthzTemplate
	if err := c.do(ctx, "GET", authzBase(realm)+"/templates/"+url.PathEscape(id), nil, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (c *Client) UpdateAuthzTemplate(ctx context.Context, realm, id string, req AuthzTemplateRequest) (*AuthzTemplate, error) {
	var t AuthzTemplate
	if err := c.do(ctx, "PUT", authzBase(realm)+"/templates/"+url.PathEscape(id), req, &t); err != nil {
		return nil, err
	}
	return &t, nil
}

func (c *Client) DeleteAuthzTemplate(ctx context.Context, realm, id string) error {
	if err := c.do(ctx, "DELETE", authzBase(realm)+"/templates/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("deleting authorization template %q: %w", id, err)
	}
	return nil
}

func (c *Client) GetResourceOwners(ctx context.Context, realm string) ([]ResourceOwner, error) {
	var env resourceOwnersEnvelope
	if err := c.do(ctx, "GET", "/realms/"+url.PathEscape(realm)+"/resource-owners", nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

func (c *Client) ReplaceResourceOwners(ctx context.Context, realm string, owners []ResourceOwner) ([]ResourceOwner, error) {
	var env resourceOwnersEnvelope
	if owners == nil {
		owners = []ResourceOwner{}
	}
	req := replaceResourceOwnersRequest{Owners: owners}
	if err := c.do(ctx, "PUT", "/realms/"+url.PathEscape(realm)+"/resource-owners", req, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}
