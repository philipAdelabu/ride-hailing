package jwtkeys

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	vault "github.com/hashicorp/vault/api"
)

const defaultVaultSecret = "jwt_keys"

// VaultConfig captures the settings required to connect to HashiCorp Vault.
type VaultConfig struct {
	Address   string
	Token     string
	Path      string
	Namespace string
}

// newVaultStore creates a Store backed by HashiCorp Vault's KV engine.
func newVaultStore(cfg VaultConfig) (Store, error) {
	if cfg.Address == "" || cfg.Token == "" || cfg.Path == "" {
		return nil, fmt.Errorf("vault store requires address, token, and path")
	}

	mount, secret, err := normalizeVaultPath(cfg.Path)
	if err != nil {
		return nil, err
	}

	clientCfg := vault.DefaultConfig()
	clientCfg.Address = cfg.Address

	client, err := vault.NewClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create vault client: %w", err)
	}

	client.SetToken(cfg.Token)
	if cfg.Namespace != "" {
		client.SetNamespace(cfg.Namespace)
	}

	return &vaultStore{
		client:     client,
		mountPath:  mount,
		secretPath: secret,
	}, nil
}

type vaultStore struct {
	client     *vault.Client
	mountPath  string
	secretPath string
}

func (s *vaultStore) Load(ctx context.Context) ([]SigningKey, error) {
	secret, err := s.client.KVv2(s.mountPath).Get(ctx, s.secretPath)
	if err != nil {
		var respErr *vault.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return nil, nil
		}
		return nil, err
	}

	raw, ok := secret.Data["keys"].(string)
	if !ok || raw == "" {
		return nil, nil
	}

	var keys []SigningKey
	if err := json.Unmarshal([]byte(raw), &keys); err != nil {
		return nil, err
	}
	return keys, nil
}

func (s *vaultStore) Save(ctx context.Context, keys []SigningKey) error {
	payload, err := json.Marshal(keys)
	if err != nil {
		return err
	}

	data := map[string]interface{}{"keys": string(payload)}
	_, err = s.client.KVv2(s.mountPath).Put(ctx, s.secretPath, data)
	return err
}

func normalizeVaultPath(path string) (mount string, secret string, err error) {
	clean := strings.Trim(path, "/")
	if clean == "" {
		return "", "", fmt.Errorf("vault path must include mount point and secret")
	}

	parts := strings.SplitN(clean, "/", 2)
	mount = parts[0]
	if mount == "" {
		return "", "", fmt.Errorf("vault path missing mount point")
	}

	if len(parts) == 1 || parts[1] == "" {
		return mount, defaultVaultSecret, nil
	}

	secret = strings.TrimPrefix(parts[1], "data/")
	secret = strings.TrimPrefix(secret, "metadata/")
	if secret == "" {
		secret = defaultVaultSecret
	}

	return mount, secret, nil
}
