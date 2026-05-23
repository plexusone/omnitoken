package omnitoken

import "errors"

var (
	// ErrCredentialsNotFound is returned when credentials are not found in the vault.
	ErrCredentialsNotFound = errors.New("credentials not found")

	// ErrTokenNotFound is returned when a token is not found in the vault.
	ErrTokenNotFound = errors.New("token not found")

	// ErrTokenExpired is returned when a token has expired and cannot be refreshed.
	ErrTokenExpired = errors.New("token expired")

	// ErrRefreshFailed is returned when token refresh fails.
	ErrRefreshFailed = errors.New("token refresh failed")

	// ErrInvalidCredentials is returned when credentials are malformed.
	ErrInvalidCredentials = errors.New("invalid credentials")

	// ErrVaultNotConfigured is returned when vault is not configured.
	ErrVaultNotConfigured = errors.New("vault not configured")

	// ErrUnsupportedCredentialType is returned for unsupported credential types.
	ErrUnsupportedCredentialType = errors.New("unsupported credential type")
)
