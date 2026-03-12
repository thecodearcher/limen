package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "github.com/jackc/pgx/v5/stdlib"

	"github.com/thecodearcher/aegis"
)

type postgresDriver struct {
	baseDriver
	currentSchema string
}

func NewPostgresDriver() Driver {
	return &postgresDriver{}
}

func (d *postgresDriver) Name() string {
	return string(DriverPostgres)
}

func (d *postgresDriver) Connect(dsn string) (*sql.DB, error) {
	conn, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, err
	}

	d.currentSchema = d.getCurrentSchema(conn)
	return conn, nil
}

func (d *postgresDriver) getCurrentSchema(db *sql.DB) string {
	if d.currentSchema != "" {
		return d.currentSchema
	}

	var schema string
	err := db.QueryRow("SELECT current_schema()").Scan(&schema)
	if err != nil {
		schema = "public"
	}

	return schema
}

func (d *postgresDriver) getSchema() string {
	if d.currentSchema == "" {
		return "public"
	}
	return d.currentSchema
}

func (d *postgresDriver) TableExistsBatchQuery(tableNames []string) (string, []any) {
	schema := d.getSchema()
	if len(tableNames) == 0 {
		return "SELECT table_name FROM information_schema.tables WHERE 1=0", []any{}
	}

	// Build placeholders for IN clause
	placeholders := make([]string, len(tableNames))
	args := make([]any, len(tableNames)+1)
	args[0] = schema
	for i, name := range tableNames {
		placeholders[i] = fmt.Sprintf("$%d", i+2)
		args[i+1] = name
	}

	return fmt.Sprintf(`
		SELECT table_name 
		FROM information_schema.tables 
		WHERE table_schema = $1 
		  AND table_name IN (%s)
	`, strings.Join(placeholders, ", ")), args
}

func (d *postgresDriver) IntrospectColumnsQuery(tableName string) (string, []any) {
	schema := d.getSchema()
	return `
		SELECT column_name
		FROM information_schema.columns
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`, []any{schema, tableName}
}

func (d *postgresDriver) IntrospectIndexesQuery(tableName string) (string, []any) {
	schema := d.getSchema()
	return `
		SELECT
  		idx.relname AS indexname,
  		string_agg(pg_get_indexdef(i.indexrelid, k, true), ',' ORDER BY k) AS columns,
  		i.indisunique AS is_unique
		FROM pg_index i
		JOIN pg_class tbl ON tbl.oid = i.indrelid
		JOIN pg_namespace ns ON ns.oid = tbl.relnamespace
		JOIN pg_class idx ON idx.oid = i.indexrelid
		JOIN generate_series(1, i.indnkeyatts) AS k ON true
		WHERE tbl.relname = $1
		  AND ns.nspname = $2
		  AND NOT i.indisprimary
		GROUP BY idx.relname, i.indisunique
		ORDER BY idx.relname;
	`, []any{tableName, schema}
}

func (d *postgresDriver) IntrospectForeignKeysQuery(tableName string) (string, []any) {
	schema := d.getSchema()
	return `
		SELECT constraint_name
		FROM information_schema.table_constraints
		WHERE table_schema = $1 AND constraint_type = 'FOREIGN KEY' AND table_name = $2
	`, []any{schema, tableName}
}

func (d *postgresDriver) MapGoTypeToSQL(goType aegis.ColumnType, isAutoIncrement bool) string {
	if isAutoIncrement {
		switch goType {
		case aegis.ColumnTypeInt, aegis.ColumnTypeInt32:
			return "SERIAL"
		case aegis.ColumnTypeInt64:
			return "BIGSERIAL"
		}
	}

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
		return "TIMESTAMPTZ"
	case aegis.ColumnTypeUUID:
		return "UUID"
	case aegis.ColumnTypeMapStringAny:
		return "JSONB"
	default:
		return "TEXT"
	}
}

func (d *postgresDriver) MapSQLTypeToGoType(dataType string) aegis.ColumnType {
	switch dataType {
	case "uuid":
		return aegis.ColumnTypeUUID
	case "bool", "boolean":
		return aegis.ColumnTypeBool
	case "int4", "integer":
		return aegis.ColumnTypeInt32
	case "int8", "bigint":
		return aegis.ColumnTypeInt64
	case "timestamp", "timestamptz":
		return aegis.ColumnTypeTime
	case "jsonb", "json":
		return aegis.ColumnTypeMapStringAny
	case "varchar", "text", "char":
		return aegis.ColumnTypeString
	default:
		return aegis.ColumnTypeString
	}
}

func (d *postgresDriver) GetAutoIncrementSuffix() string {
	return ""
}

func (d *postgresDriver) FormatDefaultValue(defaultValue string) string {
	switch defaultValue {
	case string(aegis.DatabaseDefaultValueNow):
		return "CURRENT_TIMESTAMP"
	case string(aegis.DatabaseDefaultValueUUID):
		return "gen_random_uuid()"
	}
	return defaultValue
}

func (d *postgresDriver) DropIndexSQL(tableName, indexName string) string {
	return fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)
}

func (d *postgresDriver) DropForeignKeySQL(tableName, constraintName string) string {
	return fmt.Sprintf("DROP CONSTRAINT %s", constraintName)
}
