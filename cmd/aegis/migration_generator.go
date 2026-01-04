package main

import (
	"fmt"
	"strings"

	"github.com/thecodearcher/aegis"
)

type DatabaseDriver string

// Database driver constants
const (
	DriverPostgres   DatabaseDriver = "postgres"
	DriverPostgreSQL DatabaseDriver = "postgresql"
	DriverMySQL      DatabaseDriver = "mysql"
	DriverMariaDB    DatabaseDriver = "mariadb"
	DriverSQLite     DatabaseDriver = "sqlite"
	DriverSQLite3    DatabaseDriver = "sqlite3"
	DriverSQLServer  DatabaseDriver = "sqlserver"
	DriverMSSQL      DatabaseDriver = "mssql"
)

// sqlMigrationGenerator implements MigrationGenerator for SQL-based databases
type sqlMigrationGenerator struct {
	driver              DatabaseDriver // Database driver name (postgres, mysql, sqlite, etc.)
	useAutoIncrementIDs bool           // Whether ID fields should use auto-increment
}

// NewSQLMigrationGenerator creates a new SQL migration generator
func NewSQLMigrationGenerator(driver DatabaseDriver, useAutoIncrementIDs bool) MigrationGenerator {
	return &sqlMigrationGenerator{
		driver:              driver,
		useAutoIncrementIDs: useAutoIncrementIDs,
	}
}

func (g *sqlMigrationGenerator) GenerateUpMigration(schema aegis.SchemaDefinition) (string, error) {
	return g.GenerateCreateTable(schema)
}

func (g *sqlMigrationGenerator) GenerateDownMigration(schema aegis.SchemaDefinition) (string, error) {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;", schema.GetTableName()), nil
}

func (g *sqlMigrationGenerator) GenerateCreateTable(schema aegis.SchemaDefinition) (string, error) {
	var buf strings.Builder

	fmt.Fprintf(&buf, "CREATE TABLE IF NOT EXISTS %s (\n", schema.GetTableName())

	columns := make([]string, 0, len(schema.Columns))
	for _, field := range schema.Columns {
		colDef := g.generateColumnDefinition(field)
		columns = append(columns, fmt.Sprintf("  %s", colDef))
	}

	buf.WriteString(strings.Join(columns, ",\n"))

	pkFields := make([]string, 0)
	for _, field := range schema.Columns {
		if field.IsPrimaryKey {
			pkFields = append(pkFields, field.Name)
		}
	}

	if len(pkFields) > 0 {
		fmt.Fprintf(&buf, ",\n  PRIMARY KEY (%s)", strings.Join(pkFields, ", "))
	}

	for _, fk := range schema.ForeignKeys {
		g.generateForeignKeyDefinition(&buf, fk)
	}

	buf.WriteString("\n);\n")

	for _, idx := range schema.Indexes {
		g.generateIndexDefinition(&buf, idx, schema.GetTableName())
	}

	return buf.String(), nil
}

func (g *sqlMigrationGenerator) GenerateAlterTable(tableName aegis.SchemaTableName, newFields []aegis.ColumnDefinition) (string, error) {
	if len(newFields) == 0 {
		return "", nil
	}

	var buf strings.Builder
	fmt.Fprintf(&buf, "ALTER TABLE %s\n", tableName)

	alterStatements := make([]string, 0, len(newFields))
	for _, field := range newFields {
		colDef := g.generateColumnDefinition(field)
		alterStatements = append(alterStatements, fmt.Sprintf("  ADD COLUMN %s", colDef))
	}

	buf.WriteString(strings.Join(alterStatements, ",\n"))
	buf.WriteString(";\n")

	return buf.String(), nil
}

func (g *sqlMigrationGenerator) generateColumnDefinition(field aegis.ColumnDefinition) string {
	var parts []string

	parts = append(parts, field.Name)

	isAutoIncrement := field.LogicalField == string(aegis.SchemaIDField) && g.useAutoIncrementIDs

	sqlType := g.mapGoTypeToSQL(field.Type, isAutoIncrement)
	parts = append(parts, sqlType)

	// Add auto-increment suffix for non-PostgreSQL databases
	if autoIncrementSuffix := g.getAutoIncrementSuffix(isAutoIncrement); autoIncrementSuffix != "" {
		parts = append(parts, autoIncrementSuffix)
	}

	if !field.IsNullable && !field.IsPrimaryKey {
		parts = append(parts, "NOT NULL")
	}

	if field.DefaultValue != "" {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", field.DefaultValue))
	}

	return strings.Join(parts, " ")
}

