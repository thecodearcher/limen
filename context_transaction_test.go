package limen

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type testTxAdapter struct {
	testMemoryAdapter
	committed  bool
	rolledBack bool
}

type testTx struct {
	parent *testTxAdapter
	testMemoryAdapter
}

func (a *testTxAdapter) BeginTx(_ context.Context) (DatabaseTx, error) {
	tx := &testTx{
		parent:            a,
		testMemoryAdapter: testMemoryAdapter{tables: make(map[SchemaTableName]*memTable)},
	}
	return tx, nil
}

func (tx *testTx) Commit() error {
	tx.parent.committed = true
	return nil
}

func (tx *testTx) Rollback() error {
	tx.parent.rolledBack = true
	return nil
}

func newTestLimenWithTxAdapter(t *testing.T) (*Limen, *testTxAdapter) {
	t.Helper()

	adapter := &testTxAdapter{
		testMemoryAdapter: testMemoryAdapter{
			tables: make(map[SchemaTableName]*memTable),
		},
	}

	l, err := New(&Config{
		BaseURL:  "http://localhost:8080",
		Database: adapter,
		Secret:   testSecret,
	})
	require.NoError(t, err)
	return l, adapter
}

func TestWithTransaction_Commit(t *testing.T) {
	t.Parallel()

	l, adapter := newTestLimenWithTxAdapter(t)

	err := l.core.WithTransaction(context.Background(), func(ctx context.Context) error {
		return nil
	})

	require.NoError(t, err)
	assert.True(t, adapter.committed, "transaction should be committed on success")
	assert.False(t, adapter.rolledBack)
}

func TestWithTransaction_RollbackOnError(t *testing.T) {
	t.Parallel()

	l, adapter := newTestLimenWithTxAdapter(t)
	testErr := errors.New("operation failed")

	err := l.core.WithTransaction(context.Background(), func(ctx context.Context) error {
		return testErr
	})

	assert.ErrorIs(t, err, testErr)
	assert.True(t, adapter.rolledBack, "transaction should be rolled back on error")
	assert.False(t, adapter.committed)
}

func TestWithTransaction_RollbackOnPanic(t *testing.T) {
	t.Parallel()

	l, adapter := newTestLimenWithTxAdapter(t)

	assert.Panics(t, func() {
		_ = l.core.WithTransaction(context.Background(), func(ctx context.Context) error {
			panic("unexpected failure")
		})
	})

	assert.True(t, adapter.rolledBack, "transaction should be rolled back on panic")
	assert.False(t, adapter.committed)
}

func TestWithTransaction_NonTransactionalAdapter(t *testing.T) {
	t.Parallel()

	l := newTestLimen(t)

	called := false
	err := l.core.WithTransaction(context.Background(), func(ctx context.Context) error {
		called = true
		return nil
	})

	require.NoError(t, err)
	assert.True(t, called, "fn should execute even without transaction support")
}

func TestWithTransaction_ContextPropagation(t *testing.T) {
	t.Parallel()

	l, _ := newTestLimenWithTxAdapter(t)

	err := l.core.WithTransaction(context.Background(), func(ctx context.Context) error {
		tx := getTxFromContext(ctx)
		assert.NotNil(t, tx, "transaction should be available in context")
		return nil
	})

	require.NoError(t, err)
}
