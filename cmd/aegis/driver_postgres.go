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
		SELECT 
			column_name,
			udt_name as data_type,
			is_nullable,
			column_default,
			(SELECT COUNT(*) FROM information_schema.table_constraints tc
			 JOIN information_schema.constraint_column_usage ccu ON tc.constraint_name = ccu.constraint_name
			 WHERE tc.table_schema = $1 AND tc.table_name = $2 AND tc.constraint_type = 'PRIMARY KEY' AND ccu.column_name = c.column_name) > 0 as is_primary_key
		FROM information_schema.columns c
		WHERE table_schema = $1 AND table_name = $2
		ORDER BY ordinal_position
	`, []any{schema, tableName}
}

func (d *postgresDriver) IntrospectIndexesQuery(tableName string) (string, []any) {
	schema := d.getSchema()
	return `
		SELECT 
			i.indexname,
			string_agg(a.attname, ',' ORDER BY array_position(ix.indkey, a.attnum)) as columns,
			i.indexdef LIKE '%UNIQUE%' as is_unique
		FROM pg_indexes i
		JOIN pg_index ix ON i.indexname = (SELECT relname FROM pg_class WHERE oid = ix.indexrelid)
		JOIN pg_attribute a ON a.attrelid = ix.indrelid AND a.attnum = ANY(ix.indkey)
		WHERE i.tablename = $1 AND i.schemaname = $2
		GROUP BY i.indexname, i.indexdef
		HAVING NOT (i.indexname LIKE '%_pkey')
	`, []any{tableName, schema}
}

func (d *postgresDriver) IntrospectForeignKeysQuery(tableName string) (string, []any) {
	schema := d.getSchema()
	return `
		SELECT
			tc.constraint_name,
			kcu.column_name,
			ccu.table_name AS foreign_table_name,
			ccu.column_name AS foreign_column_name,
			rc.delete_rule,
			rc.update_rule
		FROM information_schema.table_constraints AS tc
		JOIN information_schema.key_column_usage AS kcu
			ON tc.constraint_name = kcu.constraint_name
		JOIN information_schema.constraint_column_usage AS ccu
			ON ccu.constraint_name = tc.constraint_name
		JOIN information_schema.referential_constraints AS rc
			ON rc.constraint_name = tc.constraint_name
		WHERE tc.table_schema = $1 AND tc.constraint_type = 'FOREIGN KEY' AND tc.table_name = $2
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
		return "TIMESTAMP"
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

func (d *postgresDriver) DropIndexSQL(tableName, indexName string) string {
	return fmt.Sprintf("DROP INDEX IF EXISTS %s", indexName)
}

func (d *postgresDriver) DropForeignKeySQL(tableName, constraintName string) string {
	return fmt.Sprintf("DROP CONSTRAINT %s", constraintName)
}

func (d *postgresDriver) ParseColumnRow(scan func(dest ...any) error) (aegis.ColumnDefinition, error) {
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
