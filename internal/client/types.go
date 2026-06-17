package client

import "time"

// This file mirrors the FerrisKey REST API DTOs (OpenAPI 3.1, FerrisKey API
// v0.6.x). Only the fields the provider reads or writes are modelled. Pointer
// fields map to nullable JSON fields so the zero value ("absent") is
// distinguishable from an explicit empty value.

// ---------------------------------------------------------------------------
// OAuth2 / OIDC token endpoint
// ---------------------------------------------------------------------------

// JwtToken is the response of the OIDC token endpoint.
//
// NOTE: although the OpenAPI document declares the token endpoint request body
// as application/json, the server actually expects
// application/x-www-form-urlencoded (the utoipa annotation does not match the
// Axum Form extractor). The client encodes requests as form data accordingly.
type JwtToken struct {
	AccessToken      string `json:"access_token"`
	TokenType        string `json:"token_type"`
	RefreshToken     string `json:"refresh_token"`
	ExpiresIn        int    `json:"expires_in"`
	RefreshExpiresIn int    `json:"refresh_expires_in"`
	IDToken          string `json:"id_token,omitempty"`
	SessionState     string `json:"session_state,omitempty"`
}

// APIError is the error envelope returned by the FerrisKey API. The API uses
// two shapes: a general error ({code,status,message}) and a validation error
// ({errors:[{field,message}]}, typically with HTTP 422). Both are captured.
type APIError struct {
	Code    string          `json:"code"`
	Status  int             `json:"status"`
	Message string          `json:"message"`
	Errors  []APIFieldError `json:"errors"`
}

// APIFieldError is a single field-level validation error.
type APIFieldError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

// ---------------------------------------------------------------------------
// Realm
// ---------------------------------------------------------------------------

// Realm mirrors the Realm schema.
type Realm struct {
	ID        string        `json:"id"`
	Name      string        `json:"name"`
	Settings  *RealmSetting `json:"settings,omitempty"`
	CreatedAt time.Time     `json:"created_at"`
	UpdatedAt time.Time     `json:"updated_at"`
}

// RealmSetting mirrors the RealmSetting schema (server-managed; read-only in
// the provider).
type RealmSetting struct {
	ID                       string    `json:"id"`
	RealmID                  string    `json:"realm_id"`
	DefaultSigningAlgorithm  *string   `json:"default_signing_algorithm,omitempty"`
	UserRegistrationEnabled  bool      `json:"user_registration_enabled"`
	ForgotPasswordEnabled    bool      `json:"forgot_password_enabled"`
	RememberMeEnabled        bool      `json:"remember_me_enabled"`
	MagicLinkEnabled         bool      `json:"magic_link_enabled"`
	MagicLinkTTL             int64     `json:"magic_link_ttl"`
	PasskeyEnabled           bool      `json:"passkey_enabled"`
	CompassEnabled           bool      `json:"compass_enabled"`
	AccessTokenLifetime      int64     `json:"access_token_lifetime"`
	RefreshTokenLifetime     int64     `json:"refresh_token_lifetime"`
	IDTokenLifetime          int64     `json:"id_token_lifetime"`
	TemporaryTokenLifetime   int64     `json:"temporary_token_lifetime"`
	EmailVerificationEnabled bool      `json:"email_verification_enabled"`
	EmailVerificationTTL     int64     `json:"email_verification_ttl_hours"`
	UpdatedAt                time.Time `json:"updated_at"`
}

// CreateRealmRequest is the body of POST /realms (CreateRealmValidator).
type CreateRealmRequest struct {
	Name string `json:"name"`
}

// UpdateRealmRequest is the body of PUT /realms/{name} (UpdateRealmValidator).
type UpdateRealmRequest struct {
	Name string `json:"name"`
}

