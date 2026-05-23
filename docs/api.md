# API Reference

## TokenManager

The main entry point for credential and token management.

### Creating a TokenManager

```go
// Full configuration
mgr, err := omnitoken.New(omnitoken.Config{
    Vault:         vault,           // omnivault.Vault implementation
    AutoRefresh:   true,            // Auto-refresh expired tokens
    RefreshBuffer: 5 * time.Minute, // Refresh before expiry
})
```

### Credential Operations

```go
// Store credentials
err := mgr.SetCredentials(ctx, "name", creds)

// Retrieve credentials
creds, err := mgr.GetCredentials(ctx, "name")

// Delete credentials
err := mgr.DeleteCredentials(ctx, "name")

// List all credential names
names, err := mgr.ListCredentials(ctx)
```

### Token Operations

```go
// Get an authenticated HTTP client
client, err := mgr.GetClient(ctx, "name")

// Get the OAuth2 token directly
token, err := mgr.GetToken(ctx, "name")

// Force a token refresh
token, err := mgr.RefreshToken(ctx, "name")
```

### goauth Integration

```go
// Get goauth TokenSet interface
tokenSet := mgr.TokenSet()

// Get credentials store
credStore := mgr.CredentialsStore()
```

### Cleanup

```go
// Close the manager and release resources
err := mgr.Close()
```

## Loading Specific Credential Types

### Google Service Account

```go
err := mgr.LoadGoogleServiceAccount(ctx, "google", "/path/to/sa.json", []string{
    "https://www.googleapis.com/auth/presentations.readonly",
    "https://www.googleapis.com/auth/documents.readonly",
})
```

### goauth Credentials File

```go
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

## Config Options

| Field | Type | Description |
|-------|------|-------------|
| `Vault` | `vault.Vault` | Vault backend for storage |
| `AutoRefresh` | `bool` | Automatically refresh expired tokens |
| `RefreshBuffer` | `time.Duration` | Time before expiry to trigger refresh |
| `Logger` | `*slog.Logger` | Logger for debug output |

## Token Lifecycle

The TokenManager handles the complete token lifecycle:

1. Retrieves credentials from vault
2. Checks for cached/stored valid token
3. Refreshes expired tokens using refresh_token if available
4. Obtains new tokens when refresh isn't possible
5. Stores tokens in vault for persistence across restarts
