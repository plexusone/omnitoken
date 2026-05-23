// Package omnitoken provides a token management SDK that bridges goauth credentials
// with vault-based storage via omnivault.
//
// # Overview
//
// omnitoken enables applications (particularly MCP servers via omniskill) to:
//
//   - Store and retrieve goauth Credentials in various vault backends
//   - Automatically manage OAuth2 token lifecycle (acquisition, refresh, caching)
//   - Implement goauth's TokenSet interface for vault-backed token storage
//   - Support multiple credential types (OAuth2, JWT, Basic Auth, API keys)
//
// # Architecture
//
// omnitoken sits between three libraries:
//
//   - goauth: Provides the Credentials struct and token acquisition logic
//   - omnivault: Provides vault abstractions for 30+ backends (1Password, HashiCorp Vault, etc.)
//   - omniskill: MCP server framework that uses omnitoken for credential management
//
// # Basic Usage
//
//	import (
//	    "github.com/plexusone/omnitoken"
//	    "github.com/plexusone/omnivault/providers/memory"
//	)
//
//	// Create vault backend (use real vault in production)
//	vault := memory.New()
//
//	// Create token manager
//	mgr, err := omnitoken.New(omnitoken.Config{
//	    Vault: vault,
//	})
//	if err != nil {
//	    log.Fatal(err)
//	}
//	defer mgr.Close()
//
//	// Store credentials
//	creds := &goauth.Credentials{
//	    Service: "github",
//	    Type:    goauth.TypeOAuth2,
//	    OAuth2: &goauth.CredentialsOAuth2{
//	        ClientID:     "...",
//	        ClientSecret: "...",
//	        GrantType:    "client_credentials",
//	    },
//	}
//	mgr.SetCredentials(ctx, "my-github-app", creds)
//
//	// Get authenticated HTTP client
//	client, err := mgr.GetClient(ctx, "my-github-app")
//
// # Vault Integration
//
// omnitoken uses omnivault for storage, supporting backends like:
//
//   - 1Password (op://)
//   - HashiCorp Vault (vault://)
//   - OpenBao (openbao://)
//   - Bitwarden (bw://)
//   - AWS Secrets Manager (aws-sm://)
//   - Azure Key Vault (azure-kv://)
//   - GCP Secret Manager (gcp-sm://)
//   - macOS Keychain (keychain://)
//   - Environment variables (env://)
//   - Files (file://)
//
// # Token Lifecycle
//
// The TokenManager handles the complete token lifecycle:
//
//  1. Retrieves credentials from vault
//  2. Checks for cached/stored valid token
//  3. Refreshes expired tokens using refresh_token if available
//  4. Obtains new tokens when refresh isn't possible
//  5. Stores tokens in vault for persistence across restarts
//
// # goauth Integration
//
// omnitoken implements goauth's TokenSet interface via VaultTokenSet,
// enabling integration with goauth's multi-service token management:
//
//	tokenSet := mgr.TokenSet()
//	// Use with goauth's NewClientWithTokenSet, OAuth2Manager, etc.
package omnitoken
