package jwtkeys

import (
	"context"
	"encoding/base64"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

// mockStore is a test Store backed by a simple slice.
type mockStore struct {
	keys    []SigningKey
	saveErr error
	loadErr error
	saved   []SigningKey // last saved payload
}

func (m *mockStore) Load(_ context.Context) ([]SigningKey, error) {
	if m.loadErr != nil {
		return nil, m.loadErr
	}
	cp := make([]SigningKey, len(m.keys))
	copy(cp, m.keys)
	return cp, nil
}

func (m *mockStore) Save(_ context.Context, keys []SigningKey) error {
	if m.saveErr != nil {
		return m.saveErr
	}
	m.saved = make([]SigningKey, len(keys))
	copy(m.saved, keys)
	m.keys = make([]SigningKey, len(keys))
	copy(m.keys, keys)
	return nil
}

func validSecret() string {
	return base64.StdEncoding.EncodeToString([]byte("this-is-a-48-byte-secret-for-testing-purposes!!"))
}

func makeKey(id string, createdAt time.Time, rotationInterval, gracePeriod time.Duration, revoked bool) SigningKey {
	return SigningKey{
		ID:        id,
		Secret:    validSecret(),
		CreatedAt: createdAt,
		ExpiresAt: createdAt.Add(rotationInterval + gracePeriod),
		Revoked:   revoked,
	}
}

// ---------------------------------------------------------------------------
// SigningKey.SecretBytes
// ---------------------------------------------------------------------------

func TestSigningKey_SecretBytes(t *testing.T) {
	raw := []byte("test-secret-bytes")
	key := SigningKey{
		Secret: base64.StdEncoding.EncodeToString(raw),
	}

	decoded, err := key.SecretBytes()
	require.NoError(t, err)
	assert.Equal(t, raw, decoded)
}

func TestSigningKey_SecretBytes_InvalidBase64(t *testing.T) {
	key := SigningKey{Secret: "!!!not-valid-base64!!!"}
	_, err := key.SecretBytes()
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// SigningKey.Clone
// ---------------------------------------------------------------------------

func TestSigningKey_Clone(t *testing.T) {
	original := &SigningKey{
		ID:        "kid_123",
		Secret:    validSecret(),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(time.Hour),
		Revoked:   false,
	}

	cloned := original.Clone()

	assert.Equal(t, original.ID, cloned.ID)
	assert.Equal(t, original.Secret, cloned.Secret)
	assert.Equal(t, original.CreatedAt, cloned.CreatedAt)
	assert.Equal(t, original.ExpiresAt, cloned.ExpiresAt)

	// Mutation of clone must not affect original
	cloned.ID = "kid_modified"
	assert.NotEqual(t, original.ID, cloned.ID)
}

func TestSigningKey_Clone_Nil(t *testing.T) {
	var key *SigningKey
	assert.Nil(t, key.Clone())
}

// ---------------------------------------------------------------------------
// NewManager – defaults
// ---------------------------------------------------------------------------

func TestNewManager_DefaultRotationAndGrace(t *testing.T) {
	store := &mockStore{}
	mgr, err := NewManager(context.Background(), Config{
		Store: store,
	})
	require.NoError(t, err)
	assert.Equal(t, 30*24*time.Hour, mgr.rotationInterval)
	assert.Equal(t, 30*24*time.Hour, mgr.gracePeriod)
}

func TestNewManager_CustomRotationAndGrace(t *testing.T) {
	store := &mockStore{}
	mgr, err := NewManager(context.Background(), Config{
		Store:            store,
		RotationInterval: 7 * 24 * time.Hour,
		GracePeriod:      3 * 24 * time.Hour,
	})
	require.NoError(t, err)
	assert.Equal(t, 7*24*time.Hour, mgr.rotationInterval)
	assert.Equal(t, 3*24*time.Hour, mgr.gracePeriod)
}

// ---------------------------------------------------------------------------
// NewManager – seeds initial key
// ---------------------------------------------------------------------------

func TestNewManager_SeedsInitialKey_WhenStoreEmpty(t *testing.T) {
	store := &mockStore{}
	mgr, err := NewManager(context.Background(), Config{Store: store})
	require.NoError(t, err)

	assert.NotEmpty(t, mgr.activeID)
	assert.Len(t, mgr.keys, 1)

	// The key should be persisted
	assert.Len(t, store.saved, 1)
}

func TestNewManager_SeedsWithLegacySecret(t *testing.T) {
	store := &mockStore{}
	legacySecret := "my-legacy-secret-key"

	mgr, err := NewManager(context.Background(), Config{
		Store:        store,
		LegacySecret: legacySecret,
	})
	require.NoError(t, err)

	key, err := mgr.CurrentSigningKey()
	require.NoError(t, err)

	decoded, err := key.SecretBytes()
	require.NoError(t, err)
	assert.Equal(t, []byte(legacySecret), decoded)
}

// ---------------------------------------------------------------------------
// NewManager – read-only mode
// ---------------------------------------------------------------------------

func TestNewManager_ReadOnly_NoSeed(t *testing.T) {
	store := &mockStore{}
	mgr, err := NewManager(context.Background(), Config{
		Store:    store,
		ReadOnly: true,
	})
	require.NoError(t, err)

	assert.True(t, mgr.readOnly)
	assert.Empty(t, mgr.activeID)
	assert.Empty(t, mgr.keys)
}

func TestNewManager_ReadOnly_LoadsExistingKeys(t *testing.T) {
	now := time.Now()
	store := &mockStore{
		keys: []SigningKey{
			makeKey("kid_1", now, 30*24*time.Hour, 30*24*time.Hour, false),
		},
	}

	mgr, err := NewManager(context.Background(), Config{
		Store:    store,
		ReadOnly: true,
	})
	require.NoError(t, err)

	assert.Equal(t, "kid_1", mgr.activeID)
	assert.Len(t, mgr.keys, 1)
}

// ---------------------------------------------------------------------------
// NewManager – loads existing keys from store
// ---------------------------------------------------------------------------

func TestNewManager_LoadsExistingKeys(t *testing.T) {
	now := time.Now()
	store := &mockStore{
		keys: []SigningKey{
			makeKey("kid_old", now.Add(-48*time.Hour), 30*24*time.Hour, 30*24*time.Hour, false),
			makeKey("kid_new", now, 30*24*time.Hour, 30*24*time.Hour, false),
		},
	}

	mgr, err := NewManager(context.Background(), Config{Store: store})
	require.NoError(t, err)

	// Should pick the newest non-revoked key as active
	assert.Equal(t, "kid_new", mgr.activeID)
	assert.Len(t, mgr.keys, 2)
}

func TestNewManager_SkipsRevokedKeysForActive(t *testing.T) {
	now := time.Now()
	store := &mockStore{
		keys: []SigningKey{
			makeKey("kid_old", now.Add(-48*time.Hour), 30*24*time.Hour, 30*24*time.Hour, false),
			makeKey("kid_revoked", now, 30*24*time.Hour, 30*24*time.Hour, true),
		},
	}

	mgr, err := NewManager(context.Background(), Config{Store: store})
	require.NoError(t, err)

	assert.Equal(t, "kid_old", mgr.activeID)
}

// ---------------------------------------------------------------------------
// NewManager – memory store fallback
// ---------------------------------------------------------------------------

func TestNewManager_MemoryStoreFallback(t *testing.T) {
	mgr, err := NewManager(context.Background(), Config{})
	require.NoError(t, err)
	assert.NotNil(t, mgr)
	assert.NotEmpty(t, mgr.activeID) // seeded a key
}

// ---------------------------------------------------------------------------
// NewManager – store load error
// ---------------------------------------------------------------------------

func TestNewManager_StoreLoadError(t *testing.T) {
	store := &mockStore{loadErr: assert.AnError}
	_, err := NewManager(context.Background(), Config{Store: store})
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// CurrentSigningKey
// ---------------------------------------------------------------------------

func TestCurrentSigningKey_ReturnsClone(t *testing.T) {
	store := &mockStore{}
	mgr, err := NewManager(context.Background(), Config{Store: store})
	require.NoError(t, err)

	key1, err := mgr.CurrentSigningKey()
	require.NoError(t, err)

	key2, err := mgr.CurrentSigningKey()
	require.NoError(t, err)

	// Should return equivalent but distinct objects
	assert.Equal(t, key1.ID, key2.ID)

	key1.ID = "mutated"
	key2Again, _ := mgr.CurrentSigningKey()
	assert.NotEqual(t, "mutated", key2Again.ID)
}

func TestCurrentSigningKey_NoActiveKey(t *testing.T) {
	store := &mockStore{}
	mgr, err := NewManager(context.Background(), Config{
		Store:    store,
		ReadOnly: true,
	})
	require.NoError(t, err)

	_, err = mgr.CurrentSigningKey()
	assert.Error(t, err)
	assert.Equal(t, errNoActiveKey, err)
}

// ---------------------------------------------------------------------------
// ResolveKey
// ---------------------------------------------------------------------------

func TestResolveKey_ValidKey(t *testing.T) {
	now := time.Now()
	store := &mockStore{
		keys: []SigningKey{
			makeKey("kid_1", now, 30*24*time.Hour, 30*24*time.Hour, false),
		},
	}

	mgr, err := NewManager(context.Background(), Config{Store: store})
	require.NoError(t, err)

	secret, err := mgr.ResolveKey("kid_1")
	require.NoError(t, err)
	assert.NotEmpty(t, secret)
}

func TestResolveKey_EmptyKID(t *testing.T) {
	store := &mockStore{}
	mgr, err := NewManager(context.Background(), Config{Store: store})
	require.NoError(t, err)

	_, err = mgr.ResolveKey("")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestResolveKey_UnknownKID(t *testing.T) {
	store := &mockStore{}
	mgr, err := NewManager(context.Background(), Config{Store: store})
	require.NoError(t, err)

	_, err = mgr.ResolveKey("kid_nonexistent")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestResolveKey_RevokedKey(t *testing.T) {
	now := time.Now()
	store := &mockStore{
		keys: []SigningKey{
			makeKey("kid_revoked", now, 30*24*time.Hour, 30*24*time.Hour, true),
			makeKey("kid_active", now, 30*24*time.Hour, 30*24*time.Hour, false),
		},
	}

	mgr, err := NewManager(context.Background(), Config{Store: store})
	require.NoError(t, err)

	_, err = mgr.ResolveKey("kid_revoked")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestResolveKey_ExpiredKey(t *testing.T) {
	expired := time.Now().Add(-100 * 24 * time.Hour)
	store := &mockStore{
		keys: []SigningKey{
			{
				ID:        "kid_expired",
				Secret:    validSecret(),
				CreatedAt: expired,
				ExpiresAt: expired.Add(time.Hour), // already past
				Revoked:   false,
			},
			makeKey("kid_active", time.Now(), 30*24*time.Hour, 30*24*time.Hour, false),
		},
	}

	mgr, err := NewManager(context.Background(), Config{Store: store})
	require.NoError(t, err)

	_, err = mgr.ResolveKey("kid_expired")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

// ---------------------------------------------------------------------------
// LegacyKey
// ---------------------------------------------------------------------------

func TestLegacyKey(t *testing.T) {
	store := &mockStore{}
	mgr, err := NewManager(context.Background(), Config{
		Store:        store,
		LegacySecret: "legacy-secret-123",
	})
	require.NoError(t, err)

	assert.Equal(t, []byte("legacy-secret-123"), mgr.LegacyKey())
}

func TestLegacyKey_Empty(t *testing.T) {
	store := &mockStore{}
	mgr, err := NewManager(context.Background(), Config{Store: store})
	require.NoError(t, err)

	assert.Empty(t, mgr.LegacyKey())
}

// ---------------------------------------------------------------------------
// EnsureRotation
// ---------------------------------------------------------------------------

func TestEnsureRotation_RotatesWhenDue(t *testing.T) {
	rotationInterval := 1 * time.Hour
	gracePeriod := 30 * time.Minute

	oldCreated := time.Now().Add(-2 * time.Hour) // past rotation interval
	store := &mockStore{
		keys: []SigningKey{
			makeKey("kid_old", oldCreated, rotationInterval, gracePeriod, false),
		},
	}

	mgr, err := NewManager(context.Background(), Config{
		Store:            store,
		RotationInterval: rotationInterval,
		GracePeriod:      gracePeriod,
	})
	require.NoError(t, err)

	// The manager already loaded the old key; its creation time is past the interval
	err = mgr.EnsureRotation(context.Background())
	require.NoError(t, err)

	// Should have created a new key
	assert.NotEqual(t, "kid_old", mgr.activeID)
	assert.GreaterOrEqual(t, len(mgr.keys), 1)
}

func TestEnsureRotation_NoOpWhenFresh(t *testing.T) {
	rotationInterval := 24 * time.Hour
	gracePeriod := 24 * time.Hour

	store := &mockStore{
		keys: []SigningKey{
			makeKey("kid_fresh", time.Now(), rotationInterval, gracePeriod, false),
		},
	}

	mgr, err := NewManager(context.Background(), Config{
		Store:            store,
		RotationInterval: rotationInterval,
		GracePeriod:      gracePeriod,
	})
	require.NoError(t, err)

	originalID := mgr.activeID
	err = mgr.EnsureRotation(context.Background())
	require.NoError(t, err)

	assert.Equal(t, originalID, mgr.activeID)
}

func TestEnsureRotation_ReadOnly_NoOp(t *testing.T) {
	now := time.Now()
	store := &mockStore{
		keys: []SigningKey{
			makeKey("kid_1", now.Add(-48*time.Hour), time.Hour, time.Hour, false),
		},
	}

	mgr, err := NewManager(context.Background(), Config{
		Store:            store,
		ReadOnly:         true,
		RotationInterval: time.Hour,
	})
	require.NoError(t, err)

	err = mgr.EnsureRotation(context.Background())
	require.NoError(t, err)

	// Should not have rotated
	assert.Equal(t, "kid_1", mgr.activeID)
}

// ---------------------------------------------------------------------------
// EnsureRotation – pruning expired keys
// ---------------------------------------------------------------------------

func TestEnsureRotation_PrunesExpiredKeys(t *testing.T) {
	rotationInterval := time.Hour
	gracePeriod := time.Hour

	now := time.Now()
	store := &mockStore{
		keys: []SigningKey{
			{
				ID:        "kid_expired",
				Secret:    validSecret(),
				CreatedAt: now.Add(-5 * time.Hour),
				ExpiresAt: now.Add(-1 * time.Hour), // already expired
				Revoked:   false,
			},
			makeKey("kid_fresh", now, rotationInterval, gracePeriod, false),
		},
	}

	mgr, err := NewManager(context.Background(), Config{
		Store:            store,
		RotationInterval: rotationInterval,
		GracePeriod:      gracePeriod,
	})
	require.NoError(t, err)

	err = mgr.EnsureRotation(context.Background())
	require.NoError(t, err)

	// Expired key should be pruned
	_, err = mgr.ResolveKey("kid_expired")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

// ---------------------------------------------------------------------------
// StaticProvider
// ---------------------------------------------------------------------------

func TestStaticProvider_ResolveKey(t *testing.T) {
	provider := NewStaticProvider("static-secret")

	secret, err := provider.ResolveKey("any-kid")
	require.NoError(t, err)
	assert.Equal(t, []byte("static-secret"), secret)

	// kid is ignored
	secret2, err := provider.ResolveKey("")
	require.NoError(t, err)
	assert.Equal(t, secret, secret2)
}

func TestStaticProvider_ResolveKey_EmptySecret(t *testing.T) {
	provider := NewStaticProvider("")

	_, err := provider.ResolveKey("any")
	assert.ErrorIs(t, err, ErrKeyNotFound)
}

func TestStaticProvider_LegacyKey(t *testing.T) {
	provider := NewStaticProvider("my-secret")
	assert.Equal(t, []byte("my-secret"), provider.LegacyKey())
}

func TestStaticProvider_LegacyKey_Empty(t *testing.T) {
	provider := NewStaticProvider("")
	assert.Empty(t, provider.LegacyKey())
}

// ---------------------------------------------------------------------------
// generateKeyID
// ---------------------------------------------------------------------------

func TestGenerateKeyID(t *testing.T) {
	now := time.Date(2025, 6, 15, 12, 0, 0, 0, time.UTC)
	kid := generateKeyID(now)

	assert.Contains(t, kid, "kid_")
	assert.NotEmpty(t, kid)

	// Different timestamps yield different IDs
	now2 := now.Add(time.Nanosecond)
	kid2 := generateKeyID(now2)
	assert.NotEqual(t, kid, kid2)
}

// ---------------------------------------------------------------------------
// generateSecret
// ---------------------------------------------------------------------------

func TestGenerateSecret(t *testing.T) {
	secret1, err := generateSecret()
	require.NoError(t, err)
	assert.Len(t, secret1, 48) // 384 bits

	secret2, err := generateSecret()
	require.NoError(t, err)

	// Two random secrets should be different
	assert.NotEqual(t, secret1, secret2)
}

// ---------------------------------------------------------------------------
// memoryStore
// ---------------------------------------------------------------------------

func TestMemoryStore_LoadEmpty(t *testing.T) {
	store := newMemoryStore()
	keys, err := store.Load(context.Background())
	require.NoError(t, err)
	assert.Empty(t, keys)
}

func TestMemoryStore_SaveAndLoad(t *testing.T) {
	store := newMemoryStore()

	input := []SigningKey{
		{ID: "k1", Secret: validSecret(), CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)},
		{ID: "k2", Secret: validSecret(), CreatedAt: time.Now(), ExpiresAt: time.Now().Add(time.Hour)},
	}

	err := store.Save(context.Background(), input)
	require.NoError(t, err)

	loaded, err := store.Load(context.Background())
	require.NoError(t, err)
	assert.Len(t, loaded, 2)
	assert.Equal(t, "k1", loaded[0].ID)
	assert.Equal(t, "k2", loaded[1].ID)
}

func TestMemoryStore_SaveOverwrites(t *testing.T) {
	store := newMemoryStore()

	first := []SigningKey{{ID: "a"}}
	_ = store.Save(context.Background(), first)

	second := []SigningKey{{ID: "b"}, {ID: "c"}}
	_ = store.Save(context.Background(), second)

	loaded, _ := store.Load(context.Background())
	assert.Len(t, loaded, 2)
	assert.Equal(t, "b", loaded[0].ID)
}

func TestMemoryStore_LoadReturnsCopy(t *testing.T) {
	store := newMemoryStore()
	_ = store.Save(context.Background(), []SigningKey{{ID: "x"}})

	loaded1, _ := store.Load(context.Background())
	loaded1[0].ID = "mutated"

	loaded2, _ := store.Load(context.Background())
	assert.Equal(t, "x", loaded2[0].ID) // original not mutated
}

// ---------------------------------------------------------------------------
// normalizeVaultPath
// ---------------------------------------------------------------------------

func TestNormalizeVaultPath(t *testing.T) {
	tests := []struct {
		name       string
		path       string
		wantMount  string
		wantSecret string
		wantErr    bool
	}{
		{
			name:       "mount only",
			path:       "secret",
			wantMount:  "secret",
			wantSecret: "jwt_keys", // default
		},
		{
			name:       "mount and secret",
			path:       "secret/mykeys",
			wantMount:  "secret",
			wantSecret: "mykeys",
		},
		{
			name:       "mount and nested secret",
			path:       "secret/services/jwt",
			wantMount:  "secret",
			wantSecret: "services/jwt",
		},
		{
			name:       "strips leading/trailing slashes",
			path:       "/secret/mykeys/",
			wantMount:  "secret",
			wantSecret: "mykeys",
		},
		{
			name:       "strips data/ prefix",
			path:       "secret/data/mykeys",
			wantMount:  "secret",
			wantSecret: "mykeys",
		},
		{
			name:       "strips metadata/ prefix",
			path:       "secret/metadata/mykeys",
			wantMount:  "secret",
			wantSecret: "mykeys",
		},
		{
			name:       "data/ prefix only gives default secret",
			path:       "secret/data/",
			wantMount:  "secret",
			wantSecret: "data",
		},
		{
			name:    "empty path",
			path:    "",
			wantErr: true,
		},
		{
			name:    "only slashes",
			path:    "///",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mount, secret, err := normalizeVaultPath(tt.path)
			if tt.wantErr {
				assert.Error(t, err)
				return
			}
			require.NoError(t, err)
			assert.Equal(t, tt.wantMount, mount)
			assert.Equal(t, tt.wantSecret, secret)
		})
	}
}

// ---------------------------------------------------------------------------
// KeyProvider interface compliance
// ---------------------------------------------------------------------------

func TestManager_ImplementsKeyProvider(t *testing.T) {
	store := &mockStore{}
	mgr, err := NewManager(context.Background(), Config{Store: store})
	require.NoError(t, err)

	var _ KeyProvider = mgr
}

func TestStaticProvider_ImplementsKeyProvider(t *testing.T) {
	var _ KeyProvider = NewStaticProvider("secret")
}

// ---------------------------------------------------------------------------
// Concurrent access safety
// ---------------------------------------------------------------------------

func TestManager_ConcurrentResolveKey(t *testing.T) {
	store := &mockStore{}
	mgr, err := NewManager(context.Background(), Config{Store: store})
	require.NoError(t, err)

	activeKey, _ := mgr.CurrentSigningKey()

	done := make(chan struct{})
	for i := 0; i < 50; i++ {
		go func() {
			defer func() { done <- struct{}{} }()
			_, _ = mgr.ResolveKey(activeKey.ID)
		}()
	}

	for i := 0; i < 50; i++ {
		<-done
	}
}

// ---------------------------------------------------------------------------
// Error sentinel values
// ---------------------------------------------------------------------------

func TestErrorSentinels(t *testing.T) {
	assert.EqualError(t, ErrKeyNotFound, "jwtkeys: signing key not found")
	assert.EqualError(t, errNoActiveKey, "jwtkeys: no active signing key available")
	assert.EqualError(t, errReadOnly, "jwtkeys: manager is read-only")
}
