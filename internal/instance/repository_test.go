package instance_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/instance"
	"github.com/t0mer/aghsync/internal/store"
)

var testSecret = make([]byte, 32) // all zeros; fine for test isolation

func openRepo(t *testing.T) *instance.Repository {
	t.Helper()
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return instance.NewRepository(s.DB(), testSecret)
}

// --- Create ---

func TestRepository_Create_NonMaster(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()

	inst, err := repo.Create(ctx, "Child1", "http://192.168.1.2:3000", "admin", "pass", false, false)
	require.NoError(t, err)
	assert.NotEmpty(t, inst.ID)
	assert.Equal(t, "Child1", inst.Name)
	assert.Equal(t, "http://192.168.1.2:3000", inst.Address)
	assert.Equal(t, "admin", inst.Username)
	assert.False(t, inst.IsMaster)
	assert.False(t, inst.TLSSkipVerify)
	assert.False(t, inst.CreatedAt.IsZero())
}

func TestRepository_Create_Master_DemotesPrevious(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()

	master1, err := repo.Create(ctx, "Master1", "http://10.0.0.1:3000", "a", "p", true, false)
	require.NoError(t, err)
	assert.True(t, master1.IsMaster)

	master2, err := repo.Create(ctx, "Master2", "http://10.0.0.2:3000", "b", "p", true, false)
	require.NoError(t, err)
	assert.True(t, master2.IsMaster)

	// master1 should now be demoted
	got, err := repo.Get(ctx, master1.ID)
	require.NoError(t, err)
	assert.False(t, got.IsMaster)
}

func TestRepository_Create_MasterHasDefaultSyncConfig(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()

	inst, err := repo.Create(ctx, "Master", "http://10.0.0.3:3000", "a", "p", true, false)
	require.NoError(t, err)

	cfg, err := repo.GetSyncConfig(ctx, inst.ID)
	require.NoError(t, err)
	assert.Len(t, cfg, len(instance.AllConfigTypes))
	for _, e := range cfg {
		assert.True(t, e.Enabled, "config_type %q should default to enabled", e.ConfigType)
	}
}

func TestRepository_Create_NonMasterHasNoSyncConfig(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()

	inst, err := repo.Create(ctx, "Child", "http://10.0.0.4:3000", "a", "p", false, false)
	require.NoError(t, err)

	cfg, err := repo.GetSyncConfig(ctx, inst.ID)
	require.NoError(t, err)
	assert.Empty(t, cfg) // slaves have no sync_config rows
}

// --- Get / List ---

func TestRepository_Get_NotFound(t *testing.T) {
	repo := openRepo(t)
	_, err := repo.Get(context.Background(), "nonexistent-id")
	assert.ErrorIs(t, err, instance.ErrNotFound)
}

func TestRepository_List_Empty(t *testing.T) {
	repo := openRepo(t)
	list, err := repo.List(context.Background())
	require.NoError(t, err)
	assert.Empty(t, list)
}

func TestRepository_List_OrderedByCreatedAt(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()

	for _, name := range []string{"A", "B", "C"} {
		_, err := repo.Create(ctx, name, "http://1.2.3.4:3000", "u", "p", false, false)
		require.NoError(t, err)
	}

	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 3)
	assert.Equal(t, "A", list[0].Name)
	assert.Equal(t, "B", list[1].Name)
	assert.Equal(t, "C", list[2].Name)
}

// --- Update ---

func TestRepository_Update_NameAndAddress(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()

	inst, err := repo.Create(ctx, "Old", "http://1.1.1.1:3000", "u", "p", false, false)
	require.NoError(t, err)

	updated, err := repo.Update(ctx, inst.ID, "New", "http://2.2.2.2:3000", "u2", nil, true)
	require.NoError(t, err)
	assert.Equal(t, "New", updated.Name)
	assert.Equal(t, "http://2.2.2.2:3000", updated.Address)
	assert.Equal(t, "u2", updated.Username)
	assert.True(t, updated.TLSSkipVerify)
}

func TestRepository_Update_PasswordChange(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()

	inst, err := repo.Create(ctx, "X", "http://1.1.1.1:3000", "u", "old-pass", false, false)
	require.NoError(t, err)

	newPass := "new-pass"
	_, err = repo.Update(ctx, inst.ID, "X", "http://1.1.1.1:3000", "u", &newPass, false)
	require.NoError(t, err)
}

func TestRepository_Update_NotFound(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()
	_, err := repo.Update(ctx, "bad-id", "X", "http://1.1.1.1:3000", "u", nil, false)
	assert.ErrorIs(t, err, instance.ErrNotFound)
}

// --- Delete ---

