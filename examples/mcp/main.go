// Package main demonstrates using omnitoken with MCP servers.
//
// This example shows how omnitoken can provide credentials to MCP server tools.
package main

import (
	"context"
	"fmt"
	"log"
	"log/slog"
	"net/http"
	"os"

	"github.com/grokify/goauth"
	"github.com/plexusone/omnitoken"
	"github.com/plexusone/omnivault/providers/memory"
)

// ToolContext provides credentials to MCP tools.
// This would be injected into tool handlers in omniskill.
type ToolContext struct {
	tokenMgr *omnitoken.TokenManager
	logger   *slog.Logger
}

// GetClient returns an authenticated HTTP client for the named service.
// MCP tools can use this to make authenticated API calls.
func (tc *ToolContext) GetClient(ctx context.Context, serviceName string) (*http.Client, error) {
	return tc.tokenMgr.GetClient(ctx, serviceName)
}

// GetCredentials returns credentials for inspection or custom auth flows.
func (tc *ToolContext) GetCredentials(ctx context.Context, serviceName string) (*goauth.Credentials, error) {
	return tc.tokenMgr.GetCredentials(ctx, serviceName)
}

func main() {
	ctx := context.Background()
	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

	// In production, use a real vault backend.
	// Example: 1Password via op:// URIs, HashiCorp Vault, etc.
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

	// Pre-populate credentials (in production, these would be loaded from vault)
	services := map[string]*goauth.Credentials{
		"github": {
			Service: "github",
			Type:    goauth.TypeOAuth2,
			OAuth2: &goauth.CredentialsOAuth2{
				ClientID:     os.Getenv("GITHUB_CLIENT_ID"),
				ClientSecret: os.Getenv("GITHUB_CLIENT_SECRET"),
				GrantType:    "client_credentials",
				Scopes:       []string{"repo", "user"},
			},
		},
		"slack": {
			Service: "slack",
			Type:    goauth.TypeOAuth2,
			OAuth2: &goauth.CredentialsOAuth2{
				ClientID:     os.Getenv("SLACK_CLIENT_ID"),
				ClientSecret: os.Getenv("SLACK_CLIENT_SECRET"),
				GrantType:    "client_credentials",
			},
		},
		"internal-api": {
			Service: "internal-api",
			Type:    goauth.TypeBasic,
			Basic: &goauth.CredentialsBasicAuth{
				Username:  os.Getenv("API_USERNAME"),
				Password:  os.Getenv("API_PASSWORD"),
				ServerURL: os.Getenv("API_SERVER_URL"),
			},
		},
	}

	for name, creds := range services {
		if err := mgr.SetCredentials(ctx, name, creds); err != nil {
			logger.Warn("failed to store credentials", "name", name, "error", err)
		}
	}

	// Create tool context for MCP tools
	toolCtx := &ToolContext{
		tokenMgr: mgr,
		logger:   logger,
	}

	// Example: Tool that uses GitHub API
	fmt.Println("=== GitHub Tool Example ===")
	if client, err := toolCtx.GetClient(ctx, "github"); err != nil {
		fmt.Printf("Failed to get GitHub client: %v\n", err)
	} else {
		fmt.Printf("Got GitHub client: %T\n", client)
		// In real code: resp, err := client.Get("https://api.github.com/user/repos")
	}

	// Example: Tool that uses internal API with basic auth
	fmt.Println("\n=== Internal API Tool Example ===")
	if creds, err := toolCtx.GetCredentials(ctx, "internal-api"); err != nil {
		fmt.Printf("Failed to get internal API credentials: %v\n", err)
	} else {
		fmt.Printf("Got credentials for: %s (type: %s)\n", creds.Service, creds.Type)
	}

	// List all available services
	fmt.Println("\n=== Available Services ===")
	names, err := mgr.ListCredentials(ctx)
	if err != nil {
		log.Fatalf("failed to list credentials: %v", err)
	}
	for _, name := range names {
		creds, _ := mgr.GetCredentials(ctx, name)
		fmt.Printf("- %s (type: %s)\n", name, creds.Type)
	}

	fmt.Println("\nDone!")
}
