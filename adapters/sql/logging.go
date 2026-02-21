package sql

import (
	"context"
	"database/sql"
	"time"

	"github.com/jmoiron/sqlx"
)

// QueryLogger is called for each executed query when set on the adapter.
// Duration is the wall-clock time of the DB round-trip; args may be []any or map for named params.
type QueryLogger interface {
	LogQuery(ctx context.Context, query string, args any, duration time.Duration, err error)
}

// Logging wrappers delegate to db/tx and call LogQuery after each operation (for profiling or debugging).
type loggingExt struct {
	sqlx.ExtContext
	logger QueryLogger
}

func (l *loggingExt) ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error) {
	start := time.Now()
	res, err := l.ExtContext.ExecContext(ctx, query, args...)
	l.logger.LogQuery(ctx, query, args, time.Since(start), err)
	return res, err
}

type loggingNamedExecer struct {
	inner  namedExecer
	logger QueryLogger
}

func (l *loggingNamedExecer) NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error) {
	start := time.Now()
	res, err := l.inner.NamedExecContext(ctx, query, arg)
	l.logger.LogQuery(ctx, query, arg, time.Since(start), err)
	return res, err
}

type loggingQueryer struct {
	inner  queryerContext
	logger QueryLogger
}

func (l *loggingQueryer) QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row {
	start := time.Now()
	row := l.inner.QueryRowxContext(ctx, query, args...)
	l.logger.LogQuery(ctx, query, args, time.Since(start), nil)
	return row
}

func (l *loggingQueryer) QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error) {
	start := time.Now()
	rows, err := l.inner.QueryxContext(ctx, query, args...)
	l.logger.LogQuery(ctx, query, args, time.Since(start), err)
	return rows, err
}

func (l *loggingQueryer) QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row {
	start := time.Now()
	row := l.inner.QueryRowContext(ctx, query, args...)
	l.logger.LogQuery(ctx, query, args, time.Since(start), nil)
	return row
}
