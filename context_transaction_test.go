package limen

import (
	"context"
	"errors"
	"testing"
	"time"

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
		testMemoryAdapter: testMemoryAdapter{tables: make(map[SchemaTableName]*testMemTable)},
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

func newTestLimenWithTxAdapter(t *testing.T, hooks ...DatabaseHook) (*Limen, *testTxAdapter) {
	t.Helper()

	adapter := &testTxAdapter{
		testMemoryAdapter: testMemoryAdapter{
			tables: make(map[SchemaTableName]*testMemTable),
		},
	}

	return newTestLimenFrom(t, &Config{Database: adapter, DatabaseHooks: hooks}), adapter
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

func TestWithTransaction_NestedJoinsExisting(t *testing.T) {
	t.Parallel()

	l, adapter := newTestLimenWithTxAdapter(t)

	var outerTx, innerTx DatabaseTx
	err := l.core.WithTransaction(context.Background(), func(ctx context.Context) error {
		outerTx = getTxFromContext(ctx)
		return l.core.WithTransaction(ctx, func(nestedCtx context.Context) error {
			innerTx = getTxFromContext(nestedCtx)
			return nil
		})
	})

	require.NoError(t, err)
	assert.NotNil(t, outerTx)
	assert.Same(t, outerTx, innerTx, "nested WithTransaction should reuse the outer transaction")
	assert.True(t, adapter.committed)
	assert.False(t, adapter.rolledBack)
}

func TestWithTransaction_NestedErrorRollsBackOuter(t *testing.T) {
	t.Parallel()

	l, adapter := newTestLimenWithTxAdapter(t)
	testErr := errors.New("nested failure")

	err := l.core.WithTransaction(context.Background(), func(ctx context.Context) error {
		return l.core.WithTransaction(ctx, func(context.Context) error {
			return testErr
		})
	})

	assert.ErrorIs(t, err, testErr)
	assert.True(t, adapter.rolledBack)
	assert.False(t, adapter.committed)
}

func TestWithTransaction_AfterHooksRunAfterCommitInWriteOrder(t *testing.T) {
	t.Parallel()

	var l *Limen
	var adapter *testTxAdapter
	var order []string
	var committedAtRun []bool
	var txVisibleToHook []bool
	l, adapter = newTestLimenWithTxAdapter(t, AfterCreate(AnySchema, func(hc *DatabaseHookContext) error {
		order = append(order, string(hc.Schema.GetSchemaName()))
		committedAtRun = append(committedAtRun, adapter.committed)
		txVisibleToHook = append(txVisibleToHook, getTxFromContext(hc.Context) != nil)
		return nil
	}))

	err := l.core.WithTransaction(context.Background(), func(ctx context.Context) error {
		if err := l.core.DBAction.CreateUser(ctx, &User{Email: "a@example.com"}, nil); err != nil {
			return err
		}
		_, err := l.core.DBAction.CreateVerification(ctx, "verify", "a@example.com", "token", time.Hour)
		return err
	})

	require.NoError(t, err)
	assert.Equal(t, []string{"users", "verifications"}, order)
	assert.Equal(t, []bool{true, true}, committedAtRun)
	assert.Equal(t, []bool{false, false}, txVisibleToHook)
}

func TestWithTransaction_AfterHooksDroppedOnRollback(t *testing.T) {
	t.Parallel()

	calls := 0
	l, adapter := newTestLimenWithTxAdapter(t, AfterCreate(CoreSchemaUsers, func(*DatabaseHookContext) error {
		calls++
		return nil
	}))
	testErr := errors.New("operation failed")

	err := l.core.WithTransaction(context.Background(), func(ctx context.Context) error {
		if err := l.core.DBAction.CreateUser(ctx, &User{Email: "a@example.com"}, nil); err != nil {
			return err
		}
		return testErr
	})

	assert.ErrorIs(t, err, testErr)
	assert.True(t, adapter.rolledBack)
	assert.Zero(t, calls)
}

func TestWithTransaction_AfterHookErrorIsReturnedAfterCommit(t *testing.T) {
	t.Parallel()

	hookErr := errors.New("audit failed")
	l, adapter := newTestLimenWithTxAdapter(t, AfterCreate(CoreSchemaUsers, func(*DatabaseHookContext) error {
		return hookErr
	}))

	err := l.core.WithTransaction(context.Background(), func(ctx context.Context) error {
		return l.core.DBAction.CreateUser(ctx, &User{Email: "a@example.com"}, nil)
	})

	assert.ErrorIs(t, err, hookErr)
	assert.True(t, adapter.committed)
	assert.False(t, adapter.rolledBack)
}

func TestWithTransaction_NestedWritesQueueOnOuterTransaction(t *testing.T) {
	t.Parallel()

	calls := 0
	l, _ := newTestLimenWithTxAdapter(t, AfterCreate(CoreSchemaUsers, func(*DatabaseHookContext) error {
		calls++
		return nil
	}))

	callsAfterInner := -1
	err := l.core.WithTransaction(context.Background(), func(ctx context.Context) error {
		err := l.core.WithTransaction(ctx, func(nestedCtx context.Context) error {
			return l.core.DBAction.CreateUser(nestedCtx, &User{Email: "a@example.com"}, nil)
		})
		callsAfterInner = calls
		return err
	})

	require.NoError(t, err)
	assert.Zero(t, callsAfterInner)
	assert.Equal(t, 1, calls)
}
