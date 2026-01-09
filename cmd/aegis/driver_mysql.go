package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"github.com/thecodearcher/aegis"
)

type mysqlDriver struct {
	baseDriver
}

// NewMySQLDriver creates a new MySQL/MariaDB driver
func NewMySQLDriver() Driver {
	return &mysqlDriver{}
}

func (d *mysqlDriver) Name() string {
	return string(DriverMySQL)
}

func (d *mysqlDriver) Connect(dsn string) (*sql.DB, error) {
	return sql.Open("mysql", dsn)
}

func (d *mysqlDriver) TableExistsBatchQuery(tableNames []string) (string, []any) {
	if len(tableNames) == 0 {
		return "SELECT table_name FROM information_schema.tables WHERE 1=0", []any{}
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(tableNames))
	args := make([]any, len(tableNames))
	for i, name := range tableNames {
		placeholders[i] = "?"
		args[i] = name
	}

	return fmt.Sprintf(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = DATABASE() 
		  AND table_name IN (%s)
	`, strings.Join(placeholders, ", ")), args
}

func (d *mysqlDriver) IntrospectColumnsQuery(tableName string) (string, []any) {
	return `
		SELECT 
			column_name,
			data_type,
			is_nullable,
			column_default,
			column_key = 'PRI' as is_primary_key
		FROM information_schema.columns
		WHERE table_schema = DATABASE() AND table_name = ?
		ORDER BY ordinal_position
	`, []any{tableName}
}

func (d *mysqlDriver) IntrospectIndexesQuery(tableName string) (string, []any) {
	return `
		SELECT 
			index_name,
			GROUP_CONCAT(column_name ORDER BY seq_in_index) as columns,
			non_unique = 0 as is_unique
		FROM information_schema.statistics
		WHERE table_schema = DATABASE() AND table_name = ?
		GROUP BY index_name, non_unique
		HAVING index_name != 'PRIMARY'
	`, []any{tableName}
}

func (d *mysqlDriver) IntrospectForeignKeysQuery(tableName string) (string, []any) {
	return `
		SELECT
			constraint_name,
			column_name,
			referenced_table_name,
			referenced_column_name,
			delete_rule,
			update_rule
		FROM information_schema.key_column_usage
		WHERE table_schema = DATABASE() 
			AND table_name = ?
			AND referenced_table_name IS NOT NULL
	`, []any{tableName}
}

func (d *mysqlDriver) MapGoTypeToSQL(goType aegis.ColumnType, isAutoIncrement bool) string {
	switch goType {
	case aegis.ColumnTypeInt, aegis.ColumnTypeInt32:
		return "INTEGER"
	case aegis.ColumnTypeInt64:
		return "BIGINT"
	case aegis.ColumnTypeBool:
		return "BOOLEAN"
	case aegis.ColumnTypeString:
		return "VARCHAR(255)"
	case aegis.ColumnTypeTime:
		return "DATETIME"
	case aegis.ColumnTypeUUID:
		return "VARCHAR(36)"
	case aegis.ColumnTypeMapStringAny:
		return "JSON"
	default:
		return "TEXT"
	}
}

func (d *mysqlDriver) MapSQLTypeToGoType(dataType string) aegis.ColumnType {
	dataType = strings.ToUpper(dataType)
	switch dataType {
	case "UUID", "VARCHAR(36)", "CHAR(36)":
		return aegis.ColumnTypeUUID
	case "BOOLEAN", "BOOL", "TINYINT(1)":
		return aegis.ColumnTypeBool
	case "INTEGER", "INT", "INT4", "SMALLINT", "MEDIUMINT":
		return aegis.ColumnTypeInt32
	case "BIGINT", "INT8":
		return aegis.ColumnTypeInt64
	case "TIMESTAMP", "DATETIME", "DATE":
		return aegis.ColumnTypeTime
	case "JSON":
		return aegis.ColumnTypeMapStringAny
	case "VARCHAR", "TEXT", "CHAR", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT":
		return aegis.ColumnTypeString
	default:
		return aegis.ColumnTypeString
	}
}

func (d *mysqlDriver) GetAutoIncrementSuffix() string {
	return "AUTO_INCREMENT"
}

func (d *mysqlDriver) DropIndexSQL(tableName, indexName string) string {
	return fmt.Sprintf("DROP INDEX %s ON %s", indexName, tableName)
}

func (d *mysqlDriver) DropForeignKeySQL(tableName, constraintName string) string {
	return fmt.Sprintf("DROP FOREIGN KEY %s", constraintName)
}

func (d *mysqlDriver) ParseColumnRow(scan func(dest ...any) error) (aegis.ColumnDefinition, error) {
	var colName, dataType, isNullable string
	var colDefault sql.NullString
	var isPK bool

	if err := scan(&colName, &dataType, &isNullable, &colDefault, &isPK); err != nil {
		return aegis.ColumnDefinition{}, err
	}

	colType := d.MapSQLTypeToGoType(dataType)

	col, err := d.parseColumnRowCommon(colName, isNullable, colDefault, isPK)
	if err != nil {
		return aegis.ColumnDefinition{}, err
	}

	col.Type = colType
	return col, nil
}
