// Package omnitoken provides a token management SDK that bridges goauth credentials
// with vault-based storage via omnivault. It enables MCP servers and other applications
// to manage OAuth2 tokens and service credentials generically across multiple vault backends.
//
// Key features:
//   - Store and retrieve goauth Credentials in vaults (1Password, HashiCorp Vault, etc.)
//   - Implement goauth's TokenSet interface for vault-backed token storage
//   - Automatic token refresh with configurable buffer time
//   - Support for multiple credential types (OAuth2, JWT, Basic Auth, etc.)
//
// Basic usage:
//
//	import (
//	    "github.com/plexusone/omnitoken"
//	    "github.com/plexusone/omnivault/providers/memory"
//	)
//
//	// Create vault backend
//	vault := memory.New()
//
//	// Create token manager
//	mgr, err := omnitoken.New(omnitoken.Config{
//	    Vault: vault,
//	})
//
//	// Get an authenticated HTTP client
//	client, err := mgr.GetClient(ctx, "my-app")
package omnitoken

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/grokify/goauth"
	"github.com/grokify/goauth/multiservice/tokens"
	"github.com/plexusone/omnivault/vault"
	"golang.org/x/oauth2"
)

// TokenManager manages credentials and tokens using vault storage.
// It provides a unified interface for obtaining authenticated HTTP clients
// from stored credentials, with automatic token refresh and caching.
type TokenManager struct {
	config     Config
	creds      *CredentialsStore
	tokenSet   *VaultTokenSet
	tokenCache sync.Map // map[string]*cachedToken
}

// cachedToken holds a token with its associated configuration.
type cachedToken struct {
	token     *oauth2.Token
	config    *oauth2.Config
	creds     *goauth.Credentials
	expiresAt time.Time
}

// New creates a new TokenManager with the given configuration.
func New(cfg Config) (*TokenManager, error) {
	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &TokenManager{
		config:   cfg,
		creds:    NewCredentialsStore(cfg.Vault, cfg.CredentialsPrefix, cfg.Logger),
		tokenSet: NewVaultTokenSet(cfg.Vault, cfg.TokensPrefix, cfg.Logger),
	}, nil
}

// NewWithVault creates a new TokenManager with the given vault and default configuration.
func NewWithVault(v vault.Vault) (*TokenManager, error) {
	cfg := DefaultConfig()
	cfg.Vault = v
	return New(cfg)
}

// GetClient returns an authenticated HTTP client for the named credentials.
// It retrieves credentials from the vault, obtains or refreshes the token as needed,
// and returns a client configured with the token.
func (tm *TokenManager) GetClient(ctx context.Context, name string) (*http.Client, error) {
	creds, err := tm.creds.Get(ctx, name)
	if err != nil {
		return nil, err
	}

	// Try to get cached/stored token first
	if tm.config.AutoRefresh {
		if token, err := tm.getValidToken(ctx, name, creds); err == nil {
			return tm.clientFromToken(token), nil
		}
	}

	// Get new client from credentials
	client, err := creds.NewClient(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to create client: %w", err)
	}

	// Cache the token if available
	if creds.Token != nil {
		if err := tm.storeToken(name, creds, creds.Token); err != nil {
			tm.config.Logger.Warn("failed to cache token",
				"name", name,
				"error", err,
			)
		}
	}

	return client, nil
}

// GetToken returns a valid OAuth2 token for the named credentials.
// It retrieves from cache/vault if valid, or obtains a new token if needed.
func (tm *TokenManager) GetToken(ctx context.Context, name string) (*oauth2.Token, error) {
	creds, err := tm.creds.Get(ctx, name)
	if err != nil {
		return nil, err
	}

	// Try cached/stored token first
	if token, err := tm.getValidToken(ctx, name, creds); err == nil {
		return token, nil
	}

	// Get new token
	token, err := creds.NewToken(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to get new token: %w", err)
	}

	// Store the new token
	if err := tm.storeToken(name, creds, token); err != nil {
		tm.config.Logger.Warn("failed to store token",
			"name", name,
			"error", err,
		)
	}

	return token, nil
}

// GetCredentials retrieves credentials from the vault by name.
func (tm *TokenManager) GetCredentials(ctx context.Context, name string) (*goauth.Credentials, error) {
	return tm.creds.Get(ctx, name)
}

// SetCredentials stores credentials in the vault.
func (tm *TokenManager) SetCredentials(ctx context.Context, name string, creds *goauth.Credentials) error {
	return tm.creds.Set(ctx, name, creds)
}

// DeleteCredentials removes credentials from the vault.
func (tm *TokenManager) DeleteCredentials(ctx context.Context, name string) error {
	// Also delete associated token
	if err := tm.tokenSet.DeleteToken(name); err != nil {
		tm.config.Logger.Warn("failed to delete associated token",
			"name", name,
			"error", err,
		)
	}
	tm.tokenCache.Delete(name)
	return tm.creds.Delete(ctx, name)
}

// ListCredentials returns all credential names in the vault.
func (tm *TokenManager) ListCredentials(ctx context.Context) ([]string, error) {
	return tm.creds.List(ctx)
}

