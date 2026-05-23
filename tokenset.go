package omnitoken

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/grokify/goauth/multiservice/tokens"
	"github.com/plexusone/omnivault/vault"
	"golang.org/x/oauth2"
)

// VaultTokenSet implements goauth's tokens.TokenSet interface backed by an omnivault.
// This allows goauth's multi-service token management to use vault storage.
type VaultTokenSet struct {
	vault  vault.Vault
	prefix string
	logger *slog.Logger
}

// NewVaultTokenSet creates a new VaultTokenSet.
func NewVaultTokenSet(v vault.Vault, prefix string, logger *slog.Logger) *VaultTokenSet {
	if prefix == "" {
		prefix = "tokens/"
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &VaultTokenSet{
		vault:  v,
		prefix: prefix,
		logger: logger,
	}
}

// GetTokenInfo retrieves token info from the vault.
func (ts *VaultTokenSet) GetTokenInfo(key string) (*tokens.TokenInfo, error) {
	key = tokens.FormatKey(key)
	path := ts.keyToPath(key)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	secret, err := ts.vault.Get(ctx, path)
	if err != nil {
		if isNotFoundError(err) {
			return nil, fmt.Errorf("%w: %s", ErrTokenNotFound, key)
		}
		return nil, fmt.Errorf("failed to get token info: %w", err)
	}

	tokenInfo, err := parseTokenInfo(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to parse token info: %w", err)
	}

	return tokenInfo, nil
}

// GetToken retrieves just the OAuth2 token from the vault.
func (ts *VaultTokenSet) GetToken(key string) (*oauth2.Token, error) {
	tokenInfo, err := ts.GetTokenInfo(key)
	if err != nil {
		return nil, err
	}
	if tokenInfo.Token == nil {
		return nil, fmt.Errorf("%w: token is nil for key %s", ErrTokenNotFound, key)
	}
	return tokenInfo.Token, nil
}

// SetTokenInfo stores token info in the vault.
func (ts *VaultTokenSet) SetTokenInfo(key string, tokenInfo *tokens.TokenInfo) error {
	key = tokens.FormatKey(key)
	path := ts.keyToPath(key)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	secret, err := tokenInfoToSecret(tokenInfo)
	if err != nil {
		return fmt.Errorf("failed to serialize token info: %w", err)
	}

	if err := ts.vault.Set(ctx, path, secret); err != nil {
		return fmt.Errorf("failed to store token info: %w", err)
	}

	ts.logger.Debug("stored token info",
		"key", key,
		"service_key", tokenInfo.ServiceKey,
		"service_type", tokenInfo.ServiceType,
	)

	return nil
}

// DeleteToken removes a token from the vault.
func (ts *VaultTokenSet) DeleteToken(key string) error {
	key = tokens.FormatKey(key)
	path := ts.keyToPath(key)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := ts.vault.Delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete token: %w", err)
	}

	ts.logger.Debug("deleted token", "key", key)
	return nil
}

// ListTokens returns all token keys in the vault.
func (ts *VaultTokenSet) ListTokens() ([]string, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	paths, err := ts.vault.List(ctx, ts.prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list tokens: %w", err)
	}

	keys := make([]string, 0, len(paths))
	for _, path := range paths {
		key := ts.pathToKey(path)
		if key != "" {
			keys = append(keys, key)
		}
	}

	return keys, nil
}

// ExistsToken checks if a token exists in the vault.
func (ts *VaultTokenSet) ExistsToken(key string) (bool, error) {
	key = tokens.FormatKey(key)
	path := ts.keyToPath(key)

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	return ts.vault.Exists(ctx, path)
}

// keyToPath converts a token key to a vault path.
func (ts *VaultTokenSet) keyToPath(key string) string {
	return ts.prefix + sanitizeKey(key)
}

// pathToKey converts a vault path to a token key.
func (ts *VaultTokenSet) pathToKey(path string) string {
	if !strings.HasPrefix(path, ts.prefix) {
		return ""
	}
	return strings.TrimPrefix(path, ts.prefix)
}

// sanitizeKey sanitizes a key for use in vault paths.
func sanitizeKey(key string) string {
	key = strings.TrimSpace(key)
	key = strings.ReplaceAll(key, "/", "_")
	key = strings.ReplaceAll(key, "\\", "_")
	return key
}

// parseTokenInfo parses a vault secret into TokenInfo.
func parseTokenInfo(secret *vault.Secret) (*tokens.TokenInfo, error) {
	var tokenInfo tokens.TokenInfo

	// First try to unmarshal from the Value field (JSON)
	if secret.Value != "" {
		if err := json.Unmarshal([]byte(secret.Value), &tokenInfo); err == nil {
			return &tokenInfo, nil
		}
	}

	// Fall back to Fields for multi-field secrets
	if secret.Fields != nil {
		tokenInfo.ServiceKey = secret.Fields["service_key"]
		tokenInfo.ServiceType = secret.Fields["service_type"]

		if tokenJSON := secret.Fields["token"]; tokenJSON != "" {
			var token oauth2.Token
			if err := json.Unmarshal([]byte(tokenJSON), &token); err != nil {
				return nil, fmt.Errorf("failed to parse token field: %w", err)
			}
			tokenInfo.Token = &token
		}

		return &tokenInfo, nil
	}

	return nil, fmt.Errorf("secret has no parseable token info")
}

// tokenInfoToSecret converts TokenInfo to a vault secret.
func tokenInfoToSecret(tokenInfo *tokens.TokenInfo) (*vault.Secret, error) {
	data, err := json.Marshal(tokenInfo)
	if err != nil {
		return nil, err
	}

	// Also store as fields for providers that support multi-field
	fields := make(map[string]string)
	fields["service_key"] = tokenInfo.ServiceKey
	fields["service_type"] = tokenInfo.ServiceType

	if tokenInfo.Token != nil {
		//nolint:gosec // G117: OAuth token stored in vault per RFC 6749
		tokenData, err := json.Marshal(tokenInfo.Token)
		if err != nil {
			return nil, fmt.Errorf("failed to serialize token: %w", err)
		}
		fields["token"] = string(tokenData)
		fields["access_token"] = tokenInfo.Token.AccessToken
		if tokenInfo.Token.RefreshToken != "" {
			fields["refresh_token"] = tokenInfo.Token.RefreshToken
		}
		if !tokenInfo.Token.Expiry.IsZero() {
			fields["expiry"] = tokenInfo.Token.Expiry.Format(time.RFC3339)
		}
	}

	now := vault.Now()
	return &vault.Secret{
		Value:  string(data),
		Fields: fields,
		Metadata: vault.Metadata{
			ModifiedAt: now,
			Tags: map[string]string{
				"type":         "oauth2_token",
				"service_key":  tokenInfo.ServiceKey,
				"service_type": tokenInfo.ServiceType,
			},
		},
	}, nil
}

// isNotFoundError checks if an error indicates a secret was not found.
func isNotFoundError(err error) bool {
	return err == vault.ErrSecretNotFound ||
		strings.Contains(err.Error(), "not found") ||
		strings.Contains(err.Error(), "does not exist")
}

// Ensure VaultTokenSet implements tokens.TokenSet.
var _ tokens.TokenSet = (*VaultTokenSet)(nil)
