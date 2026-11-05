package client

import (
	"context"
	"encoding/json"
	"fmt"
	"net/url"
)

type WebhookRetryPolicy struct {
	MaxAttempts     *int64 `json:"max_attempts"`
	BaseDelayMs     *int64 `json:"base_delay_ms"`
	MaxDelayMs      *int64 `json:"max_delay_ms"`
	MaxTotalDelayMs *int64 `json:"max_total_delay_ms"`
}

type WebhookSubscriber struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	WebhookID string `json:"webhook_id"`
}

type Webhook struct {
	ID          string              `json:"id"`
	Endpoint    string              `json:"endpoint"`
	Name        *string             `json:"name"`
	Description *string             `json:"description"`
	Subscribers []WebhookSubscriber `json:"subscribers"`
	RetryPolicy *WebhookRetryPolicy `json:"retry_policy"`
}

type CreateWebhookRequest struct {
	Endpoint    string              `json:"endpoint"`
	Name        *string             `json:"name,omitempty"`
	Description *string             `json:"description,omitempty"`
	Headers     map[string]string   `json:"headers,omitempty"`
	Subscribers []string            `json:"subscribers"`
	RetryPolicy *WebhookRetryPolicy `json:"retry_policy,omitempty"`
}

type UpdateWebhookRequest struct {
	Endpoint    string              `json:"endpoint"`
	Name        *string             `json:"name"`
	Description *string             `json:"description"`
	Headers     map[string]string   `json:"headers,omitempty"`
	Subscribers []string            `json:"subscribers"`
	RetryPolicy *WebhookRetryPolicy `json:"retry_policy"`
}

type webhookEnvelope struct {
	Data Webhook `json:"data"`
}

type rotateWebhookSecretResponse struct {
	Secret string `json:"secret"`
}

func webhookBase(realm string) string {
	return fmt.Sprintf("/realms/%s/webhooks", url.PathEscape(realm))
}

func (c *Client) GetWebhook(ctx context.Context, realm, id string) (*Webhook, error) {
	var w *Webhook
	if err := c.do(ctx, "GET", webhookBase(realm)+"/"+url.PathEscape(id), nil, &w); err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrNotFound
	}
	return w, nil
}

