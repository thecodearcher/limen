package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/thecodearcher/limen"
)

func (a *Adapter) Create(ctx context.Context, tableName limen.SchemaTableName, data map[string]any) error {
	if len(data) == 0 {
		return fmt.Errorf("create: no data")
	}
	cols := sortedKeys(data)
	quotedCols := make([]string, len(cols))
	for i, c := range cols {
		quotedCols[i] = a.quoteIdent(c)
	}
	// Named params: INSERT INTO t (c1, c2) VALUES (:c1, :c2)
	namedPlaceholders := make([]string, len(cols))
	for i, c := range cols {
		namedPlaceholders[i] = ":" + c
	}
	query := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)",
		a.quoteIdent(string(tableName)),
		strings.Join(quotedCols, ", "),
		strings.Join(namedPlaceholders, ", "))
	_, err := a.getNamed().NamedExecContext(ctx, query, data)
	return err
}

func (a *Adapter) FindOne(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where, orderBy []limen.OrderBy) (map[string]any, error) {
	whereSQL, args := a.buildWhere(conditions)
	query := "SELECT * FROM " + a.quoteIdent(string(tableName))
	if whereSQL != "" {
		query += " WHERE " + whereSQL
	}
	if len(orderBy) > 0 {
		parts := make([]string, len(orderBy))
		for i, o := range orderBy {
			parts[i] = a.quoteIdent(o.Column) + " " + string(o.Direction)
		}
		query += " ORDER BY " + strings.Join(parts, ", ")
	}
	query += " LIMIT 1"
	query = a.rebind(query)

	row := a.getQueryer().QueryRowxContext(ctx, query, args...)
	dest := make(map[string]any)
	err := row.MapScan(dest)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, limen.ErrRecordNotFound
	}
	if err != nil {
		return nil, err
	}
	if a.db.DriverName() == "mysql" {
		normalizeRow(dest)
	}
	return dest, nil
}

func (a *Adapter) FindMany(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where, options *limen.QueryOptions) ([]map[string]any, error) {
	whereSQL, args := a.buildWhere(conditions)
	query := "SELECT * FROM " + a.quoteIdent(string(tableName))
	if whereSQL != "" {
		query += " WHERE " + whereSQL
	}
	if options != nil && len(options.OrderBy) > 0 {
		parts := make([]string, len(options.OrderBy))
		for i, o := range options.OrderBy {
			parts[i] = a.quoteIdent(o.Column) + " " + string(o.Direction)
		}
		query += " ORDER BY " + strings.Join(parts, ", ")
	}
	if options != nil && options.Limit > 0 {
		query += " LIMIT " + fmt.Sprintf("%d", options.Limit)
	}
	if options != nil && options.Offset > 0 {
		query += " OFFSET " + fmt.Sprintf("%d", options.Offset)
	}
	query = a.rebind(query)

	rows, err := a.getQueryer().QueryxContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var results []map[string]any
	for rows.Next() {
		m := make(map[string]any)
		if err := rows.MapScan(m); err != nil {
			return nil, err
		}
		if a.db.DriverName() == "mysql" {
			normalizeRow(m)
		}
		results = append(results, m)
	}
	return results, rows.Err()
}

func (a *Adapter) Update(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where, updates map[string]any) error {
	if len(updates) == 0 {
		return nil
	}

	if len(conditions) == 0 {
		return fmt.Errorf("update: conditions required to prevent accidental table-wide update")
	}
	setParts := make([]string, 0, len(updates))
	setArgs := make([]any, 0, len(updates))
	for _, k := range sortedKeys(updates) {
		setParts = append(setParts, a.quoteIdent(k)+" = ?")
		setArgs = append(setArgs, updates[k])
	}
	whereSQL, whereArgs := a.buildWhere(conditions)
	args := append(setArgs, whereArgs...)
	query := "UPDATE " + a.quoteIdent(string(tableName)) + " SET " + strings.Join(setParts, ", ") + " WHERE " + whereSQL
	query = a.rebind(query)
	_, err := a.getExt().ExecContext(ctx, query, args...)
	return err
}

func (a *Adapter) Delete(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where) error {
	if len(conditions) == 0 {
		return fmt.Errorf("delete: conditions required to prevent accidental table-wide delete")
	}
	whereSQL, args := a.buildWhere(conditions)
	query := "DELETE FROM " + a.quoteIdent(string(tableName)) + " WHERE " + whereSQL
	query = a.rebind(query)
	_, err := a.getExt().ExecContext(ctx, query, args...)
	return err
}

func (a *Adapter) Exists(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where) (bool, error) {
	whereSQL, args := a.buildWhere(conditions)
	query := "SELECT 1 FROM " + a.quoteIdent(string(tableName))
	if whereSQL != "" {
		query += " WHERE " + whereSQL
	}
	query += " LIMIT 1"
	query = a.rebind(query)

	var dummy int
	err := a.getQueryer().QueryRowContext(ctx, query, args...).Scan(&dummy)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	return true, nil
}

func (a *Adapter) Count(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where) (int64, error) {
	whereSQL, args := a.buildWhere(conditions)
	query := "SELECT COUNT(*) FROM " + a.quoteIdent(string(tableName))
	if whereSQL != "" {
		query += " WHERE " + whereSQL
	}
	query = a.rebind(query)

	var n int64
	err := a.getQueryer().QueryRowContext(ctx, query, args...).Scan(&n)
	return n, err
}
