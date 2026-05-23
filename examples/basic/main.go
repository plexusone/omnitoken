// Package main demonstrates basic usage of omnitoken.
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
