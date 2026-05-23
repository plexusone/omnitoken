# OmniToken

[![Go CI][go-ci-svg]][go-ci-url]
[![Go Lint][go-lint-svg]][go-lint-url]
[![Go Report Card][goreport-svg]][goreport-url]
[![Docs][docs-godoc-svg]][docs-godoc-url]
[![License][license-svg]][license-url]

 [go-ci-svg]: https://github.com/plexusone/omnitoken/actions/workflows/go-ci.yaml/badge.svg?branch=main
 [go-ci-url]: https://github.com/plexusone/omnitoken/actions/workflows/go-ci.yaml
 [go-lint-svg]: https://github.com/plexusone/omnitoken/actions/workflows/go-lint.yaml/badge.svg?branch=main
 [go-lint-url]: https://github.com/plexusone/omnitoken/actions/workflows/go-lint.yaml
 [goreport-svg]: https://goreportcard.com/badge/github.com/plexusone/omnitoken
 [goreport-url]: https://goreportcard.com/report/github.com/plexusone/omnitoken
 [docs-godoc-svg]: https://pkg.go.dev/badge/github.com/plexusone/omnitoken
 [docs-godoc-url]: https://pkg.go.dev/github.com/plexusone/omnitoken
 [license-svg]: https://img.shields.io/badge/license-MIT-blue.svg
 [license-url]: https://github.com/plexusone/omnitoken/blob/main/LICENSE