// RefreshToken forces a token refresh for the named credentials.
func (tm *TokenManager) RefreshToken(ctx context.Context, name string) (*oauth2.Token, error) {
	creds, err := tm.creds.Get(ctx, name)
	if err != nil {
		return nil, err
	}

	// Get current token for refresh
	currentToken, _ := tm.tokenSet.GetToken(name)
	if currentToken == nil || currentToken.RefreshToken == "" {
		// No refresh token, get new token
		return tm.GetToken(ctx, name)
	}

	// Build OAuth2 config for refresh
	oauth2Config := tm.buildOAuth2Config(creds)
	if oauth2Config == nil {
		return nil, ErrUnsupportedCredentialType
	}

	// Perform refresh
	tokenSource := oauth2Config.TokenSource(ctx, currentToken)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrRefreshFailed, err)
	}

	// Store refreshed token
	if err := tm.storeToken(name, creds, newToken); err != nil {
		tm.config.Logger.Warn("failed to store refreshed token",
			"name", name,
			"error", err,
		)
	}

	return newToken, nil
}

// TokenSet returns the underlying goauth TokenSet implementation.
// This allows integration with goauth's multi-service token management.
func (tm *TokenManager) TokenSet() tokens.TokenSet {
	return tm.tokenSet
}

// CredentialsStore returns the underlying credentials store.
func (tm *TokenManager) CredentialsStore() *CredentialsStore {
	return tm.creds
}

// Close releases resources held by the token manager.
func (tm *TokenManager) Close() error {
	return tm.config.Vault.Close()
}

// getValidToken retrieves a valid token from cache or vault.
func (tm *TokenManager) getValidToken(ctx context.Context, name string, creds *goauth.Credentials) (*oauth2.Token, error) {
	// Check in-memory cache first
	if cached, ok := tm.tokenCache.Load(name); ok {
		ct := cached.(*cachedToken)
		if tm.isTokenValid(ct.token) {
			return ct.token, nil
		}

		// Try to refresh if we have a refresh token
		if ct.token.RefreshToken != "" && ct.config != nil {
			if newToken, err := tm.refreshCachedToken(ctx, name, ct); err == nil {
				return newToken, nil
			}
		}
	}

	// Check vault storage
	tokenInfo, err := tm.tokenSet.GetTokenInfo(name)
	if err != nil {
		return nil, err
	}

	if tokenInfo.Token == nil {
		return nil, ErrTokenNotFound
	}

	if tm.isTokenValid(tokenInfo.Token) {
		// Update cache
		tm.tokenCache.Store(name, &cachedToken{
			token:     tokenInfo.Token,
			config:    tm.buildOAuth2Config(creds),
			creds:     creds,
			expiresAt: tokenInfo.Token.Expiry,
		})
		return tokenInfo.Token, nil
	}

	// Try refresh
	if tokenInfo.Token.RefreshToken != "" {
		oauth2Config := tm.buildOAuth2Config(creds)
		if oauth2Config != nil {
			tokenSource := oauth2Config.TokenSource(ctx, tokenInfo.Token)
			if newToken, err := tokenSource.Token(); err == nil {
				if err := tm.storeToken(name, creds, newToken); err == nil {
					return newToken, nil
				}
			}
		}
	}

	return nil, ErrTokenExpired
}

// isTokenValid checks if a token is still valid considering the refresh buffer.
func (tm *TokenManager) isTokenValid(token *oauth2.Token) bool {
	if token == nil {
		return false
	}
	if token.Expiry.IsZero() {
		// No expiry set, consider valid if access token present
		return token.AccessToken != ""
	}
	// Token is valid if it expires after now + buffer
	return token.Expiry.After(time.Now().Add(tm.config.RefreshBuffer))
}

// storeToken stores a token in both cache and vault.
func (tm *TokenManager) storeToken(name string, creds *goauth.Credentials, token *oauth2.Token) error {
	// Store in vault
	tokenInfo := &tokens.TokenInfo{
		ServiceKey:  name,
		ServiceType: creds.Service,
		Token:       token,
	}
	if err := tm.tokenSet.SetTokenInfo(name, tokenInfo); err != nil {
		return err
	}

	// Store in cache
	tm.tokenCache.Store(name, &cachedToken{
		token:     token,
		config:    tm.buildOAuth2Config(creds),
		creds:     creds,
		expiresAt: token.Expiry,
	})

	return nil
}

// refreshCachedToken refreshes a cached token.
func (tm *TokenManager) refreshCachedToken(ctx context.Context, name string, ct *cachedToken) (*oauth2.Token, error) {
	if ct.config == nil {
		return nil, errors.New("no oauth2 config for refresh")
	}

	tokenSource := ct.config.TokenSource(ctx, ct.token)
	newToken, err := tokenSource.Token()
	if err != nil {
		return nil, err
	}

	// Update cache and storage
	if err := tm.storeToken(name, ct.creds, newToken); err != nil {
		tm.config.Logger.Warn("failed to store refreshed token",
			"name", name,
			"error", err,
		)
	}

	return newToken, nil
}

// buildOAuth2Config builds an oauth2.Config from goauth credentials.
func (tm *TokenManager) buildOAuth2Config(creds *goauth.Credentials) *oauth2.Config {
	if creds.OAuth2 == nil {
		return nil
	}
	return &oauth2.Config{
		ClientID:     creds.OAuth2.ClientID,
		ClientSecret: creds.OAuth2.ClientSecret,
		Endpoint:     creds.OAuth2.Endpoint,
		RedirectURL:  creds.OAuth2.RedirectURL,
		Scopes:       creds.OAuth2.Scopes,
	}
}

// clientFromToken creates an HTTP client from a token.
func (tm *TokenManager) clientFromToken(token *oauth2.Token) *http.Client {
	return oauth2.NewClient(context.Background(), oauth2.StaticTokenSource(token))
}
