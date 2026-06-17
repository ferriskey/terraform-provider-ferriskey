package client

import (
	"context"
	"fmt"
	"net/url"
)

func clientBase(realm string) string {
	return fmt.Sprintf("/realms/%s/clients", url.PathEscape(realm))
}

// GetClient fetches a client by UUID within a realm. Returns ErrNotFound if
// absent.
func (c *Client) GetClient(ctx context.Context, realm, id string) (*APIClient, error) {
	var env clientEnvelope
	if err := c.do(ctx, "GET", clientBase(realm)+"/"+url.PathEscape(id), nil, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// CreateClient creates a client. The 201 response is the bare Client object.
func (c *Client) CreateClient(ctx context.Context, realm string, req CreateClientRequest) (*APIClient, error) {
	var created APIClient
	if err := c.do(ctx, "POST", clientBase(realm), req, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// UpdateClient patches a client. The response is an { "data": Client }
// envelope.
func (c *Client) UpdateClient(ctx context.Context, realm, id string, req UpdateClientRequest) (*APIClient, error) {
	var env clientEnvelope
	if err := c.do(ctx, "PATCH", clientBase(realm)+"/"+url.PathEscape(id), req, &env); err != nil {
		return nil, err
	}
	return &env.Data, nil
}

// DeleteClient deletes a client by UUID.
func (c *Client) DeleteClient(ctx context.Context, realm, id string) error {
	if err := c.do(ctx, "DELETE", clientBase(realm)+"/"+url.PathEscape(id), nil, nil); err != nil {
		return fmt.Errorf("deleting client %q: %w", id, err)
	}
	return nil
}

// ---------------------------------------------------------------------------
// Redirect URIs (managed via dedicated sub-endpoints)
// ---------------------------------------------------------------------------

// ListRedirectURIs returns the redirect URIs configured for a client.
func (c *Client) ListRedirectURIs(ctx context.Context, realm, clientID string) ([]RedirectURI, error) {
	var uris []RedirectURI
	path := clientBase(realm) + "/" + url.PathEscape(clientID) + "/redirects"
	if err := c.do(ctx, "GET", path, nil, &uris); err != nil {
		return nil, err
	}
	return uris, nil
}

// CreateRedirectURI adds a redirect URI to a client.
func (c *Client) CreateRedirectURI(ctx context.Context, realm, clientID, value string, enabled bool) (*RedirectURI, error) {
	var created RedirectURI
	path := clientBase(realm) + "/" + url.PathEscape(clientID) + "/redirects"
	req := CreateRedirectURIRequest{Value: value, Enabled: enabled}
	if err := c.do(ctx, "POST", path, req, &created); err != nil {
		return nil, err
	}
	return &created, nil
}

// DeleteRedirectURI removes a redirect URI from a client by its UUID.
func (c *Client) DeleteRedirectURI(ctx context.Context, realm, clientID, uriID string) error {
	path := clientBase(realm) + "/" + url.PathEscape(clientID) + "/redirects/" + url.PathEscape(uriID)
	if err := c.do(ctx, "DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("deleting redirect URI %q: %w", uriID, err)
	}
	return nil
}

// SetRedirectURIs reconciles a client's redirect URIs to exactly the desired
// set of values: it adds the missing ones and deletes the extras. Order is not
// significant (the provider models redirect URIs as a Set), so only set
// membership is reconciled. It returns the resulting list.
func (c *Client) SetRedirectURIs(ctx context.Context, realm, clientID string, desired []string) ([]RedirectURI, error) {
	current, err := c.ListRedirectURIs(ctx, realm, clientID)
	if err != nil {
		return nil, err
	}

	want := make(map[string]struct{}, len(desired))
	for _, v := range desired {
		want[v] = struct{}{}
	}
	have := make(map[string]RedirectURI, len(current))
	for _, u := range current {
		have[u.Value] = u
	}

	// Delete URIs that are no longer desired.
	for value, u := range have {
		if _, ok := want[value]; !ok {
			if err := c.DeleteRedirectURI(ctx, realm, clientID, u.ID); err != nil {
				return nil, err
			}
		}
	}
	// Add URIs that are missing.
	for _, value := range desired {
		if _, ok := have[value]; !ok {
			if _, err := c.CreateRedirectURI(ctx, realm, clientID, value, true); err != nil {
				return nil, err
			}
		}
	}

	return c.ListRedirectURIs(ctx, realm, clientID)
}
