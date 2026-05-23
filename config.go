package omnitoken

import (
	"log/slog"
	"time"

	"github.com/plexusone/omnivault/vault"
)

// Config configures the TokenManager.
type Config struct {
	// Vault is the vault backend for storing credentials and tokens.
	// Required.
	Vault vault.Vault

	// CredentialsPrefix is the path prefix for storing credentials in the vault.
	// Default: "credentials/"
	CredentialsPrefix string

	// TokensPrefix is the path prefix for storing tokens in the vault.
	// Default: "tokens/"
	TokensPrefix string

	// RefreshBuffer is the time before expiry to trigger token refresh.
	// Default: 60 seconds
	RefreshBuffer time.Duration

	// AutoRefresh enables automatic token refresh when tokens are near expiry.
	// Default: true
	AutoRefresh bool

	// Logger is the logger for the token manager.
	// Default: slog.Default()
	Logger *slog.Logger
}

// DefaultConfig returns a Config with default values.
// Vault must still be set before use.
func DefaultConfig() Config {
	return Config{
		CredentialsPrefix: "credentials/",
		TokensPrefix:      "tokens/",
		RefreshBuffer:     60 * time.Second,
		AutoRefresh:       true,
		Logger:            slog.Default(),
	}
}

// Validate validates the configuration and returns an error if invalid.
func (c *Config) Validate() error {
	if c.Vault == nil {
		return ErrVaultNotConfigured
	}
	return nil
}

// applyDefaults applies default values to unset fields.
func (c *Config) applyDefaults() {
	if c.CredentialsPrefix == "" {
		c.CredentialsPrefix = "credentials/"
	}
	if c.TokensPrefix == "" {
		c.TokensPrefix = "tokens/"
	}
	if c.RefreshBuffer == 0 {
		c.RefreshBuffer = 60 * time.Second
	}
	if c.Logger == nil {
		c.Logger = slog.Default()
	}
}
