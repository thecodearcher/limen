package main

import (
	"database/sql"

	"github.com/thecodearcher/aegis"
)

// Driver defines the interface for database driver-specific operations
type Driver interface {
	Name() string
	Connect(dsn string) (*sql.DB, error)
	TableExistsBatchQuery(tableNames []string) (string, []any)
	IntrospectColumnsQuery(tableName string) (string, []any)
	IntrospectIndexesQuery(tableName string) (string, []any)
	IntrospectForeignKeysQuery(tableName string) (string, []any)
	MapGoTypeToSQL(goType aegis.ColumnType, isAutoIncrement bool) string
	MapSQLTypeToGoType(dataType string) aegis.ColumnType
	GetAutoIncrementSuffix() string
	// FormatDefaultValue converts special default value syntax to database-specific SQL
	// Special syntax patterns like "@now()" or "@uuid()" are converted to the appropriate
	// database function (e.g., now() for PostgreSQL, CURRENT_TIMESTAMP for MySQL)
	FormatDefaultValue(defaultValue string) string
	DropColumnSQL(tableName, columnName string) string
	DropIndexSQL(tableName, indexName string) string
	DropForeignKeySQL(tableName, constraintName string) string
	// ParseColumnRow parses a row from IntrospectColumnsQuery into a ColumnDefinition
	// The columns returned by the query should match the order expected by this method
	ParseColumnRow(scan func(dest ...any) error) (aegis.ColumnDefinition, error)

	// ParseIndexRow parses a row from IntrospectIndexesQuery into an IndexDefinition
	// The columns returned by the query should match the order expected by this method
	ParseIndexRow(scan func(dest ...any) error) (aegis.IndexDefinition, error)

	// ParseForeignKeyRow parses a row from IntrospectForeignKeysQuery into a ForeignKeyDefinition
	// The columns returned by the query should match the order expected by this method
	ParseForeignKeyRow(scan func(dest ...any) error) (aegis.ForeignKeyDefinition, error)
}
