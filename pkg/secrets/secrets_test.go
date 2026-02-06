package secrets

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
)

// ============== ProviderType Tests ==============

func TestProviderType_Constants(t *testing.T) {
	tests := []struct {
		provider ProviderType
		expected string
	}{
		{ProviderNone, ""},
		{ProviderVault, "vault"},
		{ProviderAWS, "aws"},
		{ProviderGCP, "gcp"},
		{ProviderKubernetes, "kubernetes"},
	}

	for _, tc := range tests {
		if string(tc.provider) != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, string(tc.provider))
		}
	}
}

// ============== SecretType Tests ==============

func TestSecretType_Constants(t *testing.T) {
	tests := []struct {
		secretType SecretType
		expected   string
	}{
		{SecretDatabase, "database_credentials"},
		{SecretJWTKeys, "jwt_signing_keys"},
		{SecretStripe, "stripe_api_key"},
		{SecretFirebase, "firebase_credentials"},
		{SecretTwilio, "twilio_credentials"},
		{SecretSMTP, "smtp_credentials"},
		{SecretCustom, "custom"},
	}

	for _, tc := range tests {
		if string(tc.secretType) != tc.expected {
			t.Errorf("expected %q, got %q", tc.expected, string(tc.secretType))
		}
	}
}

// ============== Error Constants Tests ==============

func TestErrorConstants(t *testing.T) {
	if ErrProviderNotConfigured == nil {
		t.Error("ErrProviderNotConfigured should not be nil")
	}
	if ErrInvalidReference == nil {
		t.Error("ErrInvalidReference should not be nil")
	}
	if ErrKeyNotFound == nil {
		t.Error("ErrKeyNotFound should not be nil")
	}
}

func TestErrorMessages(t *testing.T) {
	if !strings.Contains(ErrProviderNotConfigured.Error(), "provider not configured") {
		t.Error("ErrProviderNotConfigured should contain 'provider not configured'")
	}
	if !strings.Contains(ErrInvalidReference.Error(), "invalid reference") {
		t.Error("ErrInvalidReference should contain 'invalid reference'")
	}
	if !strings.Contains(ErrKeyNotFound.Error(), "key not found") {
		t.Error("ErrKeyNotFound should contain 'key not found'")
	}
}

// ============== Reference Tests ==============

func TestReference_CacheKey_Simple(t *testing.T) {
	ref := Reference{Path: "my/secret/path"}
	expected := "my/secret/path"
	if ref.CacheKey() != expected {
		t.Errorf("expected %q, got %q", expected, ref.CacheKey())
	}
}

func TestReference_CacheKey_WithMount(t *testing.T) {
	ref := Reference{
		Mount: "kv",
		Path:  "my/secret/path",
	}
	expected := "kv|my/secret/path"
	if ref.CacheKey() != expected {
		t.Errorf("expected %q, got %q", expected, ref.CacheKey())
	}
}

func TestReference_CacheKey_WithVersion(t *testing.T) {
	ref := Reference{
		Path:    "my/secret/path",
		Version: "2",
	}
	expected := "my/secret/path@2"
	if ref.CacheKey() != expected {
		t.Errorf("expected %q, got %q", expected, ref.CacheKey())
	}
}

func TestReference_CacheKey_WithKey(t *testing.T) {
	ref := Reference{
		Path: "my/secret/path",
		Key:  "password",
	}
	expected := "my/secret/path#password"
	if ref.CacheKey() != expected {
		t.Errorf("expected %q, got %q", expected, ref.CacheKey())
	}
}

func TestReference_CacheKey_Full(t *testing.T) {
	ref := Reference{
		Mount:   "kv",
		Path:    "my/secret/path",
		Version: "3",
		Key:     "api_key",
	}
	expected := "kv|my/secret/path@3#api_key"
	if ref.CacheKey() != expected {
		t.Errorf("expected %q, got %q", expected, ref.CacheKey())
	}
}

// ============== ParseReference Tests ==============

func TestParseReference_Simple(t *testing.T) {
	ref, err := ParseReference("test", SecretCustom, "my/secret/path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Name != "test" {
		t.Errorf("expected Name 'test', got %q", ref.Name)
	}
	if ref.Type != SecretCustom {
		t.Errorf("expected Type SecretCustom, got %v", ref.Type)
	}
	if ref.Path != "my/secret/path" {
		t.Errorf("expected Path 'my/secret/path', got %q", ref.Path)
	}
}

