# Installation

## Go Library

```bash
go get github.com/plexusone/omnitoken
```

## Optional: Vault Providers

To use password manager backends (1Password, Bitwarden, Keeper), install the omnivault-desktop module:

```bash
go get github.com/plexusone/omnivault-desktop
```

Then import it in your code:

```go
import _ "github.com/plexusone/omnivault-desktop"
```

## Requirements

- Go 1.26 or later
- For vault backends: appropriate CLI tools or API tokens configured

## Dependencies

OmniToken depends on:

- [goauth](https://github.com/grokify/goauth) - Credential types and token management
- [omnivault](https://github.com/plexusone/omnivault) - Vault abstraction layer
- [golang.org/x/oauth2](https://pkg.go.dev/golang.org/x/oauth2) - OAuth2 token handling
