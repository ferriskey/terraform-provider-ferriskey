package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"time"
)

// OrgAttribute mirrors an organization or group key/value attribute. The
// organization variant carries organization_id, the group variant group_id.
type OrgAttribute struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id,omitempty"`
	GroupID        string    `json:"group_id,omitempty"`
	Key            string    `json:"key"`
	Value          string    `json:"value"`
	CreatedAt      time.Time `json:"created_at"`
}

// OrgAttributeRequest is the body of the attribute PUT endpoints
// (UpsertAttributeValidator).
type OrgAttributeRequest struct {
	Value string `json:"value"`
}

type orgAttributeListEnvelope struct {
	Data []OrgAttribute `json:"data"`
}

// OrgGroup mirrors the Group schema (a group inside an organization).
type OrgGroup struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	ParentGroupID  *string   `json:"parent_group_id"`
	Name           string    `json:"name"`
	Description    *string   `json:"description"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

// CreateOrgGroupRequest is the body of POST .../groups (CreateGroupValidator).
type CreateOrgGroupRequest struct {
	Name          string  `json:"name"`
	Description   *string `json:"description,omitempty"`
	ParentGroupID *string `json:"parent_group_id,omitempty"`
}

// UpdateOrgGroupRequest is the body of PUT .../groups/{id}
// (UpdateGroupValidator). Absent fields are left unchanged; an empty
// description clears it.
type UpdateOrgGroupRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

type orgGroupListEnvelope struct {
	Data     []OrgGroup  `json:"data"`
	Metadata orgPageMeta `json:"metadata"`
}

type orgPageMeta struct {
	TotalPages int `json:"total_pages"`
}

// OrgGroupMember mirrors a group membership row (list variant, enriched with
// user identity fields that POST does not return).
type OrgGroupMember struct {
	ID        string    `json:"id"`
	GroupID   string    `json:"group_id"`
	UserID    string    `json:"user_id"`
	Username  string    `json:"username,omitempty"`
	CreatedAt time.Time `json:"created_at"`
}

// OrgGroupMemberRequest is the body of POST .../groups/{id}/members
// (AddGroupMemberValidator).
type OrgGroupMemberRequest struct {
	UserID string `json:"user_id"`
}

type orgGroupMemberListEnvelope struct {
	Data     []OrgGroupMember `json:"data"`
	Metadata orgPageMeta      `json:"metadata"`
}

// OrgRoleRequest is the body of the role assignment POST endpoints
// (AssignMemberRoleValidator / AssignGroupRoleValidator).
type OrgRoleRequest struct {
	RoleID string `json:"role_id"`
}

const orgExtPageLimit = 100

func orgExtBase(realm, orgID string) string {
	return orgBase(realm) + "/" + url.PathEscape(orgID)
}

func orgExtGroupBase(realm, orgID, groupID string) string {
	return orgExtBase(realm, orgID) + "/groups/" + url.PathEscape(groupID)
}

func orgExtPage(path string, page int) string {
	return path + "?limit=" + strconv.Itoa(orgExtPageLimit) + "&page=" + strconv.Itoa(page)
}

// ListOrganizationAttributes returns all attributes of an organization. The
// response is a { "data": [...] } envelope.
func (c *Client) ListOrganizationAttributes(ctx context.Context, realm, orgID string) ([]OrgAttribute, error) {
	var env orgAttributeListEnvelope
	if err := c.do(ctx, "GET", orgExtBase(realm, orgID)+"/attributes", nil, &env); err != nil {
		return nil, err
	}
	return env.Data, nil
}

// UpsertOrganizationAttribute sets an organization attribute. The 200 response
// is the bare attribute.
func (c *Client) UpsertOrganizationAttribute(ctx context.Context, realm, orgID, key, value string) (*OrgAttribute, error) {
	var attr OrgAttribute
	path := orgExtBase(realm, orgID) + "/attributes/" + url.PathEscape(key)
	if err := c.do(ctx, "PUT", path, OrgAttributeRequest{Value: value}, &attr); err != nil {
		return nil, err
	}
	return &attr, nil
}

// DeleteOrganizationAttribute deletes an organization attribute (204, also for
// an absent key).
func (c *Client) DeleteOrganizationAttribute(ctx context.Context, realm, orgID, key string) error {
	path := orgExtBase(realm, orgID) + "/attributes/" + url.PathEscape(key)
	if err := c.do(ctx, "DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("deleting organization attribute %q: %w", key, err)
	}
	return nil
}

// ListOrganizationMemberRoles returns the roles assigned to an organization
// member. The response is a bare array of Role.
func (c *Client) ListOrganizationMemberRoles(ctx context.Context, realm, orgID, userID string) ([]Role, error) {
	var roles []Role
	path := orgExtBase(realm, orgID) + "/members/" + url.PathEscape(userID) + "/roles"
	if err := c.do(ctx, "GET", path, nil, &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

// AssignOrganizationMemberRole assigns a role to an organization member. The
// user must already be a member (404 otherwise); assigning twice is a 500.
func (c *Client) AssignOrganizationMemberRole(ctx context.Context, realm, orgID, userID, roleID string) error {
	path := orgExtBase(realm, orgID) + "/members/" + url.PathEscape(userID) + "/roles"
	if err := c.do(ctx, "POST", path, OrgRoleRequest{RoleID: roleID}, nil); err != nil {
		return fmt.Errorf("assigning role %q to organization member %q: %w", roleID, userID, err)
	}
	return nil
}

// RemoveOrganizationMemberRole removes a role from an organization member.
func (c *Client) RemoveOrganizationMemberRole(ctx context.Context, realm, orgID, userID, roleID string) error {
	path := orgExtBase(realm, orgID) + "/members/" + url.PathEscape(userID) + "/roles/" + url.PathEscape(roleID)
	if err := c.do(ctx, "DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("removing role %q from organization member %q: %w", roleID, userID, err)
	}
	return nil
}

// GetOrganizationGroup fetches a group by UUID. The response is the bare
// group.
func (c *Client) GetOrganizationGroup(ctx context.Context, realm, orgID, groupID string) (*OrgGroup, error) {
	var g OrgGroup
	if err := c.do(ctx, "GET", orgExtGroupBase(realm, orgID, groupID), nil, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

// CreateOrganizationGroup creates a group. The 201 response is the bare group.
func (c *Client) CreateOrganizationGroup(ctx context.Context, realm, orgID string, req CreateOrgGroupRequest) (*OrgGroup, error) {
	var g OrgGroup
	if err := c.do(ctx, "POST", orgExtBase(realm, orgID)+"/groups", req, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

// UpdateOrganizationGroup updates a group. The 200 response is the bare group.
func (c *Client) UpdateOrganizationGroup(ctx context.Context, realm, orgID, groupID string, req UpdateOrgGroupRequest) (*OrgGroup, error) {
	var g OrgGroup
	if err := c.do(ctx, "PUT", orgExtGroupBase(realm, orgID, groupID), req, &g); err != nil {
		return nil, err
	}
	return &g, nil
}

// DeleteOrganizationGroup deletes a group.
func (c *Client) DeleteOrganizationGroup(ctx context.Context, realm, orgID, groupID string) error {
	if err := c.do(ctx, "DELETE", orgExtGroupBase(realm, orgID, groupID), nil, nil); err != nil {
		return fmt.Errorf("deleting organization group %q: %w", groupID, err)
	}
	return nil
}

// ListOrganizationGroups returns every group of an organization, walking the
// paginated { "data": [...], "metadata": {...} } envelope.
func (c *Client) ListOrganizationGroups(ctx context.Context, realm, orgID string) ([]OrgGroup, error) {
	var all []OrgGroup
	base := orgExtBase(realm, orgID) + "/groups"
	for page := 1; ; page++ {
		var env orgGroupListEnvelope
		if err := c.do(ctx, "GET", orgExtPage(base, page), nil, &env); err != nil {
			return nil, err
		}
		all = append(all, env.Data...)
		if page >= env.Metadata.TotalPages {
			return all, nil
		}
	}
}

// ListOrganizationGroupAttributes returns the attributes of a group. The
// response is a bare array.
func (c *Client) ListOrganizationGroupAttributes(ctx context.Context, realm, orgID, groupID string) ([]OrgAttribute, error) {
	var attrs []OrgAttribute
	if err := c.do(ctx, "GET", orgExtGroupBase(realm, orgID, groupID)+"/attributes", nil, &attrs); err != nil {
		return nil, err
	}
	return attrs, nil
}

// UpsertOrganizationGroupAttribute sets a group attribute. The 200 response is
// the bare attribute.
func (c *Client) UpsertOrganizationGroupAttribute(ctx context.Context, realm, orgID, groupID, key, value string) (*OrgAttribute, error) {
	var attr OrgAttribute
	path := orgExtGroupBase(realm, orgID, groupID) + "/attributes/" + url.PathEscape(key)
	if err := c.do(ctx, "PUT", path, OrgAttributeRequest{Value: value}, &attr); err != nil {
		return nil, err
	}
	return &attr, nil
}

// DeleteOrganizationGroupAttribute deletes a group attribute (204, also for an
// absent key).
func (c *Client) DeleteOrganizationGroupAttribute(ctx context.Context, realm, orgID, groupID, key string) error {
	path := orgExtGroupBase(realm, orgID, groupID) + "/attributes/" + url.PathEscape(key)
	if err := c.do(ctx, "DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("deleting organization group attribute %q: %w", key, err)
	}
	return nil
}

// ListOrganizationGroupMembers returns every member of a group, walking the
// paginated envelope.
func (c *Client) ListOrganizationGroupMembers(ctx context.Context, realm, orgID, groupID string) ([]OrgGroupMember, error) {
	var all []OrgGroupMember
	base := orgExtGroupBase(realm, orgID, groupID) + "/members"
	for page := 1; ; page++ {
		var env orgGroupMemberListEnvelope
		if err := c.do(ctx, "GET", orgExtPage(base, page), nil, &env); err != nil {
			return nil, err
		}
		all = append(all, env.Data...)
		if page >= env.Metadata.TotalPages {
			return all, nil
		}
	}
}

// AddOrganizationGroupMember adds a user to a group (400 already_exists if
// present).
func (c *Client) AddOrganizationGroupMember(ctx context.Context, realm, orgID, groupID, userID string) error {
	path := orgExtGroupBase(realm, orgID, groupID) + "/members"
	if err := c.do(ctx, "POST", path, OrgGroupMemberRequest{UserID: userID}, nil); err != nil {
		return fmt.Errorf("adding user %q to organization group %q: %w", userID, groupID, err)
	}
	return nil
}

// RemoveOrganizationGroupMember removes a user from a group (404 if absent).
func (c *Client) RemoveOrganizationGroupMember(ctx context.Context, realm, orgID, groupID, userID string) error {
	path := orgExtGroupBase(realm, orgID, groupID) + "/members/" + url.PathEscape(userID)
	if err := c.do(ctx, "DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("removing user %q from organization group %q: %w", userID, groupID, err)
	}
	return nil
}

// ListOrganizationGroupRoles returns the roles assigned to a group. The
// response is a bare array of Role.
func (c *Client) ListOrganizationGroupRoles(ctx context.Context, realm, orgID, groupID string) ([]Role, error) {
	var roles []Role
	if err := c.do(ctx, "GET", orgExtGroupBase(realm, orgID, groupID)+"/roles", nil, &roles); err != nil {
		return nil, err
	}
	return roles, nil
}

// AssignOrganizationGroupRole assigns a role to a group (assigning twice is a
// 500).
func (c *Client) AssignOrganizationGroupRole(ctx context.Context, realm, orgID, groupID, roleID string) error {
	path := orgExtGroupBase(realm, orgID, groupID) + "/roles"
	if err := c.do(ctx, "POST", path, OrgRoleRequest{RoleID: roleID}, nil); err != nil {
		return fmt.Errorf("assigning role %q to organization group %q: %w", roleID, groupID, err)
	}
	return nil
}

// RemoveOrganizationGroupRole removes a role from a group.
func (c *Client) RemoveOrganizationGroupRole(ctx context.Context, realm, orgID, groupID, roleID string) error {
	path := orgExtGroupBase(realm, orgID, groupID) + "/roles/" + url.PathEscape(roleID)
	if err := c.do(ctx, "DELETE", path, nil, nil); err != nil {
		return fmt.Errorf("removing role %q from organization group %q: %w", roleID, groupID, err)
	}
	return nil
}
