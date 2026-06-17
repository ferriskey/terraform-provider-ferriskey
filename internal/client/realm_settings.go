package client

import (
	"context"
	"net/url"
)

// UpdateRealmSettings applies a partial update to a realm's settings via
// PUT /realms/{name}/settings. Only the non-nil fields of req are changed. The
// response is a { "data": Realm } envelope; the updated settings are returned.
func (c *Client) UpdateRealmSettings(ctx context.Context, realm string, req UpdateRealmSettingsRequest) (*RealmSetting, error) {
	var env realmEnvelope
	if err := c.do(ctx, "PUT", "/realms/"+url.PathEscape(realm)+"/settings", req, &env); err != nil {
		return nil, err
	}
	return env.Data.Settings, nil
}

// GetRealmSettings reads a realm's current settings (via GetRealm, since the
// realm payload embeds them). Returns ErrNotFound if the realm is absent.
func (c *Client) GetRealmSettings(ctx context.Context, realm string) (*RealmSetting, error) {
	r, err := c.GetRealm(ctx, realm)
	if err != nil {
		return nil, err
	}
	return r.Settings, nil
}
