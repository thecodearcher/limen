package limen

import (
	"context"
)

type contextKeyTransaction struct{}

type contextKeyAfterHooks struct{}

// getTxFromContext retrieves the transaction from context if present.
func getTxFromContext(ctx context.Context) DatabaseTx {
	if tx, ok := ctx.Value(contextKeyTransaction{}).(DatabaseTx); ok {
		return tx
	}
	return nil
}

// WithTransaction executes fn within a database transaction.
// The transaction is automatically available in the context for all database operations
// within the callback. If the adapter doesn't support transactions, fn runs normally.
// Nested calls join the existing transaction instead of beginning a new one.
// After hooks for writes made inside fn run once the transaction has committed.
func (core *LimenCore) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
	if getTxFromContext(ctx) != nil {
		return fn(ctx)
	}

	txAdapter, ok := core.db.(TransactionalAdapter)
	if !ok {
		return fn(ctx)
	}

	tx, err := txAdapter.BeginTx(ctx)
	if err != nil {
		return err
	}

	defer func() {
		if v := recover(); v != nil {
			_ = tx.Rollback()
			panic(v)
		}
	}()

	var queued []func(context.Context) error
	txCtx := context.WithValue(ctx, contextKeyAfterHooks{}, &queued)
	txCtx = context.WithValue(txCtx, contextKeyTransaction{}, tx)
	if err := fn(txCtx); err != nil {
		_ = tx.Rollback()
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	for _, run := range queued {
		if err := run(ctx); err != nil {
			return err
		}
	}
	return nil
}
