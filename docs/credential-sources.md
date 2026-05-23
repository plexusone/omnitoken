# Credential Sources

OmniToken supports multiple credential sources through various constructors.

## Constructor Reference

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

## Using Password Managers

To use 1Password, Bitwarden, or Keeper, import [omnivault-desktop](https://github.com/plexusone/omnivault-desktop):

```go
import _ "github.com/plexusone/omnivault-desktop"
```

### 1Password

```go
// Requires: OP_SERVICE_ACCOUNT_TOKEN environment variable
mgr, err := omnitoken.NewFromVaultURI("op://MyVault")
```

### Bitwarden

```go
// Requires: BW_ACCESS_TOKEN, BW_ORGANIZATION_ID environment variables
mgr, err := omnitoken.NewFromVaultURI("bw://my-org-id")
```

### Keeper

```go
// Requires: KSM_TOKEN or KSM_CONFIG environment variable
mgr, err := omnitoken.NewFromVaultURI("keeper://")
```

## Environment Variables

| Variable | Description |
|----------|-------------|
| `OMNITOKEN_VAULT_URI` | Vault URI for `NewAuto()` |
| `OMNITOKEN_CREDENTIALS_FILE` | Credentials file path for `NewAuto()` |
| `OMNITOKEN_CREDENTIALS_NAME` | Default credential name (used by MCP servers) |

## Using NewFromEnv

The `NewFromEnv` constructor reads credentials from environment variables with a specified prefix:

```go
// Reads from environment variables starting with "MYAPP_"
mgr, err := omnitoken.NewFromEnv("MYAPP_")
```

## Using NewFromDirectory

Store credentials as individual JSON files in a directory:

```go
// Each file in /path/to/creds/ is a separate credential
mgr, err := omnitoken.NewFromDirectory("/path/to/creds")
```
