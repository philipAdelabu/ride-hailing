package secrets

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	vault "github.com/hashicorp/vault/api"
)

// VaultConfig stores the configuration required for HashiCorp Vault.
type VaultConfig struct {
	Address       string
	Token         string
	Namespace     string
	MountPath     string
	CACert        string
	CAPath        string
	ClientCert    string
	ClientKey     string
	TLSSkipVerify bool
}

type vaultProvider struct {
	client       *vault.Client
	defaultMount string
}

func newVaultProvider(cfg VaultConfig) (provider, error) {
	if cfg.Address == "" || cfg.Token == "" {
		return nil, fmt.Errorf("secrets: vault provider requires address and token")
	}

	if cfg.MountPath == "" {
		cfg.MountPath = "secret"
	}

	clientCfg := vault.DefaultConfig()
	clientCfg.Address = cfg.Address

	tlsCfg := &vault.TLSConfig{
		CACert:     cfg.CACert,
		CAPath:     cfg.CAPath,
		ClientCert: cfg.ClientCert,
		ClientKey:  cfg.ClientKey,
		Insecure:   cfg.TLSSkipVerify,
	}
	if cfg.CACert != "" || cfg.CAPath != "" || cfg.ClientCert != "" || cfg.ClientKey != "" || cfg.TLSSkipVerify {
		if err := clientCfg.ConfigureTLS(tlsCfg); err != nil {
			return nil, fmt.Errorf("secrets: failed to configure vault TLS: %w", err)
		}
	}

	client, err := vault.NewClient(clientCfg)
	if err != nil {
		return nil, fmt.Errorf("secrets: failed to create vault client: %w", err)
	}

	client.SetToken(cfg.Token)
	if cfg.Namespace != "" {
		client.SetNamespace(cfg.Namespace)
	}

	return &vaultProvider{
		client:       client,
		defaultMount: strings.Trim(cfg.MountPath, "/"),
	}, nil
}

func (v *vaultProvider) Name() ProviderType {
	return ProviderVault
}

func (v *vaultProvider) Close() error {
	// Vault client does not expose a close operation.
	return nil
}

func (v *vaultProvider) Fetch(ctx context.Context, ref Reference) (Secret, error) {
	mount, path := v.resolvePath(ref)
	if mount == "" || path == "" {
		return Secret{}, ErrInvalidReference
	}

	kv := v.client.KVv2(mount)

	var (
		secret *vault.KVSecret
		err    error
	)

	if ref.Version != "" {
		version, convErr := strconv.Atoi(ref.Version)
		if convErr != nil {
			return Secret{}, fmt.Errorf("secrets: invalid vault version %q: %w", ref.Version, convErr)
		}
		secret, err = kv.GetVersion(ctx, path, version)
	} else {
		secret, err = kv.Get(ctx, path)
	}

	if err != nil {
		var respErr *vault.ResponseError
		if errors.As(err, &respErr) && respErr.StatusCode == http.StatusNotFound {
			return Secret{}, fmt.Errorf("secrets: vault path %s not found", ref.Path)
		}
		return Secret{}, err
	}

	payload := make(map[string]string, len(secret.Data))
	for k, vRaw := range secret.Data {
		payload[k] = fmt.Sprint(vRaw)
	}

	metadata := Metadata{}
	if secret.VersionMetadata != nil {
		metadata.Version = fmt.Sprintf("%d", secret.VersionMetadata.Version)
		metadata.CreatedAt = secret.VersionMetadata.CreatedTime
		metadata.UpdatedAt = secret.VersionMetadata.CreatedTime
	}

	return Secret{
		Data:     payload,
		Metadata: metadata,
	}, nil
}

func (v *vaultProvider) resolvePath(ref Reference) (string, string) {
	mount := v.defaultMount
	if ref.Mount != "" {
		mount = strings.Trim(ref.Mount, "/")
	}

	clean := strings.Trim(ref.Path, "/")
	if clean == "" {
		return mount, ""
	}

	secretPath := strings.TrimPrefix(clean, "data/")
	secretPath = strings.TrimPrefix(secretPath, "metadata/")
	return mount, strings.Trim(secretPath, "/")
}
