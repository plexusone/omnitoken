# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html),
and commits follow [Conventional Commits](https://www.conventionalcommits.org/en/v1.0.0/).

## [Unreleased]

## [v0.1.0] - 2026-05-23

### Highlights

- Initial release of OmniToken SDK
- Bridge goauth credentials with vault-based storage via omnivault
- Full token lifecycle management

### Added

- Core `TokenManager` for credential and token operations
- `CredentialsStore` implementing vault-backed goauth credential storage
- `VaultTokenSet` implementing goauth TokenSet interface
- Multiple convenience constructors:
  - `NewFromVaultURI` - Create from vault URI
  - `NewFromCredentialsFile` - Load from goauth CredentialsSet file
  - `NewFromCredentialsSet` - Create from in-memory CredentialsSet
  - `NewFromCredentials` - Create with single credential
  - `NewFromEnv` - Load from environment variables
  - `NewFromDirectory` - Load from file directory
  - `NewAuto` - Auto-detect from environment
- Support for all goauth credential types:
  - OAuth2 (client credentials, authorization code, etc.)
  - JWT bearer tokens
  - HTTP Basic Auth
  - Custom header/query authentication
  - Google Cloud service accounts
- Automatic token refresh with configurable buffer
- Google service account loading helper
- MCP server integration examples

[unreleased]: https://github.com/plexusone/omnitoken/compare/v0.1.0...HEAD
[v0.1.0]: https://github.com/plexusone/omnitoken/releases/tag/v0.1.0