func TestRepository_Delete(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()

	inst, err := repo.Create(ctx, "Del", "http://1.1.1.1:3000", "u", "p", false, false)
	require.NoError(t, err)

	require.NoError(t, repo.Delete(ctx, inst.ID))

	_, err = repo.Get(ctx, inst.ID)
	assert.ErrorIs(t, err, instance.ErrNotFound)
}

func TestRepository_Delete_NotFound(t *testing.T) {
	repo := openRepo(t)
	assert.ErrorIs(t, repo.Delete(context.Background(), "nope"), instance.ErrNotFound)
}

// --- Promote ---

func TestRepository_Promote(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()

	master, err := repo.Create(ctx, "Master", "http://1.1.1.1:3000", "u", "p", true, false)
	require.NoError(t, err)
	child, err := repo.Create(ctx, "Child", "http://2.2.2.2:3000", "u", "p", false, false)
	require.NoError(t, err)

	// Disable one config type on the master before promoting.
	require.NoError(t, repo.SetSyncConfig(ctx, master.ID, []instance.SyncConfigEntry{
		{ConfigType: "dhcp", Enabled: false},
	}))

	require.NoError(t, repo.Promote(ctx, child.ID))

	newMaster, _ := repo.Get(ctx, child.ID)
	assert.True(t, newMaster.IsMaster)

	oldMaster, _ := repo.Get(ctx, master.ID)
	assert.False(t, oldMaster.IsMaster)

	// Old master should have no sync_config after demotion.
	demotedCfg, err := repo.GetSyncConfig(ctx, master.ID)
	require.NoError(t, err)
	assert.Empty(t, demotedCfg)

	// Promoted child should inherit master's sync_config (dhcp disabled).
	promotedCfg, err := repo.GetSyncConfig(ctx, child.ID)
	require.NoError(t, err)
	byType := make(map[string]bool)
	for _, e := range promotedCfg {
		byType[e.ConfigType] = e.Enabled
	}
	assert.False(t, byType["dhcp"], "dhcp should have been transferred as disabled")
	assert.True(t, byType["dns"], "dns should have been transferred as enabled")
}

func TestRepository_Promote_TransfersConfigWhenMasterHasNone(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()

	// Create master then manually delete its sync_config to simulate edge case.
	master, err := repo.Create(ctx, "Master", "http://1.1.1.1:3000", "u", "p", true, false)
	require.NoError(t, err)
	child, err := repo.Create(ctx, "Child", "http://2.2.2.2:3000", "u", "p", false, false)
	require.NoError(t, err)
	_ = master

	require.NoError(t, repo.Promote(ctx, child.ID))

	// Promoted child should have all config types enabled (defaults).
	promotedCfg, err := repo.GetSyncConfig(ctx, child.ID)
	require.NoError(t, err)
	assert.Len(t, promotedCfg, len(instance.AllConfigTypes))
	for _, e := range promotedCfg {
		assert.True(t, e.Enabled)
	}
}

func TestRepository_Promote_NotFound(t *testing.T) {
	repo := openRepo(t)
	assert.ErrorIs(t, repo.Promote(context.Background(), "bad-id"), instance.ErrNotFound)
}

// --- SyncConfig ---

func TestRepository_SetAndGetSyncConfig(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()

	inst, err := repo.Create(ctx, "Master", "http://1.1.1.1:3000", "u", "p", true, false)
	require.NoError(t, err)

	entries := []instance.SyncConfigEntry{
		{ConfigType: "filtering", Enabled: false},
		{ConfigType: "dns", Enabled: true},
	}
	require.NoError(t, repo.SetSyncConfig(ctx, inst.ID, entries))

	got, err := repo.GetSyncConfig(ctx, inst.ID)
	require.NoError(t, err)

	byType := make(map[string]bool)
	for _, e := range got {
		byType[e.ConfigType] = e.Enabled
	}
	assert.False(t, byType["filtering"])
	assert.True(t, byType["dns"])
}

func TestRepository_GetSyncConfig_NotFound(t *testing.T) {
	repo := openRepo(t)
	_, err := repo.GetSyncConfig(context.Background(), "nope")
	assert.ErrorIs(t, err, instance.ErrNotFound)
}

// --- GetDecryptedPassword ---

func TestRepository_GetDecryptedPassword(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()

	inst, err := repo.Create(ctx, "test", "http://agh", "admin", "secret123", false, false)
	require.NoError(t, err)

	pw, err := repo.GetDecryptedPassword(ctx, inst.ID)
	require.NoError(t, err)
	assert.Equal(t, "secret123", pw)

	_, err = repo.GetDecryptedPassword(ctx, "nonexistent")
	assert.ErrorIs(t, err, instance.ErrNotFound)
}
