package main

import (
	"database/sql"
	"fmt"

	"github.com/thecodearcher/aegis"
)

type schemaIntrospector struct {
	db     *sql.DB
	driver Driver
}

func newSchemaIntrospector(db *sql.DB, driver Driver) *schemaIntrospector {
	return &schemaIntrospector{
		db:     db,
		driver: driver,
	}
}

func (s *schemaIntrospector) getTables(tableNames []string) (map[string]bool, error) {
	result := make(map[string]bool, len(tableNames))

	query, args := s.driver.TableExistsBatchQuery(tableNames)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		result[tableName] = true
	}

	return result, nil
}

func (s *schemaIntrospector) introspectTable(tableName aegis.SchemaTableName) (*aegis.SchemaDefinition, error) {
	columns, err := s.introspectColumns(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect columns: %w", err)
	}

	indexes, err := s.introspectIndexes(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect indexes: %w", err)
	}

	foreignKeys, err := s.introspectForeignKeys(tableName)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect foreign keys: %w", err)
	}

	return &aegis.SchemaDefinition{
		TableName:   &tableName,
		Columns:     columns,
		Indexes:     indexes,
		ForeignKeys: foreignKeys,
		SchemaName:  string(tableName),
	}, nil
}

func (s *schemaIntrospector) introspectColumns(tableName aegis.SchemaTableName) ([]aegis.ColumnDefinition, error) {
	var columns []aegis.ColumnDefinition

	query, args := s.driver.IntrospectColumnsQuery(string(tableName))
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		col, err := s.driver.ParseColumnRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		columns = append(columns, col)
	}

	return columns, nil
}

func (s *schemaIntrospector) introspectIndexes(tableName aegis.SchemaTableName) ([]aegis.IndexDefinition, error) {
	var indexes []aegis.IndexDefinition

	query, args := s.driver.IntrospectIndexesQuery(string(tableName))
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		idx, err := s.driver.ParseIndexRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		indexes = append(indexes, idx)
	}

	return indexes, nil
}

func (s *schemaIntrospector) introspectForeignKeys(tableName aegis.SchemaTableName) ([]aegis.ForeignKeyDefinition, error) {
	var foreignKeys []aegis.ForeignKeyDefinition

	query, args := s.driver.IntrospectForeignKeysQuery(string(tableName))
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		fk, err := s.driver.ParseForeignKeyRow(rows.Scan)
		if err != nil {
			return nil, err
		}
		foreignKeys = append(foreignKeys, fk)
	}

	return foreignKeys, nil
}
