# MCP Integration

OmniToken is designed for use in MCP (Model Context Protocol) servers that need to access authenticated APIs.

## Basic MCP Server Pattern

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

## Environment Variables

Configure your MCP server with these environment variables:

| Variable | Description |
|----------|-------------|
| `OMNITOKEN_VAULT_URI` | Vault URI (e.g., `op://MyVault`) |
| `OMNITOKEN_CREDENTIALS_NAME` | Name of the credential to use |

## Tool Context Pattern

Create a context object that provides credentials to MCP tools:

```go
// ToolContext provides credentials to MCP tools.
type ToolContext struct {
    tokenMgr *omnitoken.TokenManager
    logger   *slog.Logger
}

// GetClient returns an authenticated HTTP client for the named service.
func (tc *ToolContext) GetClient(ctx context.Context, serviceName string) (*http.Client, error) {
    return tc.tokenMgr.GetClient(ctx, serviceName)
}

// GetCredentials returns credentials for inspection or custom auth flows.
func (tc *ToolContext) GetCredentials(ctx context.Context, serviceName string) (*goauth.Credentials, error) {
    return tc.tokenMgr.GetCredentials(ctx, serviceName)
}
```

## Multi-Service MCP Server

For MCP servers that connect to multiple services:

```go
mgr, err := omnitoken.NewFromVaultURI("op://MCPCredentials")
if err != nil {
    log.Fatal(err)
}
defer mgr.Close()

// Each tool can request its own service client
githubClient, _ := mgr.GetClient(ctx, "github")
slackClient, _ := mgr.GetClient(ctx, "slack")
jiraClient, _ := mgr.GetClient(ctx, "jira")
```

## Example: mcp-google

See [mcp-google](https://github.com/plexusone/mcp-google) for a complete example of OmniToken used in an MCP server for Google APIs:

```go
import (
    "github.com/plexusone/omnitoken"
    _ "github.com/plexusone/omnivault-desktop"
    "google.golang.org/api/slides/v1"
)

func main() {
    mgr, _ := omnitoken.NewFromVaultURI(os.Getenv("OMNITOKEN_VAULT_URI"))
    defer mgr.Close()

    client, _ := mgr.GetClient(ctx, "google")

    // Use with Google APIs
    slidesService, _ := slides.NewService(ctx, option.WithHTTPClient(client))
}
```

## Claude Desktop Configuration

Example `claude_desktop_config.json` for MCP server with OmniToken:

```json
{
  "mcpServers": {
    "google": {
      "command": "/path/to/mcp-google",
      "env": {
        "OMNITOKEN_VAULT_URI": "op://MCPCredentials",
        "OMNITOKEN_CREDENTIALS_NAME": "google-workspace"
      }
    }
  }
}
```