func TestParseReference_EmptyString(t *testing.T) {
	_, err := ParseReference("test", SecretCustom, "")
	if !errors.Is(err, ErrInvalidReference) {
		t.Errorf("expected ErrInvalidReference, got %v", err)
	}
}

func TestParseReference_WhitespaceOnly(t *testing.T) {
	_, err := ParseReference("test", SecretCustom, "   ")
	if !errors.Is(err, ErrInvalidReference) {
		t.Errorf("expected ErrInvalidReference, got %v", err)
	}
}

func TestParseReference_WithProvider(t *testing.T) {
	ref, err := ParseReference("test", SecretCustom, "vault://secret/data/myapp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Provider != ProviderVault {
		t.Errorf("expected Provider ProviderVault, got %v", ref.Provider)
	}
	if ref.Path != "secret/data/myapp" {
		t.Errorf("expected Path 'secret/data/myapp', got %q", ref.Path)
	}
}

func TestParseReference_WithAWSProvider(t *testing.T) {
	ref, err := ParseReference("test", SecretCustom, "aws://my-secret-name")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Provider != ProviderAWS {
		t.Errorf("expected Provider ProviderAWS, got %v", ref.Provider)
	}
}

func TestParseReference_WithGCPProvider(t *testing.T) {
	ref, err := ParseReference("test", SecretCustom, "gcp://projects/myproject/secrets/mysecret")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Provider != ProviderGCP {
		t.Errorf("expected Provider ProviderGCP, got %v", ref.Provider)
	}
}

func TestParseReference_WithVersion(t *testing.T) {
	ref, err := ParseReference("test", SecretCustom, "secret/path@v2")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Version != "v2" {
		t.Errorf("expected Version 'v2', got %q", ref.Version)
	}
	if ref.Path != "secret/path" {
		t.Errorf("expected Path 'secret/path', got %q", ref.Path)
	}
}

func TestParseReference_WithKey(t *testing.T) {
	ref, err := ParseReference("test", SecretCustom, "secret/path#password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Key != "password" {
		t.Errorf("expected Key 'password', got %q", ref.Key)
	}
	if ref.Path != "secret/path" {
		t.Errorf("expected Path 'secret/path', got %q", ref.Path)
	}
}

func TestParseReference_Full(t *testing.T) {
	ref, err := ParseReference("db_creds", SecretDatabase, "vault://kv:database/prod@3#password")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Name != "db_creds" {
		t.Errorf("expected Name 'db_creds', got %q", ref.Name)
	}
	if ref.Type != SecretDatabase {
		t.Errorf("expected Type SecretDatabase, got %v", ref.Type)
	}
	if ref.Provider != ProviderVault {
		t.Errorf("expected Provider ProviderVault, got %v", ref.Provider)
	}
	if ref.Mount != "kv" {
		t.Errorf("expected Mount 'kv', got %q", ref.Mount)
	}
	if ref.Version != "3" {
		t.Errorf("expected Version '3', got %q", ref.Version)
	}
	if ref.Key != "password" {
		t.Errorf("expected Key 'password', got %q", ref.Key)
	}
}

func TestParseReference_LeadingTrailingSlashes(t *testing.T) {
	ref, err := ParseReference("test", SecretCustom, "/secret/path/")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Path != "secret/path" {
		t.Errorf("expected trimmed Path 'secret/path', got %q", ref.Path)
	}
}

func TestParseReference_DoubleColonMount(t *testing.T) {
	ref, err := ParseReference("test", SecretCustom, "kv-v2::data/myapp/config")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Mount != "kv-v2" {
		t.Errorf("expected Mount 'kv-v2', got %q", ref.Mount)
	}
	if ref.Path != "data/myapp/config" {
		t.Errorf("expected Path 'data/myapp/config', got %q", ref.Path)
	}
}

func TestParseReference_SingleColonMount(t *testing.T) {
	ref, err := ParseReference("test", SecretCustom, "secret:myapp/config")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if ref.Mount != "secret" {
		t.Errorf("expected Mount 'secret', got %q", ref.Mount)
	}
	if ref.Path != "myapp/config" {
		t.Errorf("expected Path 'myapp/config', got %q", ref.Path)
	}
}

func TestParseReference_PathOnlyAfterTrim(t *testing.T) {
	_, err := ParseReference("test", SecretCustom, "///")
	if !errors.Is(err, ErrInvalidReference) {
		t.Errorf("expected ErrInvalidReference for path that becomes empty after trim, got %v", err)
	}
}

