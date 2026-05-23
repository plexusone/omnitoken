package omnitoken

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/grokify/goauth"
	"github.com/plexusone/omnivault/vault"
)

// CredentialsStore manages goauth Credentials in a vault.
type CredentialsStore struct {
	vault  vault.Vault
	prefix string
	logger *slog.Logger
}

// NewCredentialsStore creates a new CredentialsStore.
func NewCredentialsStore(v vault.Vault, prefix string, logger *slog.Logger) *CredentialsStore {
	if prefix == "" {
		prefix = "credentials/"
	}
	if logger == nil {
		logger = slog.Default()
	}
	return &CredentialsStore{
		vault:  v,
		prefix: prefix,
		logger: logger,
	}
}

// Get retrieves credentials from the vault by name.
func (cs *CredentialsStore) Get(ctx context.Context, name string) (*goauth.Credentials, error) {
	path := cs.nameToPat(name)

	secret, err := cs.vault.Get(ctx, path)
	if err != nil {
		if isNotFoundError(err) {
			return nil, fmt.Errorf("%w: %s", ErrCredentialsNotFound, name)
		}
		return nil, fmt.Errorf("failed to get credentials: %w", err)
	}

	creds, err := parseCredentials(secret)
	if err != nil {
		return nil, fmt.Errorf("failed to parse credentials: %w", err)
	}

	// Inflate endpoint information based on service name
	if err := creds.Inflate(); err != nil {
		cs.logger.Warn("failed to inflate credentials",
			"name", name,
			"error", err,
		)
	}

	cs.logger.Debug("retrieved credentials",
		"name", name,
		"service", creds.Service,
		"type", creds.Type,
	)

	return creds, nil
}

// Set stores credentials in the vault.
func (cs *CredentialsStore) Set(ctx context.Context, name string, creds *goauth.Credentials) error {
	path := cs.nameToPat(name)

	secret, err := credentialsToSecret(creds)
	if err != nil {
		return fmt.Errorf("failed to serialize credentials: %w", err)
	}

	if err := cs.vault.Set(ctx, path, secret); err != nil {
		return fmt.Errorf("failed to store credentials: %w", err)
	}

	cs.logger.Debug("stored credentials",
		"name", name,
		"service", creds.Service,
		"type", creds.Type,
	)

	return nil
}

// Delete removes credentials from the vault.
func (cs *CredentialsStore) Delete(ctx context.Context, name string) error {
	path := cs.nameToPat(name)

	if err := cs.vault.Delete(ctx, path); err != nil {
		return fmt.Errorf("failed to delete credentials: %w", err)
	}

	cs.logger.Debug("deleted credentials", "name", name)
	return nil
}

// Exists checks if credentials exist in the vault.
func (cs *CredentialsStore) Exists(ctx context.Context, name string) (bool, error) {
	path := cs.nameToPat(name)
	return cs.vault.Exists(ctx, path)
}

// List returns all credential names in the vault.
func (cs *CredentialsStore) List(ctx context.Context) ([]string, error) {
	paths, err := cs.vault.List(ctx, cs.prefix)
	if err != nil {
		return nil, fmt.Errorf("failed to list credentials: %w", err)
	}

	names := make([]string, 0, len(paths))
	for _, path := range paths {
		name := cs.pathToName(path)
		if name != "" {
			names = append(names, name)
		}
	}

	return names, nil
}

// nameToPat converts a credential name to a vault path.
func (cs *CredentialsStore) nameToPat(name string) string {
	return cs.prefix + sanitizeKey(name)
}

// pathToName converts a vault path to a credential name.
func (cs *CredentialsStore) pathToName(path string) string {
	if !strings.HasPrefix(path, cs.prefix) {
		return ""
	}
	return strings.TrimPrefix(path, cs.prefix)
}

