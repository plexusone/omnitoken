# Basic Usage Example

This example demonstrates basic usage of omnitoken with an in-memory vault.

## Full Example

```go
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"os"

	"github.com/grokify/goauth"
	"github.com/plexusone/omnitoken"
	"github.com/plexusone/omnivault/providers/memory"
)

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// Create an in-memory vault for demonstration.
	// In production, use a real vault backend like 1Password, HashiCorp Vault, etc.
	vault := memory.New()

	// Create the token manager
	mgr, err := omnitoken.New(omnitoken.Config{
		Vault:  vault,
		Logger: logger,
	})
	if err != nil {
		log.Fatalf("failed to create token manager: %v", err)
	}
	defer func() {
		if err := mgr.Close(); err != nil {
			logger.Error("failed to close token manager", "error", err)
		}
	}()

	// Store OAuth2 credentials for a service
	creds := &goauth.Credentials{
		Service: "github",
		Type:    goauth.TypeOAuth2,
		OAuth2: &goauth.CredentialsOAuth2{
			ClientID:     "your-client-id",
			ClientSecret: "your-client-secret",
			GrantType:    "client_credentials",
			Scopes:       []string{"repo", "user"},
		},
	}

	if err := mgr.SetCredentials(ctx, "my-github-app", creds); err != nil {
		log.Fatalf("failed to store credentials: %v", err)
	}

	// List all stored credentials
	names, err := mgr.ListCredentials(ctx)
	if err != nil {
		log.Fatalf("failed to list credentials: %v", err)
	}
	fmt.Printf("Stored credentials: %v\n", names)

	// Retrieve credentials
	retrieved, err := mgr.GetCredentials(ctx, "my-github-app")
	if err != nil {
		log.Fatalf("failed to get credentials: %v", err)
	}
	fmt.Printf("Retrieved: service=%s, type=%s\n", retrieved.Service, retrieved.Type)

	// In a real scenario, you would get an authenticated client:
	// client, err := mgr.GetClient(ctx, "my-github-app")
	// if err != nil {
	//     log.Fatalf("failed to get client: %v", err)
	// }
	// resp, err := client.Get("https://api.github.com/user")

	fmt.Println("Done!")
}
```

## Running the Example

```bash
cd examples/basic
go run main.go
```

## Expected Output

```
Stored credentials: [my-github-app]
Retrieved: service=github, type=oauth2
Done!
```

## Key Points

1. **In-memory vault**: Good for testing, but use a real vault in production
2. **Credentials storage**: Store goauth.Credentials objects by name
3. **Retrieval**: Get credentials back by name for use with APIs
4. **Client creation**: Use `GetClient()` to get an authenticated HTTP client