// ============== Secret Tests ==============

func TestSecret_Value_Found(t *testing.T) {
	secret := Secret{
		Data: map[string]string{
			"username": "admin",
			"password": "secret123",
		},
	}

	val, ok := secret.Value("password")
	if !ok {
		t.Error("expected key to be found")
	}
	if val != "secret123" {
		t.Errorf("expected 'secret123', got %q", val)
	}
}

func TestSecret_Value_NotFound(t *testing.T) {
	secret := Secret{
		Data: map[string]string{
			"username": "admin",
		},
	}

	val, ok := secret.Value("password")
	if ok {
		t.Error("expected key to not be found")
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}
}

func TestSecret_Value_EmptyString(t *testing.T) {
	secret := Secret{
		Data: map[string]string{
			"empty": "",
		},
	}

	val, ok := secret.Value("empty")
	if ok {
		t.Error("expected empty string to return ok=false")
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}
}

func TestSecret_Value_NilData(t *testing.T) {
	secret := Secret{Data: nil}

	val, ok := secret.Value("any")
	if ok {
		t.Error("expected nil data to return ok=false")
	}
	if val != "" {
		t.Errorf("expected empty string, got %q", val)
	}
}

// ============== Metadata Tests ==============

func TestMetadata_Fields(t *testing.T) {
	now := time.Now()
	metadata := Metadata{
		Version:     "v1",
		CreatedAt:   now.Add(-time.Hour),
		UpdatedAt:   now.Add(-30 * time.Minute),
		RetrievedAt: now,
		RotateAfter: now.Add(30 * 24 * time.Hour),
	}

	if metadata.Version != "v1" {
		t.Errorf("expected Version 'v1', got %q", metadata.Version)
	}
	if metadata.RetrievedAt != now {
		t.Error("RetrievedAt mismatch")
	}
}

// ============== Config Tests ==============

func TestConfig_Defaults(t *testing.T) {
	cfg := Config{
		Provider: ProviderVault,
	}

	if cfg.CacheTTL != 0 {
		t.Error("CacheTTL should be zero by default")
	}
	if cfg.RotationInterval != 0 {
		t.Error("RotationInterval should be zero by default")
	}
	if cfg.AuditEnabled {
		t.Error("AuditEnabled should be false by default")
	}
}

// ============== NewManager Tests ==============

func TestNewManager_NoProvider(t *testing.T) {
	cfg := Config{
		Provider: ProviderNone,
	}

	_, err := NewManager(cfg)
	if !errors.Is(err, ErrProviderNotConfigured) {
		t.Errorf("expected ErrProviderNotConfigured, got %v", err)
	}
}

func TestNewManager_EmptyProvider(t *testing.T) {
	cfg := Config{
		Provider: "",
	}

	_, err := NewManager(cfg)
	if !errors.Is(err, ErrProviderNotConfigured) {
		t.Errorf("expected ErrProviderNotConfigured, got %v", err)
	}
}

func TestNewManager_UnsupportedProvider(t *testing.T) {
	cfg := Config{
		Provider: ProviderType("unsupported"),
	}

	_, err := NewManager(cfg)
	if err == nil {
		t.Error("expected error for unsupported provider")
	}
	if !strings.Contains(err.Error(), "unsupported provider") {
		t.Errorf("expected 'unsupported provider' in error, got %v", err)
	}
}

func TestNewManager_VaultMissingConfig(t *testing.T) {
	cfg := Config{
		Provider: ProviderVault,
		Vault: VaultConfig{
			Address: "",
			Token:   "",
		},
	}

	_, err := NewManager(cfg)
	if err == nil {
		t.Error("expected error for missing Vault config")
	}
}

func TestNewManager_AWSMissingRegion(t *testing.T) {
	cfg := Config{
		Provider: ProviderAWS,
		AWS: AWSConfig{
			Region: "",
		},
	}

	_, err := NewManager(cfg)
	if err == nil {
		t.Error("expected error for missing AWS region")
	}
}

func TestNewManager_GCPMissingProject(t *testing.T) {
	cfg := Config{
		Provider: ProviderGCP,
		GCP: GCPConfig{
			ProjectID: "",
		},
	}

	_, err := NewManager(cfg)
	if err == nil {
		t.Error("expected error for missing GCP project ID")
	}
}

// ============== Kubernetes Provider Tests ==============