func (c *Client) CreateWebhook(ctx context.Context, realm string, req CreateWebhookRequest) (*Webhook, error) {
	var env webhookEnvelope
	if err := c.do(ctx, "POST", webhookBase(realm), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) UpdateWebhook(ctx context.Context, realm, id string, req UpdateWebhookRequest) (*Webhook, error) {
	var env webhookEnvelope
	if err := c.do(ctx, "PUT", webhookBase(realm)+"/"+url.PathEscape(id), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) DeleteWebhook(ctx context.Context, realm, id string) error {
	if err := c.do(ctx, "DELETE", webhookBase(realm)+"/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("deleting webhook %q: %w", id, err)
	}
	return nil
}

func (c *Client) RotateWebhookSecret(ctx context.Context, realm, id string) (string, error) {
	var out rotateWebhookSecretResponse
	if err := c.do(ctx, "POST", webhookBase(realm)+"/"+url.PathEscape(id)+"/secret/rotate", nil, &out); err != nil {
		return "", err
	}
	return out.Secret, nil
}

type EmailTemplate struct {
	ID        string          `json:"id"`
	RealmID   string          `json:"realm_id"`
	Name      string          `json:"name"`
	EmailType string          `json:"email_type"`
	Structure json.RawMessage `json:"structure"`
	MJML      string          `json:"mjml"`
}

type CreateEmailTemplateRequest struct {
	Name      string          `json:"name"`
	EmailType string          `json:"email_type"`
	Structure json.RawMessage `json:"structure"`
}

type UpdateEmailTemplateRequest struct {
	Name      string          `json:"name"`
	Structure json.RawMessage `json:"structure"`
}

type EmailTemplateVariable struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

type emailTemplateEnvelope struct {
	Data EmailTemplate `json:"data"`
}

type emailTemplateVariablesEnvelope struct {
	Data []EmailTemplateVariable `json:"data"`
}

func emailTemplateBase(realm string) string {
	return fmt.Sprintf("/realms/%s/email-templates", url.PathEscape(realm))
}

func (c *Client) GetEmailTemplate(ctx context.Context, realm, id string) (*EmailTemplate, error) {
	var env emailTemplateEnvelope
	if err := c.do(ctx, "GET", emailTemplateBase(realm)+"/"+url.PathEscape(id), nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) CreateEmailTemplate(ctx context.Context, realm string, req CreateEmailTemplateRequest) (*EmailTemplate, error) {
	var env emailTemplateEnvelope
	if err := c.do(ctx, "POST", emailTemplateBase(realm), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) UpdateEmailTemplate(ctx context.Context, realm, id string, req UpdateEmailTemplateRequest) (*EmailTemplate, error) {
	var env emailTemplateEnvelope
	if err := c.do(ctx, "PUT", emailTemplateBase(realm)+"/"+url.PathEscape(id), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) DeleteEmailTemplate(ctx context.Context, realm, id string) error {
	if err := c.do(ctx, "DELETE", emailTemplateBase(realm)+"/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("deleting email template %q: %w", id, err)
	}
	return nil
}

func (c *Client) GetEmailTemplateVariables(ctx context.Context, emailType string) ([]EmailTemplateVariable, error) {
	var env emailTemplateVariablesEnvelope
	if err := c.do(ctx, "GET", "/email-templates/variables/"+url.PathEscape(emailType), nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

type FederationProvider struct {
	ID                  string          `json:"id"`
	RealmID             string          `json:"realm_id"`
	Name                string          `json:"name"`
	ProviderType        string          `json:"provider_type"`
	Enabled             bool            `json:"enabled"`
	Priority            int64           `json:"priority"`
	Config              json.RawMessage `json:"config"`
	SyncEnabled         bool            `json:"sync_enabled"`
	SyncMode            string          `json:"sync_mode"`
	SyncIntervalMinutes *int64          `json:"sync_interval_minutes"`
}

type CreateFederationProviderRequest struct {
	Name                string          `json:"name"`
	ProviderType        string          `json:"provider_type"`
	Enabled             bool            `json:"enabled"`
	Priority            int64           `json:"priority"`
	Config              json.RawMessage `json:"config"`
	SyncEnabled         bool            `json:"sync_enabled"`
	SyncMode            string          `json:"sync_mode"`
	SyncIntervalMinutes *int64          `json:"sync_interval_minutes"`
}

type UpdateFederationProviderRequest struct {
	Name                string          `json:"name"`
	ProviderType        string          `json:"provider_type"`
	Enabled             bool            `json:"enabled"`
	Priority            int64           `json:"priority"`
	Config              json.RawMessage `json:"config"`
	SyncEnabled         bool            `json:"sync_enabled"`
	SyncMode            string          `json:"sync_mode"`
	SyncIntervalMinutes *int64          `json:"sync_interval_minutes"`
}

type federationProviderEnvelope struct {
	Data FederationProvider `json:"data"`
}

func federationProviderBase(realm string) string {
	return fmt.Sprintf("/realms/%s/federation/providers", url.PathEscape(realm))
}

func (c *Client) GetFederationProvider(ctx context.Context, realm, id string) (*FederationProvider, error) {
	var p FederationProvider
	if err := c.do(ctx, "GET", federationProviderBase(realm)+"/"+url.PathEscape(id), nil, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) CreateFederationProvider(ctx context.Context, realm string, req CreateFederationProviderRequest) (*FederationProvider, error) {
	var p FederationProvider
	if err := c.do(ctx, "POST", federationProviderBase(realm), req, &p); err != nil {
		return nil, err
	}
	return &p, nil
}

func (c *Client) UpdateFederationProvider(ctx context.Context, realm, id string, req UpdateFederationProviderRequest) (*FederationProvider, error) {
	var env federationProviderEnvelope
	if err := c.do(ctx, "PUT", federationProviderBase(realm)+"/"+url.PathEscape(id), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

func (c *Client) DeleteFederationProvider(ctx context.Context, realm, id string) error {
	if err := c.do(ctx, "DELETE", federationProviderBase(realm)+"/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("deleting federation provider %q: %w", id, err)
	}
	return nil
}

type UserAttributeEntry struct {
	ID     string `json:"id"`
	UserID string `json:"user_id"`
	Key    string `json:"key"`
	Value  string `json:"value"`
}

type setUserAttributesRequest struct {
	Attributes map[string]string `json:"attributes"`
}

type userAttributesEnvelope struct {
	Data []UserAttributeEntry `json:"data"`
}

func userAttributesBase(realm, userID string) string {
	return fmt.Sprintf("/realms/%s/users/%s/attributes", url.PathEscape(realm), url.PathEscape(userID))
}

func (c *Client) ListUserAttributes(ctx context.Context, realm, userID string) ([]UserAttributeEntry, error) {
	var env userAttributesEnvelope
	if err := c.do(ctx, "GET", userAttributesBase(realm, userID), nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

func (c *Client) SetUserAttributes(ctx context.Context, realm, userID string, attrs map[string]string) ([]UserAttributeEntry, error) {
	var env userAttributesEnvelope
	if err := c.do(ctx, "PUT", userAttributesBase(realm, userID), setUserAttributesRequest{Attributes: attrs}, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

func (c *Client) DeleteUserAttribute(ctx context.Context, realm, userID, key string) error {
	if err := c.do(ctx, "DELETE", userAttributesBase(realm, userID)+"/"+url.PathEscape(key), nil, nil); err != nil {
		return fmt.Errorf("deleting attribute %q of user %q: %w", key, userID, err)
	}
	return nil
}
