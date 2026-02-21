package sql

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/thecodearcher/aegis"
)

// Adapter implements aegis.DatabaseAdapter using sqlx (extensions on database/sql).
type Adapter struct {
	db        *sqlx.DB
	tx        *sqlx.Tx
	quoteChar string // " for PostgreSQL/SQLite, ` for MySQL; empty defaults to "
	logger    QueryLogger
}

type namedExecer interface {
	NamedExecContext(ctx context.Context, query string, arg any) (sql.Result, error)
}

type queryerContext interface {
	QueryRowxContext(ctx context.Context, query string, args ...any) *sqlx.Row
	QueryxContext(ctx context.Context, query string, args ...any) (*sqlx.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

// newSQLAdapter returns an adapter for the given sqlx DB
func newSQLAdapter(db *sqlx.DB, quoteChar string) *Adapter {
	return &Adapter{db: db, quoteChar: quoteChar}
}

// NewPostgreSQL wraps a standard *sql.DB with sqlx for PostgreSQL and returns an adapter.
func NewPostgreSQL(db *sql.DB) *Adapter {
	return newSQLAdapter(sqlx.NewDb(db, "postgres"), `"`)
}

// NewMySQL wraps a standard *sql.DB with sqlx for MySQL and returns an adapter.
//
// The go-sql-driver/mysql text protocol returns []byte for string columns (see github.com/go-sql-driver/mysql/issues/407).
// This adapter normalizes []byte to string after MapScan so both text and binary (prepared) protocols work.
func NewMySQL(db *sql.DB) *Adapter {
	return newSQLAdapter(sqlx.NewDb(db, "mysql"), "`")
}

// NewSQLite wraps a standard *sql.DB with sqlx for SQLite and returns an adapter.
func NewSQLite(db *sql.DB) *Adapter {
	return newSQLAdapter(sqlx.NewDb(db, "sqlite"), `"`)
}

// WithLogger returns a copy of the adapter with the given query logger set.
// When non-nil, each executed query is logged (query, args, duration, error).
func (a *Adapter) WithLogger(logger QueryLogger) *Adapter {
	out := *a
	out.logger = logger
	return &out
}

// rebind converts ? placeholders to the driver's form (e.g. $1, $2 for PostgreSQL).
func (a *Adapter) rebind(query string) string {
	return a.db.Rebind(query)
}

// getExt returns the executor for ExecContext (tx if active, else db), optionally wrapped for query logging.
func (a *Adapter) getExt() sqlx.ExtContext {
	var ext sqlx.ExtContext
	if a.tx != nil {
		ext = a.tx
	} else {
		ext = a.db
	}
	if a.logger != nil {
		return &loggingExt{ExtContext: ext, logger: a.logger}
	}
	return ext
}

// getNamed returns the executor for NamedExecContext (tx if active, else db), optionally wrapped for query logging.
func (a *Adapter) getNamed() namedExecer {
	var inner namedExecer
	if a.tx != nil {
		inner = a.tx
	} else {
		inner = a.db
	}
	if a.logger != nil {
		return &loggingNamedExecer{inner: inner, logger: a.logger}
	}
	return inner
}

// getQueryer returns the executor for Query/QueryRow (tx if active, else db), optionally wrapped for query logging.
func (a *Adapter) getQueryer() queryerContext {
	var q queryerContext
	if a.tx != nil {
		q = a.tx
	} else {
		q = a.db
	}
	if a.logger != nil {
		return &loggingQueryer{inner: q, logger: a.logger}
	}
	return q
}

func (a *Adapter) BeginTx(ctx context.Context) (aegis.DatabaseTx, error) {
	if a.tx != nil {
		return nil, fmt.Errorf("already in a transaction")
	}
	tx, err := a.db.BeginTxx(ctx, nil)
	if err != nil {
		return nil, err
	}
	return &Adapter{db: a.db, tx: tx, quoteChar: a.quoteChar, logger: a.logger}, nil
}

func (a *Adapter) Commit() error {
	if a.tx == nil {
		return fmt.Errorf("not in a transaction")
	}
	err := a.tx.Commit()
	a.tx = nil
	return err
}

func (a *Adapter) Rollback() error {
	if a.tx == nil {
		return fmt.Errorf("not in a transaction")
	}
	err := a.tx.Rollback()
	a.tx = nil
	return err
}

// quoteIdent quotes an identifier (e.g. table/column name). Embedded quote chars are doubled per SQL rules.
func (a *Adapter) quoteIdent(name string) string {
	q := a.quoteChar
	if q == "" {
		q = `"`
	}
	return q + strings.ReplaceAll(name, q, q+q) + q
}
