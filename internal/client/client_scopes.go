package client

import (
	"context"
	"fmt"
	"net/url"
)

const (
	ScopeAssignmentDefault  = "default"
	ScopeAssignmentOptional = "optional"
)

type ClientScope struct {
	ID                         string           `json:"id"`
	RealmID                    string           `json:"realm_id"`
	Name                       string           `json:"name"`
	Description                *string          `json:"description"`
	Protocol                   string           `json:"protocol"`
	DefaultScopeType           string           `json:"default_scope_type"`
	DynamicRegistrationAllowed bool             `json:"dynamic_registration_allowed"`
	ProtocolMappers            []ProtocolMapper `json:"protocol_mappers"`
}

type ProtocolMapper struct {
	ID            string         `json:"id"`
	ClientScopeID string         `json:"client_scope_id"`
	Name          string         `json:"name"`
	MapperType    string         `json:"mapper_type"`
	Config        map[string]any `json:"config"`
}

type CreateClientScopeRequest struct {
	Name                       string  `json:"name"`
	Description                *string `json:"description,omitempty"`
	Protocol                   string  `json:"protocol"`
	IsDefault                  bool    `json:"is_default"`
	DynamicRegistrationAllowed bool    `json:"dynamic_registration_allowed"`
}

type UpdateClientScopeRequest struct {
	Name                       *string `json:"name,omitempty"`
	Description                *string `json:"description,omitempty"`
	Protocol                   *string `json:"protocol,omitempty"`
	IsDefault                  *bool   `json:"is_default,omitempty"`
	DynamicRegistrationAllowed *bool   `json:"dynamic_registration_allowed,omitempty"`
}

type CreateProtocolMapperRequest struct {
	Name       string         `json:"name"`
	MapperType string         `json:"mapper_type"`
	Config     map[string]any `json:"config"`
}

type UpdateProtocolMapperRequest struct {
	Name       *string        `json:"name,omitempty"`
	MapperType *string        `json:"mapper_type,omitempty"`
	Config     map[string]any `json:"config"`
}

type ClientScopeMapping struct {
	ClientID         string `json:"client_id"`
	ScopeID          string `json:"scope_id"`
	DefaultScopeType string `json:"default_scope_type"`
}

func scopeBase(realm string) string {
	return fmt.Sprintf("/realms/%s/client-scopes", url.PathEscape(realm))
}

func scopeMapperPath(realm, scopeID, mapperID string) string {
	return scopeBase(realm) + "/" + url.PathEscape(scopeID) + "/protocol-mappers/" + url.PathEscape(mapperID)
}

func scopeAssignmentPath(realm, clientID, scopeID, assignmentType string) string {
	return fmt.Sprintf("/realms/%s/clients/%s/%s-client-scopes/%s",
		url.PathEscape(realm), url.PathEscape(clientID), assignmentType, url.PathEscape(scopeID))
}

func (c *Client) GetClientScope(ctx context.Context, realm, id string) (*ClientScope, error) {
	var out ClientScope
	if err := c.do(ctx, "GET", scopeBase(realm)+"/"+url.PathEscape(id), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) CreateClientScope(ctx context.Context, realm string, req CreateClientScopeRequest) (*ClientScope, error) {
	var out ClientScope
	if err := c.do(ctx, "POST", scopeBase(realm), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateClientScope(ctx context.Context, realm, id string, req UpdateClientScopeRequest) (*ClientScope, error) {
	var out ClientScope
	if err := c.do(ctx, "PATCH", scopeBase(realm)+"/"+url.PathEscape(id), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteClientScope(ctx context.Context, realm, id string) error {
	if err := c.do(ctx, "DELETE", scopeBase(realm)+"/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("deleting client scope %q: %w", id, err)
	}
	return nil
}

func (c *Client) GetProtocolMapper(ctx context.Context, realm, scopeID, mapperID string) (*ProtocolMapper, error) {
	scope, err := c.GetClientScope(ctx, realm, scopeID)
	if err != nil {
		return nil, err
	}
	for i := range scope.ProtocolMappers {
		if scope.ProtocolMappers[i].ID == mapperID {
			return &scope.ProtocolMappers[i], nil
		}
	}
	return nil, ErrNotFound
}

func (c *Client) CreateProtocolMapper(ctx context.Context, realm, scopeID string, req CreateProtocolMapperRequest) (*ProtocolMapper, error) {
	var out ProtocolMapper
	if err := c.do(ctx, "POST", scopeBase(realm)+"/"+url.PathEscape(scopeID)+"/protocol-mappers", req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UpdateProtocolMapper(ctx context.Context, realm, scopeID, mapperID string, req UpdateProtocolMapperRequest) (*ProtocolMapper, error) {
	var out ProtocolMapper
	if err := c.do(ctx, "PATCH", scopeMapperPath(realm, scopeID, mapperID), req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) DeleteProtocolMapper(ctx context.Context, realm, scopeID, mapperID string) error {
	if err := c.do(ctx, "DELETE", scopeMapperPath(realm, scopeID, mapperID), nil, nil); err != nil {
		return fmt.Errorf("deleting protocol mapper %q: %w", mapperID, err)
	}
	return nil
}

func (c *Client) ListClientAssignedScopes(ctx context.Context, realm, clientID string) ([]ClientScope, error) {
	var out []ClientScope
	path := fmt.Sprintf("/realms/%s/clients/%s/client-scopes", url.PathEscape(realm), url.PathEscape(clientID))
	if err := c.do(ctx, "GET", path, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetClientScopeAssignment(ctx context.Context, realm, clientID, scopeID string) (string, error) {
	scopes, err := c.ListClientAssignedScopes(ctx, realm, clientID)
	if err != nil {
		return "", err
	}
	for _, s := range scopes {
		if s.ID == scopeID {
			return s.DefaultScopeType, nil
		}
	}
	return "", ErrNotFound
}

func (c *Client) AssignClientScope(ctx context.Context, realm, clientID, scopeID, assignmentType string) (*ClientScopeMapping, error) {
	var out ClientScopeMapping
	if err := c.do(ctx, "PUT", scopeAssignmentPath(realm, clientID, scopeID, assignmentType), nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

func (c *Client) UnassignClientScope(ctx context.Context, realm, clientID, scopeID, assignmentType string) error {
	if err := c.do(ctx, "DELETE", scopeAssignmentPath(realm, clientID, scopeID, assignmentType), nil, nil); err != nil {
		return fmt.Errorf("unassigning client scope %q: %w", scopeID, err)
	}
	return nil
}
