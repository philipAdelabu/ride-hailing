package jwtkeys

import (
	"context"
	"time"

	"github.com/richxcame/ride-hailing/pkg/config"
)

// NewManagerFromConfig builds a Manager using the shared JWT configuration.
func NewManagerFromConfig(ctx context.Context, cfg config.JWTConfig, readOnly bool) (*Manager, error) {
	managerCfg := Config{
		KeyFilePath:      cfg.KeyFile,
		RotationInterval: time.Duration(cfg.RotationHours) * time.Hour,
		GracePeriod:      time.Duration(cfg.GraceHours) * time.Hour,
		LegacySecret:     cfg.Secret,
		ReadOnly:         readOnly,
	}

	if cfg.VaultPath != "" && cfg.VaultAddress != "" && cfg.VaultToken != "" {
		store, err := newVaultStore(VaultConfig{
			Address:   cfg.VaultAddress,
			Token:     cfg.VaultToken,
			Path:      cfg.VaultPath,
			Namespace: cfg.VaultNamespace,
		})
		if err != nil {
			return nil, err
		}
		managerCfg.Store = store
		managerCfg.KeyFilePath = ""
	}

	return NewManager(ctx, managerCfg)
}
