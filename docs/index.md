# OmniToken

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

## Key Features

| Feature | Description |
|---------|-------------|
| **Vault Integration** | Store credentials in 1Password, Bitwarden, Keeper, and other backends |
| **Token Lifecycle** | Automatic token acquisition, refresh, and caching |
| **Multiple Auth Types** | OAuth2, JWT, Basic Auth, API keys, GCP service accounts |
| **MCP Ready** | Designed for use in MCP servers |
| **goauth Compatible** | Implements goauth TokenSet interface |

## Next Steps

- [Installation](installation.md) - Install the library
- [Quick Start](quickstart.md) - Get started with basic usage
- [Credential Sources](credential-sources.md) - Learn about different credential sources
- [API Reference](api.md) - Full API documentation
- [MCP Integration](mcp-integration.md) - Use with MCP servers
