package client

import (
	"context"
	"fmt"
	"net/url"
	"time"
)

type ClientSamlConfig struct {
	ClientID                string    `json:"client_id"`
	RealmID                 string    `json:"realm_id"`
	SpEntityID              string    `json:"sp_entity_id"`
	AcsURL                  string    `json:"acs_url"`
	NameIDFormat            string    `json:"name_id_format"`
	SignAssertions          bool      `json:"sign_assertions"`
	SignDocuments           bool      `json:"sign_documents"`
	WantAuthnRequestsSigned bool      `json:"want_authn_requests_signed"`
	CreatedAt               time.Time `json:"created_at"`
	UpdatedAt               time.Time `json:"updated_at"`
}

type SetClientSamlConfigRequest struct {
	SpEntityID              string `json:"sp_entity_id"`
	AcsURL                  string `json:"acs_url"`
	NameIDFormat            string `json:"name_id_format"`
	SignAssertions          bool   `json:"sign_assertions"`
	SignDocuments           bool   `json:"sign_documents"`
	WantAuthnRequestsSigned bool   `json:"want_authn_requests_signed"`
}

type SamlAttributeMapper struct {
	ID         string    `json:"id"`
	ClientID   string    `json:"client_id"`
	Name       string    `json:"name"`
	NameFormat string    `json:"name_format"`
	Source     string    `json:"source"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type CreateSamlAttributeMapperRequest struct {
	Name       string `json:"name"`
	NameFormat string `json:"name_format,omitempty"`
	Source     string `json:"source"`
}

type ToggleClientMaintenanceRequest struct {
	Enabled         bool    `json:"enabled"`
	Reason          *string `json:"reason,omitempty"`
	SessionStrategy *string `json:"session_strategy,omitempty"`
}

type ClientMaintenanceState struct {
	Enabled         bool    `json:"maintenance_enabled"`
	Reason          *string `json:"maintenance_reason"`
	SessionStrategy string  `json:"maintenance_session_strategy"`
}

type MaintenanceWhitelistEntry struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	UserID    *string   `json:"user_id"`
	RoleID    *string   `json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

type RealmMaintenanceWhitelistEntry struct {
	ID        string    `json:"id"`
	RealmID   string    `json:"realm_id"`
	UserID    *string   `json:"user_id"`
	RoleID    *string   `json:"role_id"`
	CreatedAt time.Time `json:"created_at"`
}

type AddMaintenanceWhitelistEntryRequest struct {
	UserID *string `json:"user_id,omitempty"`
	RoleID *string `json:"role_id,omitempty"`
}

type samlmClientWhitelistList struct {
	Data []MaintenanceWhitelistEntry `json:"data"`
}

type samlmClientWhitelistOne struct {
	Data MaintenanceWhitelistEntry `json:"data"`
}

type samlmRealmWhitelistList struct {
	Data []RealmMaintenanceWhitelistEntry `json:"data"`
}

type samlmRealmWhitelistOne struct {
	Data RealmMaintenanceWhitelistEntry `json:"data"`
}

type samlmClientEnvelope struct {
	Data ClientMaintenanceState `json:"data"`
}

func samlmClientPath(realm, clientUUID string) string {
	return fmt.Sprintf("/realms/%s/clients/%s", url.PathEscape(realm), url.PathEscape(clientUUID))
}

func samlmRealmWhitelistPath(realm string) string {
	return fmt.Sprintf("/realms/%s/settings/maintenance/whitelist", url.PathEscape(realm))
}

func (c *Client) GetClientSamlConfig(ctx context.Context, realm, clientUUID string) (*ClientSamlConfig, error) {
	var cfg ClientSamlConfig
	if err := c.do(ctx, "GET", samlmClientPath(realm, clientUUID)+"/saml-config", nil, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Client) SetClientSamlConfig(ctx context.Context, realm, clientUUID string, req SetClientSamlConfigRequest) (*ClientSamlConfig, error) {
	var cfg ClientSamlConfig
	if err := c.do(ctx, "PUT", samlmClientPath(realm, clientUUID)+"/saml-config", req, &cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (c *Client) ListSamlAttributeMappers(ctx context.Context, realm, clientUUID string) ([]SamlAttributeMapper, error) {
	var mappers []SamlAttributeMapper
	if err := c.do(ctx, "GET", samlmClientPath(realm, clientUUID)+"/saml-attribute-mappers", nil, &mappers); err != nil {
		return nil, err
	}
	return mappers, nil
}

func (c *Client) CreateSamlAttributeMapper(ctx context.Context, realm, clientUUID string, req CreateSamlAttributeMapperRequest) (*SamlAttributeMapper, error) {
	var m SamlAttributeMapper
	if err := c.do(ctx, "POST", samlmClientPath(realm, clientUUID)+"/saml-attribute-mappers", req, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (c *Client) DeleteSamlAttributeMapper(ctx context.Context, realm, clientUUID, mapperID string) error {
	path := samlmClientPath(realm, clientUUID) + "/saml-attribute-mappers/" + url.PathEscape(mapperID)
	if err := c.do(ctx, "DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("deleting SAML attribute mapper %q: %w", mapperID, err)
	}
	return nil
}

func (c *Client) ToggleClientMaintenance(ctx context.Context, realm, clientUUID string, req ToggleClientMaintenanceRequest) error {
	return c.do(ctx, "PUT", samlmClientPath(realm, clientUUID)+"/maintenance", req, nil)
}

func (c *Client) GetClientMaintenance(ctx context.Context, realm, clientUUID string) (*ClientMaintenanceState, error) {
	var env samlmClientEnvelope
	if err := c.do(ctx, "GET", samlmClientPath(realm, clientUUID), nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) ListClientMaintenanceWhitelist(ctx context.Context, realm, clientUUID string) ([]MaintenanceWhitelistEntry, error) {
	var env samlmClientWhitelistList
	if err := c.do(ctx, "GET", samlmClientPath(realm, clientUUID)+"/maintenance/whitelist", nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

func (c *Client) AddClientMaintenanceWhitelistEntry(ctx context.Context, realm, clientUUID string, req AddMaintenanceWhitelistEntryRequest) (*MaintenanceWhitelistEntry, error) {
	var env samlmClientWhitelistOne
	if err := c.do(ctx, "POST", samlmClientPath(realm, clientUUID)+"/maintenance/whitelist", req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) RemoveClientMaintenanceWhitelistEntry(ctx context.Context, realm, clientUUID, entryID string) error {
	path := samlmClientPath(realm, clientUUID) + "/maintenance/whitelist/" + url.PathEscape(entryID)
	if err := c.do(ctx, "DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("removing client maintenance whitelist entry %q: %w", entryID, err)
	}
	return nil
}

func (c *Client) ListRealmMaintenanceWhitelist(ctx context.Context, realm string) ([]RealmMaintenanceWhitelistEntry, error) {
	var env samlmRealmWhitelistList
	if err := c.do(ctx, "GET", samlmRealmWhitelistPath(realm), nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

func (c *Client) AddRealmMaintenanceWhitelistEntry(ctx context.Context, realm string, req AddMaintenanceWhitelistEntryRequest) (*RealmMaintenanceWhitelistEntry, error) {
	var env samlmRealmWhitelistOne
	if err := c.do(ctx, "POST", samlmRealmWhitelistPath(realm), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) RemoveRealmMaintenanceWhitelistEntry(ctx context.Context, realm, entryID string) error {
	path := samlmRealmWhitelistPath(realm) + "/" + url.PathEscape(entryID)
	if err := c.do(ctx, "DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("removing realm maintenance whitelist entry %q: %w", entryID, err)
	}
	return nil
}
