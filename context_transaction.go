package limen

import (
	"context"
)

type contextKeyTransaction struct{}

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
func (core *LimenCore) WithTransaction(ctx context.Context, fn func(ctx context.Context) error) error {
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
			tx.Rollback()
			panic(v)
		}
	}()

	txCtx := context.WithValue(ctx, contextKeyTransaction{}, tx)
	if err := fn(txCtx); err != nil {
		tx.Rollback()
		return err
	}

	return tx.Commit()
}
