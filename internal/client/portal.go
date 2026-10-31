package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type PortalLayout struct {
	ID        string          `json:"id"`
	Name      string          `json:"name"`
	Tree      json.RawMessage `json:"tree"`
	IsDefault bool            `json:"is_default"`
}

type PortalLayoutRequest struct {
	Name string          `json:"name"`
	Tree json.RawMessage `json:"tree"`
}

type PortalTheme struct {
	ID       string          `json:"id"`
	Name     string          `json:"name"`
	LayoutID *string         `json:"layout_id"`
	Config   json.RawMessage `json:"config"`
	Pages    json.RawMessage `json:"pages"`
}

type PortalThemeRequest struct {
	Name     string          `json:"name"`
	LayoutID *string         `json:"layout_id"`
	Config   json.RawMessage `json:"config,omitempty"`
}

type portalLayoutEnvelope struct {
	Data PortalLayout `json:"data"`
}

type portalLayoutList struct {
	Data []PortalLayout `json:"data"`
}

type portalThemeEnvelope struct {
	Data PortalTheme `json:"data"`
}

type portalActiveTheme struct {
	ThemeID *string `json:"theme_id"`
}

func portalLayoutsBase(realm string) string {
	return fmt.Sprintf("/realms/%s/portal-layouts", url.PathEscape(realm))
}

func portalThemesBase(realm string) string {
	return fmt.Sprintf("/realms/%s/portal/themes", url.PathEscape(realm))
}

func (c *Client) CreatePortalLayout(ctx context.Context, realm string, req PortalLayoutRequest) (*PortalLayout, error) {
	var env portalLayoutEnvelope
	if err := c.do(ctx, "POST", portalLayoutsBase(realm), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) GetPortalLayout(ctx context.Context, realm, id string) (*PortalLayout, error) {
	var env portalLayoutEnvelope
	if err := c.do(ctx, "GET", portalLayoutsBase(realm)+"/"+url.PathEscape(id), nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) UpdatePortalLayout(ctx context.Context, realm, id string, req PortalLayoutRequest) (*PortalLayout, error) {
	var env portalLayoutEnvelope
	if err := c.do(ctx, "PUT", portalLayoutsBase(realm)+"/"+url.PathEscape(id), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) DeletePortalLayout(ctx context.Context, realm, id string) error {
	return c.do(ctx, "DELETE", portalLayoutsBase(realm)+"/"+url.PathEscape(id), nil, nil)
}

func (c *Client) SetDefaultPortalLayout(ctx context.Context, realm, id string) (*PortalLayout, error) {
	var env portalLayoutEnvelope
	if err := c.do(ctx, "PUT", portalLayoutsBase(realm)+"/"+url.PathEscape(id)+"/default", nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) GetDefaultPortalLayout(ctx context.Context, realm string) (*PortalLayout, error) {
	var list portalLayoutList
	if err := c.do(ctx, "GET", portalLayoutsBase(realm)+"?is_default=true&limit=1", nil, &list); err != nil {
		return nil, err
	}
	if len(list.Data) == 0 {
		return nil, ErrNotFound
	}
	return &list.Data[0], nil
}

func (c *Client) CreatePortalTheme(ctx context.Context, realm string, req PortalThemeRequest) (*PortalTheme, error) {
	var env portalThemeEnvelope
	if err := c.do(ctx, "POST", portalThemesBase(realm), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) GetPortalTheme(ctx context.Context, realm, id string) (*PortalTheme, error) {
	var env portalThemeEnvelope
	if err := c.do(ctx, "GET", portalThemesBase(realm)+"/"+url.PathEscape(id), nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) UpdatePortalTheme(ctx context.Context, realm, id string, req PortalThemeRequest) (*PortalTheme, error) {
	var env portalThemeEnvelope
	if err := c.do(ctx, "PUT", portalThemesBase(realm)+"/"+url.PathEscape(id), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) DeletePortalTheme(ctx context.Context, realm, id string) error {
	return c.do(ctx, "DELETE", portalThemesBase(realm)+"/"+url.PathEscape(id), nil, nil)
}

func (c *Client) UpdatePortalThemePage(ctx context.Context, realm, id, pageType string, tree json.RawMessage) (*PortalTheme, error) {
	var env portalThemeEnvelope
	path := portalThemesBase(realm) + "/" + url.PathEscape(id) + "/pages/" + url.PathEscape(pageType)
	body := struct {
		Tree json.RawMessage `json:"tree"`
	}{Tree: tree}
	if err := c.do(ctx, "PUT", path, body, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) ActivatePortalTheme(ctx context.Context, realm, id string) error {
	return c.do(ctx, "POST", portalThemesBase(realm)+"/"+url.PathEscape(id)+"/activate", nil, nil)
}

func (c *Client) GetActivePortalThemeID(ctx context.Context, realm string) (string, error) {
	var active portalActiveTheme
	path := fmt.Sprintf("/realms/%s/portal/active?page_type=login", url.PathEscape(realm))
	if err := c.do(ctx, "GET", path, nil, &active); err != nil {
		return "", err
	}
	if active.ThemeID == nil {
		return "", nil
	}
	return *active.ThemeID, nil
}
