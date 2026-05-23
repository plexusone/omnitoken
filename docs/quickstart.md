# Quick Start

## From Vault URI

The simplest way to use OmniToken is with a vault URI:

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

## From Credentials File

Load credentials from a goauth CredentialsSet JSON file:

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

## Auto-Detection

Let OmniToken automatically detect the credential source from environment variables:

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

## Basic Operations

Once you have a TokenManager, you can perform these operations:

```go
ctx := context.Background()

// Store credentials
err := mgr.SetCredentials(ctx, "my-service", creds)

// Retrieve credentials
creds, err := mgr.GetCredentials(ctx, "my-service")

// List all credentials
names, err := mgr.ListCredentials(ctx)

// Delete credentials
err := mgr.DeleteCredentials(ctx, "my-service")

// Get an authenticated HTTP client
client, err := mgr.GetClient(ctx, "my-service")

// Get the OAuth2 token directly
token, err := mgr.GetToken(ctx, "my-service")
```
