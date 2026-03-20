package gorm

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/thecodearcher/limen"
)

func setupTestGormDB(t *testing.T) *Adapter {
	t.Helper()

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), &gorm.Config{
		Logger: logger.Discard,
	})
	if err != nil {
		t.Fatalf("failed to open gorm sqlite: %v", err)
	}

	sqlDB, _ := db.DB()
	t.Cleanup(func() { sqlDB.Close() })

	db.Exec(`CREATE TABLE IF NOT EXISTS "test_items" (
		"id" INTEGER PRIMARY KEY AUTOINCREMENT,
		"name" TEXT NOT NULL,
		"email" TEXT,
		"age" INTEGER DEFAULT 0,
		"status" TEXT DEFAULT 'active'
	)`)

	return New(db)
}

func TestGorm_Create_And_FindOne(t *testing.T) {
	adapter := setupTestGormDB(t)
	ctx := context.Background()

	err := adapter.Create(ctx, "test_items", map[string]any{
		"name":  "Alice",
		"email": "alice@example.com",
		"age":   30,
	})
	assert.NoError(t, err)

	result, err := adapter.FindOne(ctx, "test_items", []limen.Where{
		limen.Eq("name", "Alice"),
	}, nil)
	assert.NoError(t, err)
	assert.Equal(t, "Alice", result["name"])
	assert.Equal(t, "alice@example.com", result["email"])
}

func TestGorm_FindOne_NotFound(t *testing.T) {
	adapter := setupTestGormDB(t)
	ctx := context.Background()

	_, err := adapter.FindOne(ctx, "test_items", []limen.Where{
		limen.Eq("name", "Nonexistent"),
	}, nil)
	assert.ErrorIs(t, err, limen.ErrRecordNotFound)
}

func TestGorm_FindMany(t *testing.T) {
	adapter := setupTestGormDB(t)
	ctx := context.Background()

	for _, name := range []string{"Alice", "Bob", "Charlie"} {
		adapter.Create(ctx, "test_items", map[string]any{"name": name})
	}

	results, err := adapter.FindMany(ctx, "test_items", nil, nil)
	assert.NoError(t, err)
	assert.Len(t, results, 3)
}

func TestGorm_Update(t *testing.T) {
	adapter := setupTestGormDB(t)
	ctx := context.Background()

	adapter.Create(ctx, "test_items", map[string]any{"name": "Alice", "email": "old@test.com"})

	err := adapter.Update(ctx, "test_items", []limen.Where{
		limen.Eq("name", "Alice"),
	}, map[string]any{"email": "new@test.com"})
	assert.NoError(t, err)

	result, _ := adapter.FindOne(ctx, "test_items", []limen.Where{limen.Eq("name", "Alice")}, nil)
	assert.Equal(t, "new@test.com", result["email"])
}

func TestGorm_Delete(t *testing.T) {
	adapter := setupTestGormDB(t)
	ctx := context.Background()

	adapter.Create(ctx, "test_items", map[string]any{"name": "ToDelete"})

	err := adapter.Delete(ctx, "test_items", []limen.Where{
		limen.Eq("name", "ToDelete"),
	})
	assert.NoError(t, err)

	_, err = adapter.FindOne(ctx, "test_items", []limen.Where{limen.Eq("name", "ToDelete")}, nil)
	assert.ErrorIs(t, err, limen.ErrRecordNotFound)
}

func TestGorm_Exists(t *testing.T) {
	adapter := setupTestGormDB(t)
	ctx := context.Background()

	adapter.Create(ctx, "test_items", map[string]any{"name": "Exists"})

	exists, err := adapter.Exists(ctx, "test_items", []limen.Where{limen.Eq("name", "Exists")})
	assert.NoError(t, err)
	assert.True(t, exists)

	exists, err = adapter.Exists(ctx, "test_items", []limen.Where{limen.Eq("name", "Nope")})
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestGorm_Count(t *testing.T) {
	adapter := setupTestGormDB(t)
	ctx := context.Background()

	for _, name := range []string{"A", "B", "C"} {
		adapter.Create(ctx, "test_items", map[string]any{"name": name})
	}

	count, err := adapter.Count(ctx, "test_items", nil)
	assert.NoError(t, err)
	assert.Equal(t, int64(3), count)
}

func TestGorm_Transaction_Commit(t *testing.T) {
	adapter := setupTestGormDB(t)
	ctx := context.Background()

	tx, err := adapter.BeginTx(ctx)
	assert.NoError(t, err)

	txAdapter := tx.(*Adapter)
	txAdapter.Create(ctx, "test_items", map[string]any{"name": "InTx"})

	err = tx.Commit()
	assert.NoError(t, err)

	result, err := adapter.FindOne(ctx, "test_items", []limen.Where{limen.Eq("name", "InTx")}, nil)
	assert.NoError(t, err)
	assert.Equal(t, "InTx", result["name"])
}

func TestGorm_Transaction_Rollback(t *testing.T) {
	adapter := setupTestGormDB(t)
	ctx := context.Background()

	tx, err := adapter.BeginTx(ctx)
	assert.NoError(t, err)

	txAdapter := tx.(*Adapter)
	txAdapter.Create(ctx, "test_items", map[string]any{"name": "Rolled"})

	err = tx.Rollback()
	assert.NoError(t, err)

	_, err = adapter.FindOne(ctx, "test_items", []limen.Where{limen.Eq("name", "Rolled")}, nil)
	assert.ErrorIs(t, err, limen.ErrRecordNotFound)
}