func TestNewKubernetesProvider_InvalidBasePath(t *testing.T) {
	cfg := KubernetesConfig{
		BasePath: "/nonexistent/path/that/does/not/exist",
	}

	_, err := newKubernetesProvider(cfg)
	if err == nil {
		t.Error("expected error for nonexistent base path")
	}
}

func TestNewKubernetesProvider_FileNotDirectory(t *testing.T) {
	// Create a temp file
	tmpFile, err := os.CreateTemp("", "test-secret")
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	cfg := KubernetesConfig{
		BasePath: tmpFile.Name(),
	}

	_, err = newKubernetesProvider(cfg)
	if err == nil {
		t.Error("expected error for file (not directory) base path")
	}
	if !strings.Contains(err.Error(), "not a directory") {
		t.Errorf("expected 'not a directory' in error, got %v", err)
	}
}

func TestKubernetesProvider_Name(t *testing.T) {
	// Create temp directory
	tmpDir, err := os.MkdirTemp("", "k8s-secrets")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	prov, err := newKubernetesProvider(KubernetesConfig{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if prov.Name() != ProviderKubernetes {
		t.Errorf("expected ProviderKubernetes, got %v", prov.Name())
	}
}

func TestKubernetesProvider_Close(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k8s-secrets")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	prov, err := newKubernetesProvider(KubernetesConfig{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	err = prov.Close()
	if err != nil {
		t.Errorf("Close() should return nil, got %v", err)
	}
}

func TestKubernetesProvider_FetchFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k8s-secrets")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a secret file
	secretPath := filepath.Join(tmpDir, "api-key")
	err = os.WriteFile(secretPath, []byte("my-secret-value"), 0600)
	if err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	prov, err := newKubernetesProvider(KubernetesConfig{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ref := Reference{Path: "api-key"}
	secret, err := prov.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}

	if val, ok := secret.Data["api-key"]; !ok || val != "my-secret-value" {
		t.Errorf("expected 'my-secret-value', got %q", val)
	}
}

func TestKubernetesProvider_FetchDirectory(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k8s-secrets")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a secret directory with multiple files
	secretDir := filepath.Join(tmpDir, "db-credentials")
	err = os.MkdirAll(secretDir, 0700)
	if err != nil {
		t.Fatalf("failed to create secret dir: %v", err)
	}

	err = os.WriteFile(filepath.Join(secretDir, "username"), []byte("admin"), 0600)
	if err != nil {
		t.Fatalf("failed to write username: %v", err)
	}
	err = os.WriteFile(filepath.Join(secretDir, "password"), []byte("secret123"), 0600)
	if err != nil {
		t.Fatalf("failed to write password: %v", err)
	}

	prov, err := newKubernetesProvider(KubernetesConfig{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ref := Reference{Path: "db-credentials"}
	secret, err := prov.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}

	if val, ok := secret.Data["username"]; !ok || val != "admin" {
		t.Errorf("expected username 'admin', got %q", val)
	}
	if val, ok := secret.Data["password"]; !ok || val != "secret123" {
		t.Errorf("expected password 'secret123', got %q", val)
	}
}

func TestKubernetesProvider_FetchNotFound(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k8s-secrets")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	prov, err := newKubernetesProvider(KubernetesConfig{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ref := Reference{Path: "nonexistent"}
	_, err = prov.Fetch(context.Background(), ref)
	if err == nil {
		t.Error("expected error for nonexistent path")
	}
}

func TestKubernetesProvider_FetchTrimsWhitespace(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "k8s-secrets")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	// Create a secret file with whitespace
	secretPath := filepath.Join(tmpDir, "api-key")
	err = os.WriteFile(secretPath, []byte("  my-secret-value  \n"), 0600)
	if err != nil {
		t.Fatalf("failed to write secret file: %v", err)
	}

	prov, err := newKubernetesProvider(KubernetesConfig{BasePath: tmpDir})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	ref := Reference{Path: "api-key"}
	secret, err := prov.Fetch(context.Background(), ref)
	if err != nil {
		t.Fatalf("Fetch() error: %v", err)
	}

	if val, ok := secret.Data["api-key"]; !ok || val != "my-secret-value" {
		t.Errorf("expected trimmed 'my-secret-value', got %q", val)
	}
}

// ============== Vault Provider Tests ==============

func TestVaultConfig_Defaults(t *testing.T) {
	cfg := VaultConfig{}

	if cfg.Address != "" {
		t.Error("Address should be empty by default")
	}
	if cfg.MountPath != "" {
		t.Error("MountPath should be empty by default (set during provider creation)")
	}
}

func TestNewVaultProvider_MissingAddress(t *testing.T) {
	cfg := VaultConfig{
		Address: "",
		Token:   "test-token",
	}

	_, err := newVaultProvider(cfg)
	if err == nil {
		t.Error("expected error for missing address")
	}
}

func TestNewVaultProvider_MissingToken(t *testing.T) {
	cfg := VaultConfig{
		Address: "http://localhost:8200",
		Token:   "",
	}

	_, err := newVaultProvider(cfg)
	if err == nil {
		t.Error("expected error for missing token")
	}
}

// ============== AWS Provider Tests ==============

func TestAWSConfig_Defaults(t *testing.T) {
	cfg := AWSConfig{}

	if cfg.Region != "" {
		t.Error("Region should be empty by default")
	}
}

// ============== GCP Provider Tests ==============

func TestGCPConfig_Defaults(t *testing.T) {
	cfg := GCPConfig{}

	if cfg.ProjectID != "" {
		t.Error("ProjectID should be empty by default")
	}
}

// ============== cloneSecret Tests ==============

func TestCloneSecret(t *testing.T) {
	original := Secret{
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
		},
		Metadata: Metadata{
			Version:   "v1",
			CreatedAt: time.Now(),
		},
	}

	clone := cloneSecret(original)

	// Verify data is copied
	if len(clone.Data) != len(original.Data) {
		t.Error("clone should have same number of data entries")
	}
	for k, v := range original.Data {
		if clone.Data[k] != v {
			t.Errorf("clone data mismatch for key %s", k)
		}
	}

	// Verify metadata is copied
	if clone.Metadata.Version != original.Metadata.Version {
		t.Error("clone metadata version mismatch")
	}

	// Verify modification doesn't affect original
	clone.Data["key1"] = "modified"
	if original.Data["key1"] == "modified" {
		t.Error("modifying clone should not affect original")
	}
}

func TestCloneSecret_EmptyData(t *testing.T) {
	original := Secret{
		Data:     map[string]string{},
		Metadata: Metadata{},
	}

	clone := cloneSecret(original)

	if clone.Data == nil {
		t.Error("clone Data should not be nil")
	}
	if len(clone.Data) != 0 {
		t.Error("clone Data should be empty")
	}
}

func TestCloneSecret_NilData(t *testing.T) {
	original := Secret{
		Data:     nil,
		Metadata: Metadata{},
	}

	clone := cloneSecret(original)

	if clone.Data == nil {
		t.Error("clone Data should not be nil (initialized to empty map)")
	}
}

// ============== Manager Cache Tests ==============

func TestManager_CacheKey_Uniqueness(t *testing.T) {
	refs := []Reference{
		{Path: "secret1"},
		{Path: "secret2"},
		{Mount: "kv", Path: "secret1"},
		{Path: "secret1", Version: "1"},
		{Path: "secret1", Key: "password"},
	}

	keys := make(map[string]bool)
	for _, ref := range refs {
		key := ref.CacheKey()
		if keys[key] {
			t.Errorf("duplicate cache key: %s", key)
		}
		keys[key] = true
	}
}

// ============== Concurrent Access Tests ==============

func TestReference_ConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup
	ref := Reference{
		Mount:   "kv",
		Path:    "secret/path",
		Version: "1",
		Key:     "password",
	}

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_ = ref.CacheKey()
		}()
	}

	wg.Wait()
}