Token management SDK that bridges [goauth](https://github.com/grokify/goauth) credentials with vault-based storage via [omnivault](https://github.com/plexusone/omnivault).

## Overview

OmniToken enables applications (particularly MCP servers) to:

- Store and retrieve goauth Credentials in various vault backends
- Automatically manage OAuth2 token lifecycle (acquisition, refresh, caching)
- Implement goauth's TokenSet interface for vault-backed token storage
- Support multiple credential types (OAuth2, JWT, Basic Auth, API keys, GCP service accounts)

## Architecture

```
┌──────────────────────────────────────────────────────────────┐
│                      Applications                            │
│  ┌───────────┐  ┌───────────┐  ┌─────────────────┐           │
│  │ mcp-google│  │  mcp-aha  │  │ mcp-confluence  │           │
│  └─────┬─────┘  └─────┬─────┘  └────────┬────────┘           │
│        └──────────────┼─────────────────┘                    │
│                       │                                      │
│              ┌────────▼────────┐                             │
│              │    omnitoken    │ ← Credential & token mgmt   │
│              └────────┬────────┘                             │
└───────────────────────┼──────────────────────────────────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
   ┌────▼────┐    ┌─────▼─────┐   ┌─────▼─────┐
   │ goauth  │    │ omnivault │   │  oauth2   │
   │(creds)  │    │ (storage) │   │ (tokens)  │
   └─────────┘    └─────┬─────┘   └───────────┘
                        │
        ┌───────────────┼───────────────┐
        │               │               │
   ┌────▼────┐    ┌─────▼─────┐   ┌─────▼─────┐
   │1Password│    │ Bitwarden │   │  Keeper   │
   └─────────┘    └───────────┘   └───────────┘
```

## Installation

```bash
go get github.com/plexusone/omnitoken
```

## Quick Start

### From Vault URI

```go
import "github.com/plexusone/omnitoken"

// Create from vault URI (1Password, Bitwarden, file, etc.)
mgr, err := omnitoken.NewFromVaultURI("op://MyVault")
if err != nil {
    log.Fatal(err)
}
defer mgr.Close()

// Get credentials stored in the vault
creds, err := mgr.GetCredentials(ctx, "my-api")

// Get an authenticated HTTP client
client, err := mgr.GetClient(ctx, "my-api")
```

### From Credentials File

```go
// Load from goauth CredentialsSet file
mgr, err := omnitoken.NewFromCredentialsFile("/path/to/credentials.json")
if err != nil {
    log.Fatal(err)
}
defer mgr.Close()

// Get client for a specific account
client, err := mgr.GetClient(ctx, "myaccount")
```

### Auto-Detection

```go
// Auto-detect from environment variables:
// - OMNITOKEN_VAULT_URI: vault URI
// - OMNITOKEN_CREDENTIALS_FILE: credentials file path
mgr, err := omnitoken.NewAuto()
if err != nil {
    log.Fatal(err)
}
defer mgr.Close()
```

## Credential Sources

| Source | Constructor | Description |
|--------|-------------|-------------|
| Vault URI | `NewFromVaultURI(uri)` | Any omnivault-supported backend |
| Credentials File | `NewFromCredentialsFile(path)` | goauth CredentialsSet JSON |
| CredentialsSet | `NewFromCredentialsSet(set)` | In-memory from goauth.CredentialsSet |
| Single Credential | `NewFromCredentials(name, creds)` | Single goauth.Credentials |
| Environment | `NewFromEnv(prefix)` | Environment variables |
| Directory | `NewFromDirectory(dir)` | File-based storage |
| Auto | `NewAuto()` | Auto-detect from environment |

## Supported Vault URIs

| Provider | URI Pattern | Requirements |
|----------|-------------|--------------|
| 1Password | `op://vault` | `OP_SERVICE_ACCOUNT_TOKEN` env var |
| Bitwarden | `bw://org-id` | `BW_ACCESS_TOKEN`, `BW_ORGANIZATION_ID` env vars |
| Keeper | `keeper://` | `KSM_TOKEN` or `KSM_CONFIG` env var |
| File | `file:///path` | None |
| Environment | `env://PREFIX_` | None |
| Memory | `memory://` | None (testing) |

To use 1Password, Bitwarden, or Keeper, import [omnivault-desktop](https://github.com/plexusone/omnivault-desktop):

```go
import _ "github.com/plexusone/omnivault-desktop"
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `OMNITOKEN_VAULT_URI` | Vault URI for `NewAuto()` |
| `OMNITOKEN_CREDENTIALS_FILE` | Credentials file path for `NewAuto()` |
| `OMNITOKEN_CREDENTIALS_NAME` | Default credential name (used by MCP servers) |

## API Reference

### TokenManager

```go
// Create token manager
mgr, err := omnitoken.New(omnitoken.Config{
    Vault:         vault,           // omnivault.Vault implementation
    AutoRefresh:   true,            // Auto-refresh expired tokens
    RefreshBuffer: 5 * time.Minute, // Refresh before expiry
})

// Credential operations
creds, err := mgr.GetCredentials(ctx, "name")
err := mgr.SetCredentials(ctx, "name", creds)
err := mgr.DeleteCredentials(ctx, "name")
names, err := mgr.ListCredentials(ctx)

// Token operations
client, err := mgr.GetClient(ctx, "name")     // Get authenticated HTTP client
token, err := mgr.GetToken(ctx, "name")       // Get OAuth2 token
token, err := mgr.RefreshToken(ctx, "name")   // Force refresh

// goauth integration
tokenSet := mgr.TokenSet()                    // Get goauth TokenSet interface
credStore := mgr.CredentialsStore()           // Get credentials store

// Cleanup
err := mgr.Close()
```

### Loading Specific Credential Types

```go
// Load Google service account
err := mgr.LoadGoogleServiceAccount(ctx, "google", "/path/to/sa.json", []string{
    "https://www.googleapis.com/auth/presentations.readonly",
    "https://www.googleapis.com/auth/documents.readonly",
})

// Load from goauth CredentialsSet file
err := mgr.LoadGoauthCredentials(ctx, "myservice", "/path/to/creds.json", "accountKey")
```

## Credential Types

OmniToken supports all goauth credential types:

| Type | Description |
|------|-------------|
| `oauth2` | OAuth2 client credentials, authorization code, etc. |
| `jwt` | JWT bearer tokens |
| `basic` | HTTP Basic Auth |
| `headerquery` | Custom header/query authentication |
| `gcpsa` | Google Cloud service account |

## Token Lifecycle

The TokenManager handles the complete token lifecycle:

1. Retrieves credentials from vault
2. Checks for cached/stored valid token
3. Refreshes expired tokens using refresh_token if available
4. Obtains new tokens when refresh isn't possible
5. Stores tokens in vault for persistence across restarts

## Usage in MCP Servers

OmniToken is designed for use in MCP servers. See [mcp-google](https://github.com/plexusone/mcp-google) for a complete example:

```go
import (
    "github.com/plexusone/omnitoken"
    _ "github.com/plexusone/omnivault-desktop"
)

func main() {
    // Create token manager from vault
    mgr, err := omnitoken.NewFromVaultURI(os.Getenv("OMNITOKEN_VAULT_URI"))
    if err != nil {
        log.Fatal(err)
    }
    defer mgr.Close()

    // Get credentials for the service
    creds, err := mgr.GetCredentials(ctx, os.Getenv("OMNITOKEN_CREDENTIALS_NAME"))
    if err != nil {
        log.Fatal(err)
    }

    // Create authenticated HTTP client
    client, err := creds.NewClient(ctx)
    if err != nil {
        log.Fatal(err)
    }

    // Use client with service SDK...
}
```

## License

MIT
