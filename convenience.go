package omnitoken

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/grokify/goauth"
	"github.com/grokify/goauth/google"
	"github.com/plexusone/omnivault"
	"github.com/plexusone/omnivault/providers/env"
	"github.com/plexusone/omnivault/providers/file"
	"github.com/plexusone/omnivault/providers/memory"
)

// NewFromCredentialsFile creates a TokenManager from a goauth CredentialsSet JSON file.
// The credentials are loaded into an in-memory vault for the session.
// This is useful for file-based credential workflows without a persistent vault.
func NewFromCredentialsFile(credentialsFile string) (*TokenManager, error) {
	set, err := goauth.ReadFileCredentialsSet(credentialsFile, true)
	if err != nil {
		return nil, fmt.Errorf("failed to read credentials file: %w", err)
	}

	return NewFromCredentialsSet(set)
}

// NewFromCredentialsSet creates a TokenManager from an existing goauth.CredentialsSet.
// The credentials are stored in an in-memory vault.
func NewFromCredentialsSet(set *goauth.CredentialsSet) (*TokenManager, error) {
	v := memory.New()

	mgr, err := New(Config{
		Vault: v,
	})
	if err != nil {
		return nil, err
	}

	// Load all credentials into the vault
	ctx := context.Background()
	if err := mgr.creds.SaveCredentialsSet(ctx, set); err != nil {
		return nil, fmt.Errorf("failed to load credentials: %w", err)
	}

	return mgr, nil
}

// NewFromCredentials creates a TokenManager with a single credential.
// The credential is stored in an in-memory vault under the given name.
func NewFromCredentials(name string, creds *goauth.Credentials) (*TokenManager, error) {
	v := memory.New()

	mgr, err := New(Config{
		Vault: v,
	})
	if err != nil {
		return nil, err
	}

	ctx := context.Background()
	if err := mgr.SetCredentials(ctx, name, creds); err != nil {
		return nil, fmt.Errorf("failed to store credentials: %w", err)
	}

	return mgr, nil
}

// NewFromEnv creates a TokenManager that reads credentials from environment variables.
// Environment variables should be named with the pattern: {prefix}{NAME}_CREDENTIALS
// containing JSON-encoded goauth.Credentials.
//
// If prefix is empty, it defaults to "OMNITOKEN_".
func NewFromEnv(prefix string) (*TokenManager, error) {
	if prefix == "" {
		prefix = "OMNITOKEN_"
	}

	v := env.NewWithConfig(env.Config{
		Prefix: prefix,
	})

	return New(Config{
		Vault:             v,
		CredentialsPrefix: "", // env vars don't need path prefix
		TokensPrefix:      "", // env vars don't need path prefix
	})
}

// NewFromDirectory creates a TokenManager using a directory for file-based storage.
// Credentials are stored as JSON files in {directory}/credentials/
// Tokens are stored as JSON files in {directory}/tokens/
//
// This provides persistent storage without requiring a vault service.
func NewFromDirectory(directory string) (*TokenManager, error) {
	v, err := file.New(file.Config{
		Directory:  directory,
		JSONFormat: true,
		Extension:  ".json",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create file vault: %w", err)
	}

	return New(Config{
		Vault: v,
	})
}

// NewFromVaultURI creates a TokenManager from a vault URI.
// Supported URI schemes:
//   - memory://           - In-memory (testing)
//   - file:///path/to/dir - File-based storage
//   - env://              - Environment variables (with optional prefix: env://PREFIX_)
//   - op://               - 1Password (requires OP_SERVICE_ACCOUNT_TOKEN env var)
//
// Additional providers can be registered via omnivault.RegisterProvider.
func NewFromVaultURI(uri string) (*TokenManager, error) {
	v, err := omnivault.VaultFromURI(uri)
	if err != nil {
		return nil, err
	}
	return New(Config{Vault: v})
}

// NewAuto creates a TokenManager by auto-detecting configuration from environment.
// It checks the following in order:
//  1. OMNITOKEN_VAULT_URI - vault URI (e.g., "op://vault", "file:///path")
//  2. OMNITOKEN_CREDENTIALS_FILE - path to goauth CredentialsSet file
//  3. Falls back to in-memory vault
func NewAuto() (*TokenManager, error) {
	// Check for vault URI
	if uri := os.Getenv("OMNITOKEN_VAULT_URI"); uri != "" {
		return NewFromVaultURI(uri)
	}

	// Check for credentials file
	if credsFile := os.Getenv("OMNITOKEN_CREDENTIALS_FILE"); credsFile != "" {
		return NewFromCredentialsFile(credsFile)
	}

	// Fall back to memory vault
	return New(Config{
		Vault: memory.New(),
	})
}

// LoadGoogleServiceAccount loads a Google service account JSON file into the TokenManager.
// The credentials are stored under the given name with type "gcpsa".
// Scopes should be the OAuth2 scopes required for the Google APIs you'll use.
func (tm *TokenManager) LoadGoogleServiceAccount(ctx context.Context, name, serviceAccountFile string, scopes []string) error {
	// Read and parse the service account file
	googleCreds, err := google.ReadCredentialsFile(serviceAccountFile)
	if err != nil {
		return fmt.Errorf("failed to read service account file: %w", err)
	}

	creds := &goauth.Credentials{
		Service: "google",
		Type:    goauth.TypeGCPSA,
		GCPSA: &goauth.CredentialsGCP{
			GCPCredentials: *googleCreds,
			Scopes:         scopes,
		},
	}

	return tm.SetCredentials(ctx, name, creds)
}

// LoadGoauthCredentials loads a goauth credential from a CredentialsSet file.
// The credential is stored under the given name (or accountKey if name is empty).
func (tm *TokenManager) LoadGoauthCredentials(ctx context.Context, name, credentialsFile, accountKey string) error {
	set, err := goauth.ReadFileCredentialsSet(credentialsFile, true)
	if err != nil {
		return fmt.Errorf("failed to read credentials file: %w", err)
	}

	creds, err := set.Get(accountKey)
	if err != nil {
		return fmt.Errorf("account key %q not found: %w", accountKey, err)
	}

	if name == "" {
		name = accountKey
	}

	return tm.SetCredentials(ctx, name, &creds)
}

// CredentialsDir returns the default credentials directory for the current user.
// On Unix: ~/.config/omnitoken/credentials
// On macOS: ~/Library/Application Support/omnitoken/credentials
// On Windows: %APPDATA%/omnitoken/credentials
func CredentialsDir() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(configDir, "omnitoken", "credentials"), nil
}
