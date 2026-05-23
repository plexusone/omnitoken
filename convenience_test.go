package omnitoken

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/grokify/goauth"
)

func TestNewFromCredentials(t *testing.T) {
	creds := &goauth.Credentials{
		Service: "test-service",
		Type:    goauth.TypeOAuth2,
		OAuth2: &goauth.CredentialsOAuth2{
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		},
	}

	mgr, err := NewFromCredentials("test-app", creds)
	if err != nil {
		t.Fatalf("NewFromCredentials failed: %v", err)
	}
	defer func() {
		if err := mgr.Close(); err != nil {
			t.Errorf("failed to close: %v", err)
		}
	}()

	ctx := context.Background()
	retrieved, err := mgr.GetCredentials(ctx, "test-app")
	if err != nil {
		t.Fatalf("GetCredentials failed: %v", err)
	}

	if retrieved.OAuth2.ClientID != "test-client-id" {
		t.Errorf("expected client_id=test-client-id, got %s", retrieved.OAuth2.ClientID)
	}
}

func TestNewFromCredentialsSet(t *testing.T) {
	set := &goauth.CredentialsSet{
		Credentials: map[string]goauth.Credentials{
			"app1": {
				Service: "service1",
				Type:    goauth.TypeBasic,
				Basic: &goauth.CredentialsBasicAuth{
					Username: "user1",
					Password: "pass1",
				},
			},
			"app2": {
				Service: "service2",
				Type:    goauth.TypeOAuth2,
				OAuth2: &goauth.CredentialsOAuth2{
					ClientID: "client2",
				},
			},
		},
	}

	mgr, err := NewFromCredentialsSet(set)
	if err != nil {
		t.Fatalf("NewFromCredentialsSet failed: %v", err)
	}
	defer func() {
		if err := mgr.Close(); err != nil {
			t.Errorf("failed to close: %v", err)
		}
	}()

	ctx := context.Background()

	// Check app1
	creds1, err := mgr.GetCredentials(ctx, "app1")
	if err != nil {
		t.Fatalf("GetCredentials(app1) failed: %v", err)
	}
	if creds1.Basic.Username != "user1" {
		t.Errorf("expected username=user1, got %s", creds1.Basic.Username)
	}

	// Check app2
	creds2, err := mgr.GetCredentials(ctx, "app2")
	if err != nil {
		t.Fatalf("GetCredentials(app2) failed: %v", err)
	}
	if creds2.OAuth2.ClientID != "client2" {
		t.Errorf("expected client_id=client2, got %s", creds2.OAuth2.ClientID)
	}

	// List all
	names, err := mgr.ListCredentials(ctx)
	if err != nil {
		t.Fatalf("ListCredentials failed: %v", err)
	}
	if len(names) != 2 {
		t.Errorf("expected 2 credentials, got %d", len(names))
	}
}

func TestNewFromVaultURI(t *testing.T) {
	tests := []struct {
		name    string
		uri     string
		wantErr bool
	}{
		{"memory", "memory://", false},
		{"memory-short", "memory", false},
		{"env", "env://", false},
		{"env-prefix", "env://TEST_", false},
		{"unsupported", "unknown://foo", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mgr, err := NewFromVaultURI(tt.uri)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
					if mgr != nil {
						if closeErr := mgr.Close(); closeErr != nil {
							t.Errorf("failed to close: %v", closeErr)
						}
					}
				}
				return
			}
			if err != nil {
				t.Fatalf("NewFromVaultURI(%q) failed: %v", tt.uri, err)
			}
			defer func() {
				if err := mgr.Close(); err != nil {
					t.Errorf("failed to close: %v", err)
				}
			}()

			// Verify we can use the manager
			ctx := context.Background()
			creds := &goauth.Credentials{
				Service: "test",
				Type:    goauth.TypeBasic,
				Basic:   &goauth.CredentialsBasicAuth{Username: "test"},
			}
			if err := mgr.SetCredentials(ctx, "test", creds); err != nil {
				t.Fatalf("SetCredentials failed: %v", err)
			}
		})
	}
}