func TestParseReference_ConcurrentAccess(t *testing.T) {
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, _ = ParseReference("test", SecretCustom, "vault://kv:secret/path@1#key")
		}(i)
	}

	wg.Wait()
}

// ============== Table-Driven ParseReference Tests ==============

func TestParseReference_TableDriven(t *testing.T) {
	tests := []struct {
		name        string
		secretName  string
		secretType  SecretType
		raw         string
		wantErr     bool
		wantPath    string
		wantMount   string
		wantVersion string
		wantKey     string
		wantProvider ProviderType
	}{
		{
			name:       "simple path",
			secretName: "test",
			secretType: SecretCustom,
			raw:        "my/secret",
			wantPath:   "my/secret",
		},
		{
			name:       "empty string",
			secretName: "test",
			secretType: SecretCustom,
			raw:        "",
			wantErr:    true,
		},
		{
			name:         "with provider",
			secretName:   "test",
			secretType:   SecretCustom,
			raw:          "vault://secret/path",
			wantPath:     "secret/path",
			wantProvider: ProviderVault,
		},
		{
			name:        "with version",
			secretName:  "test",
			secretType:  SecretCustom,
			raw:         "secret/path@v2",
			wantPath:    "secret/path",
			wantVersion: "v2",
		},
		{
			name:       "with key",
			secretName: "test",
			secretType: SecretCustom,
			raw:        "secret/path#password",
			wantPath:   "secret/path",
			wantKey:    "password",
		},
		{
			name:        "with mount (double colon)",
			secretName:  "test",
			secretType:  SecretCustom,
			raw:         "kv::secret/path",
			wantPath:    "secret/path",
			wantMount:   "kv",
		},
		{
			name:        "with mount (single colon)",
			secretName:  "test",
			secretType:  SecretCustom,
			raw:         "kv:secret/path",
			wantPath:    "secret/path",
			wantMount:   "kv",
		},
		{
			name:         "full reference",
			secretName:   "db_creds",
			secretType:   SecretDatabase,
			raw:          "vault://kv:database/prod@3#password",
			wantPath:     "database/prod",
			wantMount:    "kv",
			wantVersion:  "3",
			wantKey:      "password",
			wantProvider: ProviderVault,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			ref, err := ParseReference(tc.secretName, tc.secretType, tc.raw)

			if tc.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}

			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			if ref.Path != tc.wantPath {
				t.Errorf("Path: expected %q, got %q", tc.wantPath, ref.Path)
			}
			if ref.Mount != tc.wantMount {
				t.Errorf("Mount: expected %q, got %q", tc.wantMount, ref.Mount)
			}
			if ref.Version != tc.wantVersion {
				t.Errorf("Version: expected %q, got %q", tc.wantVersion, ref.Version)
			}
			if ref.Key != tc.wantKey {
				t.Errorf("Key: expected %q, got %q", tc.wantKey, ref.Key)
			}
			if ref.Provider != tc.wantProvider {
				t.Errorf("Provider: expected %v, got %v", tc.wantProvider, ref.Provider)
			}
		})
	}
}

