package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/go-sql-driver/mysql"

	"github.com/thecodearcher/limen"
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
		SELECT column_name
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
		SELECT DISTINCT constraint_name
		FROM information_schema.key_column_usage
		WHERE table_schema = DATABASE() 
			AND table_name = ?
			AND referenced_table_name IS NOT NULL
	`, []any{tableName}
}

func (d *mysqlDriver) MapGoTypeToSQL(goType limen.ColumnType, isAutoIncrement bool) string {
	switch goType {
	case limen.ColumnTypeInt, limen.ColumnTypeInt32:
		return "INTEGER"
	case limen.ColumnTypeInt64:
		return "BIGINT"
	case limen.ColumnTypeBool:
		return "BOOLEAN"
	case limen.ColumnTypeString:
		return "VARCHAR(255)"
	case limen.ColumnTypeTime:
		return "TIMESTAMP"
	case limen.ColumnTypeUUID:
		return "VARCHAR(36)"
	case limen.ColumnTypeMapStringAny:
		return "JSON"
	default:
		return "TEXT"
	}
}

func (d *mysqlDriver) MapSQLTypeToGoType(dataType string) limen.ColumnType {
	dataType = strings.ToUpper(dataType)
	switch dataType {
	case "UUID", "VARCHAR(36)", "CHAR(36)":
		return limen.ColumnTypeUUID
	case "BOOLEAN", "BOOL", "TINYINT(1)":
		return limen.ColumnTypeBool
	case "INTEGER", "INT", "INT4", "SMALLINT", "MEDIUMINT":
		return limen.ColumnTypeInt32
	case "BIGINT", "INT8":
		return limen.ColumnTypeInt64
	case "TIMESTAMP", "DATETIME", "DATE":
		return limen.ColumnTypeTime
	case "JSON":
		return limen.ColumnTypeMapStringAny
	case "VARCHAR", "TEXT", "CHAR", "TINYTEXT", "MEDIUMTEXT", "LONGTEXT":
		return limen.ColumnTypeString
	default:
		return limen.ColumnTypeString
	}
}

func (d *mysqlDriver) GetAutoIncrementSuffix() string {
	return "AUTO_INCREMENT"
}

func (d *mysqlDriver) FormatDefaultValue(defaultValue string) string {
	switch defaultValue {
	case string(limen.DatabaseDefaultValueNow):
		return "CURRENT_TIMESTAMP"
	case string(limen.DatabaseDefaultValueUUID):
		return "UUID()"
	}
	return defaultValue
}

func (d *mysqlDriver) DropIndexSQL(tableName, indexName string) string {
	return fmt.Sprintf("DROP INDEX %s ON %s", indexName, tableName)
}

func (d *mysqlDriver) DropForeignKeySQL(tableName, constraintName string) string {
	return fmt.Sprintf("DROP FOREIGN KEY %s", constraintName)
}
