package limen

import (
	"context"
	"sync"
	"testing"
	"testing/synctest"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMemoryCacheStore_SetAndGet(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStore()
	ctx := context.Background()

	err := store.Set(ctx, "key1", []byte("value1"), 5*time.Minute)
	require.NoError(t, err)

	got, err := store.Get(ctx, "key1")
	require.NoError(t, err)
	assert.Equal(t, []byte("value1"), got)
}

func TestMemoryCacheStore_Get_NotFound(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStore()
	ctx := context.Background()

	_, err := store.Get(ctx, "nonexistent")
	assert.ErrorIs(t, err, ErrRecordNotFound)
}

func TestMemoryCacheStore_Get_Expired(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		store := NewMemoryCacheStore()
		ctx := context.Background()

		err := store.Set(ctx, "short-lived", []byte("data"), 1*time.Millisecond)
		require.NoError(t, err)

		time.Sleep(2 * time.Millisecond)

		_, err = store.Get(ctx, "short-lived")
		assert.ErrorIs(t, err, ErrRecordNotFound)
	})
}

func TestMemoryCacheStore_Set_ZeroTTL_NeverExpires(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStore()
	ctx := context.Background()

	err := store.Set(ctx, "forever", []byte("persistent"), 0)
	require.NoError(t, err)

	got, err := store.Get(ctx, "forever")
	require.NoError(t, err)
	assert.Equal(t, []byte("persistent"), got)
}

func TestMemoryCacheStore_Set_Overwrite(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStore()
	ctx := context.Background()

	require.NoError(t, store.Set(ctx, "key", []byte("v1"), 5*time.Minute))
	require.NoError(t, store.Set(ctx, "key", []byte("v2"), 5*time.Minute))

	got, err := store.Get(ctx, "key")
	require.NoError(t, err)
	assert.Equal(t, []byte("v2"), got)
}

func TestMemoryCacheStore_Has(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStore()
	ctx := context.Background()

	exists, err := store.Has(ctx, "missing")
	require.NoError(t, err)
	assert.False(t, exists)

	require.NoError(t, store.Set(ctx, "present", []byte("data"), 5*time.Minute))
	exists, err = store.Has(ctx, "present")
	require.NoError(t, err)
	assert.True(t, exists)
}

func TestMemoryCacheStore_Has_Expired(t *testing.T) {
	t.Parallel()

	synctest.Test(t, func(t *testing.T) {
		store := NewMemoryCacheStore()
		ctx := context.Background()

		require.NoError(t, store.Set(ctx, "expiring", []byte("data"), 1*time.Millisecond))
		time.Sleep(2 * time.Millisecond)

		exists, err := store.Has(ctx, "expiring")
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestMemoryCacheStore_Delete(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStore()
	ctx := context.Background()

	require.NoError(t, store.Set(ctx, "to-delete", []byte("data"), 5*time.Minute))

	err := store.Delete(ctx, "to-delete")
	require.NoError(t, err)

	_, err = store.Get(ctx, "to-delete")
	assert.ErrorIs(t, err, ErrRecordNotFound)
}

func TestMemoryCacheStore_Delete_NonExistent(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStore()
	ctx := context.Background()

	err := store.Delete(ctx, "does-not-exist")
	assert.NoError(t, err)
}

func TestMemoryCacheStore_ValueIsolation(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStore()
	ctx := context.Background()

	original := []byte("original")
	require.NoError(t, store.Set(ctx, "iso", original, 5*time.Minute))

	original[0] = 'X'

	got, err := store.Get(ctx, "iso")
	require.NoError(t, err)
	assert.Equal(t, byte('o'), got[0], "Set should copy the value, not retain a reference")

	got[0] = 'Y'
	got2, err := store.Get(ctx, "iso")
	require.NoError(t, err)
	assert.Equal(t, byte('o'), got2[0], "Get should return a copy, not the internal value")
}

func TestMemoryCacheStore_ConcurrentAccess(t *testing.T) {
	t.Parallel()

	store := NewMemoryCacheStore()
	ctx := context.Background()

	const goroutines = 50
	var wg sync.WaitGroup
	wg.Add(goroutines)

	for i := range goroutines {
		go func(n int) {
			defer wg.Done()
			key := "key"
			val := []byte("value")

			_ = store.Set(ctx, key, val, 5*time.Minute)
			_, _ = store.Get(ctx, key)
			_, _ = store.Has(ctx, key)
			if n%3 == 0 {
				_ = store.Delete(ctx, key)
			}
		}(i)
	}

	wg.Wait()
}