// UpdateRealmSettingsRequest is the body of PUT /realms/{name}/settings
// (UpdateRealmSettingValidator). It is a partial merge: only non-nil fields are
// changed server-side. The response is a { "data": Realm } envelope.
type UpdateRealmSettingsRequest struct {
	AccessTokenLifetime       *int64  `json:"access_token_lifetime,omitempty"`
	RefreshTokenLifetime      *int64  `json:"refresh_token_lifetime,omitempty"`
	IDTokenLifetime           *int64  `json:"id_token_lifetime,omitempty"`
	TemporaryTokenLifetime    *int64  `json:"temporary_token_lifetime,omitempty"`
	UserRegistrationEnabled   *bool   `json:"user_registration_enabled,omitempty"`
	ForgotPasswordEnabled     *bool   `json:"forgot_password_enabled,omitempty"`
	RememberMeEnabled         *bool   `json:"remember_me_enabled,omitempty"`
	MagicLinkEnabled          *bool   `json:"magic_link_enabled,omitempty"`
	MagicLinkTTL              *int32  `json:"magic_link_ttl,omitempty"`
	PasskeyEnabled            *bool   `json:"passkey_enabled,omitempty"`
	CompassEnabled            *bool   `json:"compass_enabled,omitempty"`
	EmailVerificationEnabled  *bool   `json:"email_verification_enabled,omitempty"`
	EmailVerificationTTLHours *int64  `json:"email_verification_ttl_hours,omitempty"`
	DefaultSigningAlgorithm   *string `json:"default_signing_algorithm,omitempty"`
}

// dataEnvelope is the common { "data": ... } wrapper used by many endpoints.
type realmEnvelope struct {
	Data Realm `json:"data"`
}

// ---------------------------------------------------------------------------
// Client
// ---------------------------------------------------------------------------

// Client mirrors the Client schema.
type APIClient struct {
	ID                        string        `json:"id"`
	RealmID                   string        `json:"realm_id"`
	ClientID                  string        `json:"client_id"`
	Name                      string        `json:"name"`
	ClientType                string        `json:"client_type"`
	Protocol                  string        `json:"protocol"`
	Enabled                   bool          `json:"enabled"`
	PublicClient              bool          `json:"public_client"`
	ServiceAccountEnabled     bool          `json:"service_account_enabled"`
	DirectAccessGrantsEnabled bool          `json:"direct_access_grants_enabled"`
	Secret                    *string       `json:"secret,omitempty"`
	AccessTokenLifetime       *int64        `json:"access_token_lifetime,omitempty"`
	RefreshTokenLifetime      *int64        `json:"refresh_token_lifetime,omitempty"`
	IDTokenLifetime           *int64        `json:"id_token_lifetime,omitempty"`
	TemporaryTokenLifetime    *int64        `json:"temporary_token_lifetime,omitempty"`
	RedirectURIs              []RedirectURI `json:"redirect_uris,omitempty"`
	CreatedAt                 time.Time     `json:"created_at"`
	UpdatedAt                 time.Time     `json:"updated_at"`
}