func (g *sqlMigrationGenerator) generateForeignKeyDefinition(buf *strings.Builder, fk aegis.ForeignKeyDefinition) {
	fmt.Fprintf(buf, ",\n  CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
		fk.Name, fk.Column, string(fk.ReferencedSchema), string(fk.ReferencedField))

	if fk.OnDelete != "" {
		fmt.Fprintf(buf, " ON DELETE %s", fk.OnDelete)
	}

	if fk.OnUpdate != "" {
		fmt.Fprintf(buf, " ON UPDATE %s", fk.OnUpdate)
	}
}

func (g *sqlMigrationGenerator) generateIndexDefinition(buf *strings.Builder, idx aegis.IndexDefinition, tableName aegis.SchemaTableName) {
	if idx.Unique {
		fmt.Fprintf(buf, "CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s);\n",
			idx.Name, tableName, strings.Join(idx.Columns, ", "))
		return
	}

	fmt.Fprintf(buf, "CREATE INDEX IF NOT EXISTS %s ON %s (%s);\n",
		idx.Name, tableName, strings.Join(idx.Columns, ", "))
}

// mapGoTypeToSQL maps a Go type to its SQL type representation
// If isAutoIncrement is true, PostgreSQL will use SERIAL/BIGSERIAL instead of INTEGER/BIGINT
func (g *sqlMigrationGenerator) mapGoTypeToSQL(goType aegis.ColumnType, isAutoIncrement bool) string {
	if isAutoIncrement && (g.driver == DriverPostgres || g.driver == DriverPostgreSQL) {
		switch goType {
		case aegis.ColumnTypeInt, aegis.ColumnTypeInt32:
			return "SERIAL"
		case aegis.ColumnTypeInt64:
			return "BIGSERIAL"
		}
	}

	// Check common types that are the same across all drivers
	if sqlType, ok := mapCommonType(goType); ok {
		return sqlType
	}

	switch goType {
	case aegis.ColumnTypeString:
		return mapStringType(g.driver)
	case aegis.ColumnTypeTime:
		return mapTimeType(g.driver)
	case aegis.ColumnTypeUUID:
		return mapUUIDType(g.driver)
	case aegis.ColumnTypeMapStringAny:
		return mapComplexType(g.driver)
	default:
		return "TEXT"
	}
}

// getAutoIncrementSuffix returns the SQL suffix for auto-increment based on the driver
// Returns empty string for PostgreSQL (which uses SERIAL types) or if not auto-increment
func (g *sqlMigrationGenerator) getAutoIncrementSuffix(isAutoIncrement bool) string {
	if !isAutoIncrement {
		return ""
	}

	switch g.driver {
	case DriverMySQL, DriverMariaDB:
		return "AUTO_INCREMENT"
	case DriverSQLite, DriverSQLite3:
		return "AUTOINCREMENT"
	case DriverSQLServer, DriverMSSQL:
		return "IDENTITY(1,1)"
	default:
		return ""
	}
}

// mapCommonType returns the SQL type for types that are the same across all drivers
func mapCommonType(goType aegis.ColumnType) (sqlType string, ok bool) {
	switch goType {
	case aegis.ColumnTypeInt, aegis.ColumnTypeInt32:
		return "INTEGER", true
	case aegis.ColumnTypeInt64:
		return "BIGINT", true
	case aegis.ColumnTypeBool:
		return "BOOLEAN", true
	default:
		return "", false
	}
}

// mapStringType returns the SQL type for Go string type based on driver
func mapStringType(driver DatabaseDriver) string {
	switch driver {
	case DriverPostgres, DriverPostgreSQL, DriverMySQL, DriverMariaDB:
		return "VARCHAR(255)"
	default:
		return "TEXT"
	}
}

// mapUUIDType returns the SQL type for Go uuid.UUID type based on driver
func mapUUIDType(driver DatabaseDriver) string {
	switch driver {
	case DriverPostgres, DriverPostgreSQL:
		return "UUID"
	case DriverMySQL, DriverMariaDB:
		return "VARCHAR(36)"
	default:
		return "TEXT"
	}
}

// mapTimeType returns the SQL type for Go time.Time type based on driver
func mapTimeType(driver DatabaseDriver) string {
	switch driver {
	case DriverPostgres, DriverPostgreSQL:
		return "TIMESTAMP"
	default:
		return "DATETIME"
	}
}

// mapComplexType returns the SQL type for complex Go types (maps, slices) based on driver
func mapComplexType(driver DatabaseDriver) string {
	switch driver {
	case DriverPostgres, DriverPostgreSQL:
		return "JSONB"
	case DriverMySQL, DriverMariaDB:
		return "JSON"
	default:
		return "TEXT"
	}
}