// parseCredentials parses a vault secret into goauth Credentials.
func parseCredentials(secret *vault.Secret) (*goauth.Credentials, error) {
	var creds goauth.Credentials

	// First try to unmarshal from the Value field (full JSON)
	if secret.Value != "" {
		if err := json.Unmarshal([]byte(secret.Value), &creds); err == nil {
			return &creds, nil
		}
	}

	// Fall back to Fields for multi-field secrets (e.g., from password managers)
	if secret.Fields != nil {
		creds = goauth.Credentials{
			Service: secret.Fields["service"],
			Type:    secret.Fields["type"],
		}

		// Handle different credential types based on fields
		switch creds.Type {
		case goauth.TypeBasic:
			creds.Basic = &goauth.CredentialsBasicAuth{
				Username:  secret.Fields["username"],
				Password:  secret.Fields["password"],
				ServerURL: secret.Fields["server_url"],
			}
		case goauth.TypeHeaderQuery:
			creds.HeaderQuery = &goauth.CredentialsHeaderQuery{
				ServerURL: secret.Fields["server_url"],
			}
			// Parse header/query params from JSON fields if present
			if headersJSON := secret.Fields["headers"]; headersJSON != "" {
				var headers map[string][]string
				if err := json.Unmarshal([]byte(headersJSON), &headers); err == nil {
					creds.HeaderQuery.Header = headers
				}
			}
		case goauth.TypeOAuth2:
			creds.OAuth2 = &goauth.CredentialsOAuth2{
				ClientID:     secret.Fields["client_id"],
				ClientSecret: secret.Fields["client_secret"],
				ServerURL:    secret.Fields["server_url"],
				RedirectURL:  secret.Fields["redirect_url"],
				GrantType:    secret.Fields["grant_type"],
			}
			if scopesStr := secret.Fields["scopes"]; scopesStr != "" {
				creds.OAuth2.Scopes = strings.Split(scopesStr, " ")
			}
		case goauth.TypeJWT:
			creds.JWT = &goauth.CredentialsJWT{
				Issuer:        secret.Fields["issuer"],
				PrivateKey:    secret.Fields["private_key"],
				SigningMethod: secret.Fields["signing_method"],
			}
		default:
			// Try to detect type from available fields
			if secret.Fields["client_id"] != "" || secret.Fields["client_secret"] != "" {
				creds.Type = goauth.TypeOAuth2
				creds.OAuth2 = &goauth.CredentialsOAuth2{
					ClientID:     secret.Fields["client_id"],
					ClientSecret: secret.Fields["client_secret"],
					ServerURL:    secret.Fields["server_url"],
					RedirectURL:  secret.Fields["redirect_url"],
					GrantType:    secret.Fields["grant_type"],
				}
			} else if secret.Fields["username"] != "" && secret.Fields["password"] != "" {
				creds.Type = goauth.TypeBasic
				creds.Basic = &goauth.CredentialsBasicAuth{
					Username:  secret.Fields["username"],
					Password:  secret.Fields["password"],
					ServerURL: secret.Fields["server_url"],
				}
			}
		}

		return &creds, nil
	}

	return nil, fmt.Errorf("secret has no parseable credentials")
}

// credentialsToSecret converts goauth Credentials to a vault secret.
func credentialsToSecret(creds *goauth.Credentials) (*vault.Secret, error) {
	data, err := json.Marshal(creds)
	if err != nil {
		return nil, err
	}

	// Also store key fields for providers that support multi-field
	fields := make(map[string]string)
	fields["service"] = creds.Service
	fields["type"] = creds.Type

	switch creds.Type {
	case goauth.TypeBasic:
		if creds.Basic != nil {
			fields["username"] = creds.Basic.Username
			fields["password"] = creds.Basic.Password
			fields["server_url"] = creds.Basic.ServerURL
		}
	case goauth.TypeOAuth2:
		if creds.OAuth2 != nil {
			fields["client_id"] = creds.OAuth2.ClientID
			fields["client_secret"] = creds.OAuth2.ClientSecret
			fields["server_url"] = creds.OAuth2.ServerURL
			fields["redirect_url"] = creds.OAuth2.RedirectURL
			fields["grant_type"] = creds.OAuth2.GrantType
			if len(creds.OAuth2.Scopes) > 0 {
				fields["scopes"] = strings.Join(creds.OAuth2.Scopes, " ")
			}
		}
	case goauth.TypeJWT:
		if creds.JWT != nil {
			fields["issuer"] = creds.JWT.Issuer
			fields["private_key"] = creds.JWT.PrivateKey
			fields["signing_method"] = creds.JWT.SigningMethod
		}
	case goauth.TypeHeaderQuery:
		if creds.HeaderQuery != nil {
			fields["server_url"] = creds.HeaderQuery.ServerURL
			if creds.HeaderQuery.Header != nil {
				headersJSON, _ := json.Marshal(creds.HeaderQuery.Header)
				fields["headers"] = string(headersJSON)
			}
		}
	}

	now := vault.Now()
	return &vault.Secret{
		Value:  string(data),
		Fields: fields,
		Metadata: vault.Metadata{
			ModifiedAt: now,
			Tags: map[string]string{
				"type":    "goauth_credentials",
				"service": creds.Service,
				"subtype": creds.Type,
			},
		},
	}, nil
}

// CredentialsSetFromVault loads a goauth.CredentialsSet from multiple vault entries.
func (cs *CredentialsStore) LoadCredentialsSet(ctx context.Context) (*goauth.CredentialsSet, error) {
	names, err := cs.List(ctx)
	if err != nil {
		return nil, err
	}

	set := &goauth.CredentialsSet{
		Credentials: make(map[string]goauth.Credentials),
	}

	for _, name := range names {
		creds, err := cs.Get(ctx, name)
		if err != nil {
			cs.logger.Warn("failed to load credentials",
				"name", name,
				"error", err,
			)
			continue
		}
		set.Credentials[name] = *creds
	}

	return set, nil
}

// SaveCredentialsSet saves a goauth.CredentialsSet to the vault.
func (cs *CredentialsStore) SaveCredentialsSet(ctx context.Context, set *goauth.CredentialsSet) error {
	for name, creds := range set.Credentials {
		credsCopy := creds
		if err := cs.Set(ctx, name, &credsCopy); err != nil {
			return fmt.Errorf("failed to save credentials %s: %w", name, err)
		}
	}
	return nil
}

// GetWithTimeout retrieves credentials with a timeout.
func (cs *CredentialsStore) GetWithTimeout(name string, timeout time.Duration) (*goauth.Credentials, error) {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	return cs.Get(ctx, name)
}
