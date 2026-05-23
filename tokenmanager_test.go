package omnitoken

import (
	"context"
	"testing"
	"time"

	"github.com/grokify/goauth"
	"github.com/grokify/goauth/multiservice/tokens"
	"github.com/plexusone/omnivault/providers/memory"
	"golang.org/x/oauth2"
)

func TestTokenManagerBasic(t *testing.T) {
	ctx := context.Background()
	vault := memory.New()

	mgr, err := New(Config{
		Vault: vault,
	})
	if err != nil {
		t.Fatalf("failed to create token manager: %v", err)
	}
	defer func() {
		if err := mgr.Close(); err != nil {
			t.Errorf("failed to close token manager: %v", err)
		}
	}()

	// Store credentials
	creds := &goauth.Credentials{
		Service: "test-service",
		Type:    goauth.TypeOAuth2,
		OAuth2: &goauth.CredentialsOAuth2{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
			GrantType:    "client_credentials",
		},
	}

	if err := mgr.SetCredentials(ctx, "test-app", creds); err != nil {
		t.Fatalf("failed to set credentials: %v", err)
	}

	// List credentials
	names, err := mgr.ListCredentials(ctx)
	if err != nil {
		t.Fatalf("failed to list credentials: %v", err)
	}
	if len(names) != 1 || names[0] != "test-app" {
		t.Errorf("expected [test-app], got %v", names)
	}

	// Get credentials
	retrieved, err := mgr.GetCredentials(ctx, "test-app")
	if err != nil {
		t.Fatalf("failed to get credentials: %v", err)
	}
	if retrieved.Service != "test-service" {
		t.Errorf("expected service=test-service, got %s", retrieved.Service)
	}
	if retrieved.Type != goauth.TypeOAuth2 {
		t.Errorf("expected type=oauth2, got %s", retrieved.Type)
	}
	if retrieved.OAuth2.ClientID != "test-client-id" {
		t.Errorf("expected client_id=test-client-id, got %s", retrieved.OAuth2.ClientID)
	}

	// Delete credentials
	if err := mgr.DeleteCredentials(ctx, "test-app"); err != nil {
		t.Fatalf("failed to delete credentials: %v", err)
	}

	names, err = mgr.ListCredentials(ctx)
	if err != nil {
		t.Fatalf("failed to list credentials: %v", err)
	}
	if len(names) != 0 {
		t.Errorf("expected empty list, got %v", names)
	}
}

func TestVaultTokenSet(t *testing.T) {
	vault := memory.New()
	ts := NewVaultTokenSet(vault, "tokens/", nil)

	// Store token info
	token := &oauth2.Token{
		AccessToken:  "test-access-token",
		RefreshToken: "test-refresh-token",
		TokenType:    "Bearer",
		Expiry:       time.Now().Add(time.Hour),
	}

	info := &tokens.TokenInfo{
		ServiceKey:  "test-key",
		ServiceType: "test-service",
		Token:       token,
	}

	if err := ts.SetTokenInfo("test-key", info); err != nil {
		t.Fatalf("failed to set token info: %v", err)
	}

	// Retrieve token info
	retrieved, err := ts.GetTokenInfo("test-key")
	if err != nil {
		t.Fatalf("failed to get token info: %v", err)
	}
	if retrieved.ServiceKey != "test-key" {
		t.Errorf("expected service_key=test-key, got %s", retrieved.ServiceKey)
	}
	if retrieved.Token.AccessToken != "test-access-token" {
		t.Errorf("expected access_token=test-access-token, got %s", retrieved.Token.AccessToken)
	}

	// Get just the token
	tok, err := ts.GetToken("test-key")
	if err != nil {
		t.Fatalf("failed to get token: %v", err)
	}
	if tok.AccessToken != "test-access-token" {
		t.Errorf("expected access_token=test-access-token, got %s", tok.AccessToken)
	}

	// Check exists
	exists, err := ts.ExistsToken("test-key")
	if err != nil {
		t.Fatalf("failed to check exists: %v", err)
	}
	if !exists {
		t.Error("expected token to exist")
	}

	// List tokens
	keys, err := ts.ListTokens()
	if err != nil {
		t.Fatalf("failed to list tokens: %v", err)
	}
	if len(keys) != 1 {
		t.Errorf("expected 1 token, got %d", len(keys))
	}

	// Delete token
	if err := ts.DeleteToken("test-key"); err != nil {
		t.Fatalf("failed to delete token: %v", err)
	}

	exists, err = ts.ExistsToken("test-key")
	if err != nil {
		t.Fatalf("failed to check exists: %v", err)
	}
	if exists {
		t.Error("expected token to not exist after delete")
	}
}

func TestCredentialsStore(t *testing.T) {
	ctx := context.Background()
	vault := memory.New()
	store := NewCredentialsStore(vault, "credentials/", nil)

	// Test basic auth credentials
	basicCreds := &goauth.Credentials{
		Service: "basic-service",
		Type:    goauth.TypeBasic,
		Basic: &goauth.CredentialsBasicAuth{
			Username:  "testuser",
			Password:  "testpass",
			ServerURL: "https://example.com",
		},
	}

	if err := store.Set(ctx, "basic-app", basicCreds); err != nil {
		t.Fatalf("failed to set basic credentials: %v", err)
	}

	retrieved, err := store.Get(ctx, "basic-app")
	if err != nil {
		t.Fatalf("failed to get basic credentials: %v", err)
	}
	if retrieved.Type != goauth.TypeBasic {
		t.Errorf("expected type=basic, got %s", retrieved.Type)
	}
	if retrieved.Basic.Username != "testuser" {
		t.Errorf("expected username=testuser, got %s", retrieved.Basic.Username)
	}

	// Test not found
	_, err = store.Get(ctx, "nonexistent")
	if err == nil {
		t.Error("expected error for nonexistent credentials")
	}

	// Test exists
	exists, err := store.Exists(ctx, "basic-app")
	if err != nil {
		t.Fatalf("failed to check exists: %v", err)
	}
	if !exists {
		t.Error("expected credentials to exist")
	}

	exists, err = store.Exists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("failed to check exists: %v", err)
	}
	if exists {
		t.Error("expected credentials to not exist")
	}
}

func TestConfigValidation(t *testing.T) {
	// Test missing vault
	cfg := Config{}
	if err := cfg.Validate(); err != ErrVaultNotConfigured {
		t.Errorf("expected ErrVaultNotConfigured, got %v", err)
	}

	// Test valid config
	vault := memory.New()
	cfg.Vault = vault
	if err := cfg.Validate(); err != nil {
		t.Errorf("expected no error, got %v", err)
	}
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.CredentialsPrefix != "credentials/" {
		t.Errorf("expected credentials/, got %s", cfg.CredentialsPrefix)
	}
	if cfg.TokensPrefix != "tokens/" {
		t.Errorf("expected tokens/, got %s", cfg.TokensPrefix)
	}
	if cfg.RefreshBuffer != 60*time.Second {
		t.Errorf("expected 60s, got %v", cfg.RefreshBuffer)
	}
	if !cfg.AutoRefresh {
		t.Error("expected AutoRefresh=true")
	}
}

func TestTokenSetInterface(t *testing.T) {
	vault := memory.New()
	ts := NewVaultTokenSet(vault, "tokens/", nil)

	// Verify it implements the interface
	var _ tokens.TokenSet = ts
}