// RedirectURI mirrors the RedirectUri schema. Redirect URIs are managed through
// dedicated sub-endpoints rather than the client create/update bodies.
type RedirectURI struct {
	ID        string    `json:"id"`
	ClientID  string    `json:"client_id"`
	Value     string    `json:"value"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// CreateClientRequest is the body of POST /realms/{realm}/clients
// (CreateClientValidator).
type CreateClientRequest struct {
	ClientType                string `json:"client_type"`
	ClientID                  string `json:"client_id,omitempty"`
	Name                      string `json:"name,omitempty"`
	Protocol                  string `json:"protocol,omitempty"`
	Enabled                   *bool  `json:"enabled,omitempty"`
	PublicClient              *bool  `json:"public_client,omitempty"`
	ServiceAccountEnabled     *bool  `json:"service_account_enabled,omitempty"`
	DirectAccessGrantsEnabled *bool  `json:"direct_access_grants_enabled,omitempty"`
}

// UpdateClientRequest is the body of PATCH /realms/{realm}/clients/{id}
// (UpdateClientValidator). All fields are optional; nil means "leave
// unchanged".
type UpdateClientRequest struct {
	ClientID                  *string `json:"client_id,omitempty"`
	Name                      *string `json:"name,omitempty"`
	Enabled                   *bool   `json:"enabled,omitempty"`
	DirectAccessGrantsEnabled *bool   `json:"direct_access_grants_enabled,omitempty"`
	AccessTokenLifetime       *int64  `json:"access_token_lifetime,omitempty"`
	RefreshTokenLifetime      *int64  `json:"refresh_token_lifetime,omitempty"`
	IDTokenLifetime           *int64  `json:"id_token_lifetime,omitempty"`
	TemporaryTokenLifetime    *int64  `json:"temporary_token_lifetime,omitempty"`
}

type clientEnvelope struct {
	Data APIClient `json:"data"`
}

type clientsEnvelope struct {
	Data []APIClient `json:"data"`
}

// CreateRedirectURIRequest is the body of POST .../redirects.
type CreateRedirectURIRequest struct {
	Value   string `json:"value"`
	Enabled bool   `json:"enabled"`
}

// UpdateRedirectURIRequest is the body of PUT .../redirects/{id}.
type UpdateRedirectURIRequest struct {
	Enabled bool `json:"enabled"`
}

// ---------------------------------------------------------------------------
// User
// ---------------------------------------------------------------------------

// User mirrors the User schema.
type User struct {
	ID              string    `json:"id"`
	RealmID         string    `json:"realm_id"`
	Username        string    `json:"username"`
	Email           *string   `json:"email,omitempty"`
	EmailVerified   bool      `json:"email_verified"`
	Firstname       *string   `json:"firstname,omitempty"`
	Lastname        *string   `json:"lastname,omitempty"`
	Enabled         bool      `json:"enabled"`
	RequiredActions []string  `json:"required_actions"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

// CreateUserRequest is the body of POST /realms/{realm}/users
// (CreateUserValidator).
type CreateUserRequest struct {
	Username      string  `json:"username"`
	Email         *string `json:"email,omitempty"`
	EmailVerified *bool   `json:"email_verified,omitempty"`
	Firstname     *string `json:"firstname,omitempty"`
	Lastname      *string `json:"lastname,omitempty"`
}

// UpdateUserRequest is the body of PUT /realms/{realm}/users/{id}
// (UpdateUserValidator).
type UpdateUserRequest struct {
	Email           *string  `json:"email,omitempty"`
	EmailVerified   *bool    `json:"email_verified,omitempty"`
	Firstname       *string  `json:"firstname,omitempty"`
	Lastname        *string  `json:"lastname,omitempty"`
	Enabled         *bool    `json:"enabled,omitempty"`
	RequiredActions []string `json:"required_actions,omitempty"`
}

type userEnvelope struct {
	Data User `json:"data"`
}

// ---------------------------------------------------------------------------
// Role
// ---------------------------------------------------------------------------

// Role mirrors the Role schema.
type Role struct {
	ID          string    `json:"id"`
	RealmID     string    `json:"realm_id"`
	ClientID    *string   `json:"client_id,omitempty"`
	Name        string    `json:"name"`
	Description *string   `json:"description,omitempty"`
	Permissions []string  `json:"permissions"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateRoleRequest is the body of POST /realms/{realm}/roles
// (CreateRoleValidator).
type CreateRoleRequest struct {
	Name        string   `json:"name"`
	Description *string  `json:"description,omitempty"`
	Permissions []string `json:"permissions"`
}

// UpdateRoleRequest is the body of PUT /realms/{realm}/roles/{id}
// (UpdateRoleValidator). Note: the update validator does not accept
// permissions; permissions are managed via a dedicated PATCH endpoint.
type UpdateRoleRequest struct {
	Name        *string `json:"name,omitempty"`
	Description *string `json:"description,omitempty"`
}

// UpdateRolePermissionsRequest is the body of
// PATCH /realms/{realm}/roles/{id}/permissions.
type UpdateRolePermissionsRequest struct {
	Permissions []string `json:"permissions"`
}

type roleEnvelope struct {
	Data Role `json:"data"`
}

// ---------------------------------------------------------------------------
// Organization
// ---------------------------------------------------------------------------

// Organization mirrors the Organization schema.
type Organization struct {
	ID          string    `json:"id"`
	RealmID     string    `json:"realm_id"`
	Name        string    `json:"name"`
	Alias       string    `json:"alias"`
	Description *string   `json:"description,omitempty"`
	Domain      *string   `json:"domain,omitempty"`
	RedirectURL *string   `json:"redirect_url,omitempty"`
	Enabled     bool      `json:"enabled"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// CreateOrganizationRequest is the body of POST /realms/{realm}/organizations
// (CreateOrganizationValidator).
type CreateOrganizationRequest struct {
	Name        string  `json:"name"`
	Alias       string  `json:"alias"`
	Description *string `json:"description,omitempty"`
	Domain      *string `json:"domain,omitempty"`
	RedirectURL *string `json:"redirect_url,omitempty"`
	Enabled     bool    `json:"enabled"`
}

// UpdateOrganizationRequest is the body of
// PUT /realms/{realm}/organizations/{id} (UpdateOrganizationValidator).
type UpdateOrganizationRequest struct {
	Name        *string `json:"name,omitempty"`
	Alias       *string `json:"alias,omitempty"`
	Description *string `json:"description,omitempty"`
	Domain      *string `json:"domain,omitempty"`
	RedirectURL *string `json:"redirect_url,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
}

// ---------------------------------------------------------------------------
// Password policy (realm singleton; bare object, always exists)
// ---------------------------------------------------------------------------

// PasswordPolicy mirrors the realm password policy.
type PasswordPolicy struct {
	ID               string    `json:"id"`
	RealmID          string    `json:"realm_id"`
	MinLength        int64     `json:"min_length"`
	RequireUppercase bool      `json:"require_uppercase"`
	RequireLowercase bool      `json:"require_lowercase"`
	RequireNumber    bool      `json:"require_number"`
	RequireSpecial   bool      `json:"require_special"`
	MaxAgeDays       *int64    `json:"max_age_days,omitempty"`
	CreatedAt        time.Time `json:"created_at"`
	UpdatedAt        time.Time `json:"updated_at"`
}

// UpdatePasswordPolicyRequest is the body of PUT /realms/{realm}/password-policy.
type UpdatePasswordPolicyRequest struct {
	MinLength        *int64 `json:"min_length,omitempty"`
	RequireUppercase *bool  `json:"require_uppercase,omitempty"`
	RequireLowercase *bool  `json:"require_lowercase,omitempty"`
	RequireNumber    *bool  `json:"require_number,omitempty"`
	RequireSpecial   *bool  `json:"require_special,omitempty"`
	MaxAgeDays       *int64 `json:"max_age_days,omitempty"`
}

// ---------------------------------------------------------------------------
// SMTP config (realm singleton; bare object; password is write-only)
// ---------------------------------------------------------------------------

// SmtpConfig mirrors the realm SMTP configuration. The server never returns the
// password, so it is not modelled on the response.
type SmtpConfig struct {
	ID         string    `json:"id"`
	RealmID    string    `json:"realm_id"`
	Host       string    `json:"host"`
	Port       int64     `json:"port"`
	Username   string    `json:"username"`
	FromEmail  string    `json:"from_email"`
	FromName   string    `json:"from_name"`
	Encryption string    `json:"encryption"`
	CreatedAt  time.Time `json:"created_at"`
	UpdatedAt  time.Time `json:"updated_at"`
}

// UpsertSmtpConfigRequest is the body of PUT /realms/{realm}/smtp-config
// (UpsertSmtpConfigValidator). All fields are required.
type UpsertSmtpConfigRequest struct {
	Host       string `json:"host"`
	Port       int64  `json:"port"`
	Username   string `json:"username"`
	Password   string `json:"password"`
	FromEmail  string `json:"from_email"`
	FromName   string `json:"from_name"`
	Encryption string `json:"encryption"`
}

// ---------------------------------------------------------------------------
// Organization member (link; bare list)
// ---------------------------------------------------------------------------

// OrganizationMember mirrors a membership link.
type OrganizationMember struct {
	ID             string    `json:"id"`
	OrganizationID string    `json:"organization_id"`
	UserID         string    `json:"user_id"`
	CreatedAt      time.Time `json:"created_at"`
}

// AddOrganizationMemberRequest is the body of POST .../members.
type AddOrganizationMemberRequest struct {
	UserID string `json:"user_id"`
}

// ---------------------------------------------------------------------------
// Identity provider (federation; bare object keyed by alias)
// ---------------------------------------------------------------------------

// IdentityProvider mirrors a federation identity provider. The server augments
// `config` with extra null-valued keys, so the provider tracks only the keys
// the user manages (see the resource's config handling).
type IdentityProvider struct {
	Alias                     string             `json:"alias"`
	InternalID                string             `json:"internal_id"`
	ProviderID                string             `json:"provider_id"`
	Enabled                   bool               `json:"enabled"`
	DisplayName               *string            `json:"display_name,omitempty"`
	FirstBrokerLoginFlowAlias *string            `json:"first_broker_login_flow_alias,omitempty"`
	PostBrokerLoginFlowAlias  *string            `json:"post_broker_login_flow_alias,omitempty"`
	StoreToken                bool               `json:"store_token"`
	AddReadTokenRoleOnCreate  bool               `json:"add_read_token_role_on_create"`
	TrustEmail                bool               `json:"trust_email"`
	LinkOnly                  bool               `json:"link_only"`
	Config                    map[string]*string `json:"config"`
}

// CreateIdentityProviderRequest is the body of POST /identity-providers.
type CreateIdentityProviderRequest struct {
	Alias                     string            `json:"alias"`
	ProviderID                string            `json:"provider_id"`
	DisplayName               *string           `json:"display_name,omitempty"`
	Enabled                   *bool             `json:"enabled,omitempty"`
	Config                    map[string]string `json:"config,omitempty"`
	FirstBrokerLoginFlowAlias *string           `json:"first_broker_login_flow_alias,omitempty"`
	PostBrokerLoginFlowAlias  *string           `json:"post_broker_login_flow_alias,omitempty"`
	StoreToken                *bool             `json:"store_token,omitempty"`
	AddReadTokenRoleOnCreate  *bool             `json:"add_read_token_role_on_create,omitempty"`
	TrustEmail                *bool             `json:"trust_email,omitempty"`
	LinkOnly                  *bool             `json:"link_only,omitempty"`
}

// UpdateIdentityProviderRequest is the body of PUT /identity-providers/{alias}
// (UpdateIdentityProviderValidator). alias and provider_id are immutable.
type UpdateIdentityProviderRequest struct {
	DisplayName               *string           `json:"display_name,omitempty"`
	Enabled                   *bool             `json:"enabled,omitempty"`
	Config                    map[string]string `json:"config,omitempty"`
	FirstBrokerLoginFlowAlias *string           `json:"first_broker_login_flow_alias,omitempty"`
	PostBrokerLoginFlowAlias  *string           `json:"post_broker_login_flow_alias,omitempty"`
	StoreToken                *bool             `json:"store_token,omitempty"`
	AddReadTokenRoleOnCreate  *bool             `json:"add_read_token_role_on_create,omitempty"`
	TrustEmail                *bool             `json:"trust_email,omitempty"`
	LinkOnly                  *bool             `json:"link_only,omitempty"`
}