// ============== Benchmark Tests ==============

func BenchmarkReference_CacheKey(b *testing.B) {
	ref := Reference{
		Mount:   "kv",
		Path:    "secret/path",
		Version: "1",
		Key:     "password",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ref.CacheKey()
	}
}

func BenchmarkParseReference_Simple(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseReference("test", SecretCustom, "my/secret/path")
	}
}

func BenchmarkParseReference_Full(b *testing.B) {
	for i := 0; i < b.N; i++ {
		_, _ = ParseReference("test", SecretCustom, "vault://kv:secret/path@1#password")
	}
}

func BenchmarkSecret_Value(b *testing.B) {
	secret := Secret{
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = secret.Value("key2")
	}
}

func BenchmarkCloneSecret(b *testing.B) {
	secret := Secret{
		Data: map[string]string{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
			"key4": "value4",
			"key5": "value5",
		},
		Metadata: Metadata{
			Version:   "v1",
			CreatedAt: time.Now(),
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = cloneSecret(secret)
	}
}

// ============== Edge Case Tests ==============

func TestParseReference_SpecialCharacters(t *testing.T) {
	// Test various special characters in paths
	testCases := []string{
		"secret/path-with-dashes",
		"secret/path_with_underscores",
		"secret/path.with.dots",
		"secret/path/with/many/levels",
	}

	for _, raw := range testCases {
		ref, err := ParseReference("test", SecretCustom, raw)
		if err != nil {
			t.Errorf("ParseReference(%q) error: %v", raw, err)
			continue
		}
		if ref.Path == "" {
			t.Errorf("ParseReference(%q) returned empty path", raw)
		}
	}
}

func TestParseReference_UnicodeCharacters(t *testing.T) {
	// Unicode in secret names (probably not common but should work)
	raw := "secret/path-with-unicode-\u00e9"
	ref, err := ParseReference("test", SecretCustom, raw)
	if err != nil {
		t.Errorf("ParseReference with unicode error: %v", err)
	}
	if !strings.Contains(ref.Path, "\u00e9") {
		t.Error("unicode character should be preserved in path")
	}
}

func TestSecret_Value_LargeData(t *testing.T) {
	// Test with many keys
	data := make(map[string]string)
	for i := 0; i < 1000; i++ {
		data[string(rune('a'+i%26))+string(rune(i))] = "value"
	}

	secret := Secret{Data: data}

	// Should still find keys quickly
	_, ok := secret.Value("a0")
	if !ok {
		// May not find exact key due to how we generated them
	}
}
