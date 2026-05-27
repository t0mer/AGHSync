package notification_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/t0mer/aghsync/internal/notification"
	"github.com/t0mer/aghsync/internal/store"
)

var testSecret = make([]byte, 32)

func openRepo(t *testing.T) *notification.Repository {
	t.Helper()
	s, err := store.Open(":memory:")
	require.NoError(t, err)
	t.Cleanup(func() { s.Close() })
	return notification.NewRepository(s.DB(), testSecret)
}

func TestRepository_Create_And_Get(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()

	ch, err := repo.Create(ctx, "test", notification.TypeShoutrrr, `{"url":"slack://x"}`, true, false, true)
	require.NoError(t, err)
	assert.NotEmpty(t, ch.ID)
	assert.Equal(t, "test", ch.Name)
	assert.Equal(t, notification.TypeShoutrrr, ch.Type)
	assert.Equal(t, `{"url":"slack://x"}`, ch.Config)
	assert.True(t, ch.NotifySuccess)
	assert.False(t, ch.NotifyFailure)
	assert.True(t, ch.Enabled)

	got, err := repo.Get(ctx, ch.ID)
	require.NoError(t, err)
	assert.Equal(t, ch.Config, got.Config)
}

func TestRepository_Create_DuplicateName(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()
	_, err := repo.Create(ctx, "dup", notification.TypeShoutrrr, `{}`, true, true, true)
	require.NoError(t, err)
	_, err = repo.Create(ctx, "dup", notification.TypeGreenAPI, `{}`, true, true, true)
	assert.ErrorIs(t, err, notification.ErrDuplicateName)
}

func TestRepository_Get_NotFound(t *testing.T) {
	repo := openRepo(t)
	_, err := repo.Get(context.Background(), "no-such-id")
	assert.ErrorIs(t, err, notification.ErrNotFound)
}

func TestRepository_List(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()
	_, err := repo.Create(ctx, "A", notification.TypeShoutrrr, `{}`, true, true, true)
	require.NoError(t, err)
	_, err = repo.Create(ctx, "B", notification.TypeGreenAPI, `{}`, true, true, false)
	require.NoError(t, err)

	list, err := repo.List(ctx)
	require.NoError(t, err)
	require.Len(t, list, 2)
	assert.Equal(t, "A", list[0].Name)
}

func TestRepository_Update(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()
	ch, err := repo.Create(ctx, "old", notification.TypeShoutrrr, `{"url":"a"}`, true, false, true)
	require.NoError(t, err)

	updated, err := repo.Update(ctx, ch.ID, "new", notification.TypeShoutrrr, `{"url":"b"}`, false, true, false)
	require.NoError(t, err)
	assert.Equal(t, "new", updated.Name)
	assert.Equal(t, `{"url":"b"}`, updated.Config)
	assert.False(t, updated.NotifySuccess)
	assert.True(t, updated.NotifyFailure)
	assert.False(t, updated.Enabled)
}

func TestRepository_Update_KeepConfig(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()
	ch, err := repo.Create(ctx, "x", notification.TypeShoutrrr, `{"url":"keep"}`, true, true, true)
	require.NoError(t, err)

	updated, err := repo.Update(ctx, ch.ID, "x", notification.TypeShoutrrr, "", true, true, true)
	require.NoError(t, err)
	assert.Equal(t, `{"url":"keep"}`, updated.Config)
}

func TestRepository_Delete(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()
	ch, err := repo.Create(ctx, "del", notification.TypeShoutrrr, `{}`, true, true, true)
	require.NoError(t, err)
	require.NoError(t, repo.Delete(ctx, ch.ID))
	_, err = repo.Get(ctx, ch.ID)
	assert.ErrorIs(t, err, notification.ErrNotFound)
}

func TestRepository_Delete_NotFound(t *testing.T) {
	repo := openRepo(t)
	assert.ErrorIs(t, repo.Delete(context.Background(), "nope"), notification.ErrNotFound)
}

func TestRepository_ListEnabled(t *testing.T) {
	repo := openRepo(t)
	ctx := context.Background()
	_, err := repo.Create(ctx, "success-only", notification.TypeShoutrrr, `{}`, true, false, true)
	require.NoError(t, err)
	_, err = repo.Create(ctx, "failure-only", notification.TypeShoutrrr, `{}`, false, true, true)
	require.NoError(t, err)
	_, err = repo.Create(ctx, "disabled", notification.TypeShoutrrr, `{}`, true, true, false)
	require.NoError(t, err)

	succChans, err := repo.ListEnabled(ctx, true)
	require.NoError(t, err)
	assert.Len(t, succChans, 1)
	assert.Equal(t, "success-only", succChans[0].Name)

	failChans, err := repo.ListEnabled(ctx, false)
	require.NoError(t, err)
	assert.Len(t, failChans, 1)
	assert.Equal(t, "failure-only", failChans[0].Name)
}