func TestNewFromDirectory(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "omnitoken-test-*")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	t.Cleanup(func() {
		if err := os.RemoveAll(tmpDir); err != nil {
			t.Errorf("failed to remove temp dir: %v", err)
		}
	})

	mgr, err := NewFromDirectory(tmpDir)
	if err != nil {
		t.Fatalf("NewFromDirectory failed: %v", err)
	}
	defer func() {
		if err := mgr.Close(); err != nil {
			t.Errorf("failed to close: %v", err)
		}
	}()

	ctx := context.Background()

	// Store credentials
	creds := &goauth.Credentials{
		Service: "test-service",
		Type:    goauth.TypeBasic,
		Basic: &goauth.CredentialsBasicAuth{
			Username: "testuser",
			Password: "testpass",
		},
	}

	if err := mgr.SetCredentials(ctx, "test-app", creds); err != nil {
		t.Fatalf("SetCredentials failed: %v", err)
	}

	// Verify file was created
	expectedFile := filepath.Join(tmpDir, "credentials", "test-app.json")
	if _, err := os.Stat(expectedFile); os.IsNotExist(err) {
		t.Errorf("expected file %s to exist", expectedFile)
	}

	// Retrieve credentials
	retrieved, err := mgr.GetCredentials(ctx, "test-app")
	if err != nil {
		t.Fatalf("GetCredentials failed: %v", err)
	}
	if retrieved.Basic.Username != "testuser" {
		t.Errorf("expected username=testuser, got %s", retrieved.Basic.Username)
	}
}

func TestNewAuto(t *testing.T) {
	// Clear env vars
	if err := os.Unsetenv("OMNITOKEN_VAULT_URI"); err != nil {
		t.Fatalf("failed to unset OMNITOKEN_VAULT_URI: %v", err)
	}
	if err := os.Unsetenv("OMNITOKEN_CREDENTIALS_FILE"); err != nil {
		t.Fatalf("failed to unset OMNITOKEN_CREDENTIALS_FILE: %v", err)
	}

	mgr, err := NewAuto()
	if err != nil {
		t.Fatalf("NewAuto failed: %v", err)
	}
	defer func() {
		if err := mgr.Close(); err != nil {
			t.Errorf("failed to close: %v", err)
		}
	}()

	// Should use memory vault by default
	ctx := context.Background()
	creds := &goauth.Credentials{
		Service: "test",
		Type:    goauth.TypeBasic,
		Basic:   &goauth.CredentialsBasicAuth{Username: "test"},
	}
	if err := mgr.SetCredentials(ctx, "test", creds); err != nil {
		t.Fatalf("SetCredentials failed: %v", err)
	}
}

func TestNewAutoWithVaultURI(t *testing.T) {
	if err := os.Setenv("OMNITOKEN_VAULT_URI", "memory://"); err != nil {
		t.Fatalf("failed to set OMNITOKEN_VAULT_URI: %v", err)
	}
	t.Cleanup(func() {
		if err := os.Unsetenv("OMNITOKEN_VAULT_URI"); err != nil {
			t.Errorf("failed to unset OMNITOKEN_VAULT_URI: %v", err)
		}
	})

	mgr, err := NewAuto()
	if err != nil {
		t.Fatalf("NewAuto failed: %v", err)
	}
	defer func() {
		if err := mgr.Close(); err != nil {
			t.Errorf("failed to close: %v", err)
		}
	}()

	ctx := context.Background()
	creds := &goauth.Credentials{
		Service: "test",
		Type:    goauth.TypeBasic,
		Basic:   &goauth.CredentialsBasicAuth{Username: "test"},
	}
	if err := mgr.SetCredentials(ctx, "test", creds); err != nil {
		t.Fatalf("SetCredentials failed: %v", err)
	}
}

func TestCredentialsDir(t *testing.T) {
	dir, err := CredentialsDir()
	if err != nil {
		t.Fatalf("CredentialsDir failed: %v", err)
	}
	if dir == "" {
		t.Error("expected non-empty directory")
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("expected absolute path, got %s", dir)
	}
}
