package main

import (
	"fmt"
	"strings"

	"github.com/thecodearcher/aegis"
)

type DatabaseDriver string

const (
	DriverPostgres DatabaseDriver = "postgres"
	DriverMySQL    DatabaseDriver = "mysql"
)

type sqlMigrationGenerator struct {
	driver              Driver
	useAutoIncrementIDs bool
}

func newSQLMigrationGenerator(driver Driver, config *aegis.CliConfig) (*sqlMigrationGenerator, error) {
	return &sqlMigrationGenerator{
		driver:              driver,
		useAutoIncrementIDs: config.UseAutoIncrementID,
	}, nil
}

func (s *sqlMigrationGenerator) generateUpMigration(schema *aegis.SchemaDefinition, diff *schemaDiff) (string, error) {
	if diff != nil && diff.HasChanges() {
		return s.generateMigrationForExistingTable(schema.GetTableName(), diff)
	}

	return s.generateCreateTable(schema)
}

func (s *sqlMigrationGenerator) generateDownMigration(schema *aegis.SchemaDefinition, diff *schemaDiff) (string, error) {
	if diff != nil && diff.HasChanges() {
		return s.generateAlterDownMigration(schema.GetTableName(), diff)
	}
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;", schema.GetTableName()), nil
}

func (s *sqlMigrationGenerator) generateCreateTable(schema *aegis.SchemaDefinition) (string, error) {
	var buf strings.Builder

	fmt.Fprintf(&buf, "CREATE TABLE IF NOT EXISTS %s (\n", schema.GetTableName())

	columns := make([]string, 0, len(schema.Columns))
	for _, field := range schema.Columns {
		colDef := s.generateColumnDefinition(&field)
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
		foreignKeySQL := s.generateForeignKeyStatement(&fk, false)
		buf.WriteString(foreignKeySQL)
	}

	buf.WriteString("\n);\n")

	for _, idx := range schema.Indexes {
		fmt.Fprintf(&buf, "%s\n", s.generateCreateIndexStatement(&idx, schema.GetTableName()))
	}

	return buf.String(), nil
}

func (s *sqlMigrationGenerator) generateAlterTableStatement(tableName aegis.SchemaTableName, diff *schemaDiff) (string, error) {
	if len(diff.AddedColumns) == 0 && len(diff.AddedForeignKeys) == 0 {
		return "", nil
	}

	var buf strings.Builder
	statements := []string{}

	fmt.Fprintf(&buf, "ALTER TABLE %s\n", tableName)
	for _, col := range diff.AddedColumns {
		colDef := s.generateColumnDefinition(&col)
		statements = append(statements, fmt.Sprintf("ADD COLUMN %s", colDef))
	}

	for _, fk := range diff.AddedForeignKeys {
		foreignKeySQL := s.generateForeignKeyStatement(&fk, true)
		statements = append(statements, foreignKeySQL)
	}

	if len(statements) > 0 {
		buf.WriteString(strings.Join(statements, ",\n"))
		buf.WriteString(";\n")
	}
	return buf.String(), nil
}

func (s *sqlMigrationGenerator) generateMigrationForExistingTable(tableName aegis.SchemaTableName, diff *schemaDiff) (string, error) {
	var buf strings.Builder
	statements := []string{}

	alterTableStatement, err := s.generateAlterTableStatement(tableName, diff)
	if err != nil {
		return "", err
	}

	if alterTableStatement != "" {
		statements = append(statements, alterTableStatement)
	}

	for _, idx := range diff.AddedIndexes {
		statements = append(statements, s.generateCreateIndexStatement(&idx, tableName))
	}

	if len(statements) > 0 {
		buf.WriteString(strings.Join(statements, "\n"))
	}

	return buf.String(), nil
}

func (s *sqlMigrationGenerator) generateAlterDownMigration(tableName aegis.SchemaTableName, diff *schemaDiff) (string, error) {
	var buf strings.Builder
	statements := []string{}

	if len(diff.AddedColumns) > 0 || len(diff.AddedForeignKeys) > 0 {
		var alterOps []string

		if len(diff.AddedColumns) > 0 {
			dropColumnOps := s.generateDropColumnStatements(tableName, diff.AddedColumns)
			alterOps = append(alterOps, dropColumnOps...)
		}

		if len(diff.AddedForeignKeys) > 0 {
			dropFKOps := s.generateDropForeignKeyStatements(tableName, diff.AddedForeignKeys)
			alterOps = append(alterOps, dropFKOps...)
		}

		if len(alterOps) > 0 {
			alterStmt := fmt.Sprintf("ALTER TABLE %s\n  %s;", tableName, strings.Join(alterOps, ",\n  "))
			statements = append(statements, alterStmt)
		}
	}

	// DROP INDEX statements are separate (they use DROP INDEX syntax, not ALTER TABLE)
	if len(diff.AddedIndexes) > 0 {
		dropIndexStatements := s.generateDropIndexStatements(tableName, diff.AddedIndexes)
		statements = append(statements, dropIndexStatements...)
	}

	if len(statements) > 0 {
		buf.WriteString(strings.Join(statements, "\n"))
	}

	return buf.String(), nil
}

func (s *sqlMigrationGenerator) generateColumnDefinition(field *aegis.ColumnDefinition) string {
	parts := []string{field.Name}

	isAutoIncrement := field.LogicalField == string(aegis.SchemaIDField) && s.useAutoIncrementIDs

	sqlType := s.driver.MapGoTypeToSQL(field.Type, isAutoIncrement)
	parts = append(parts, sqlType)

	if isAutoIncrement {
		if autoIncrementSuffix := s.driver.GetAutoIncrementSuffix(); autoIncrementSuffix != "" {
			parts = append(parts, autoIncrementSuffix)
		}
	}

	if !field.IsNullable && !field.IsPrimaryKey {
		parts = append(parts, "NOT NULL")
	}

	if field.DefaultValue != "" {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", field.DefaultValue))
	}

	return strings.Join(parts, " ")
}

func (s *sqlMigrationGenerator) generateForeignKeyStatement(fk *aegis.ForeignKeyDefinition, alterTable bool) string {
	var buf strings.Builder

	if alterTable {
		fmt.Fprintf(&buf, "ADD CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
			fk.Name, fk.Column, string(fk.ReferencedSchema), string(fk.ReferencedField))
	} else {
		fmt.Fprintf(&buf, ",\nCONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
			fk.Name, fk.Column, string(fk.ReferencedSchema), string(fk.ReferencedField))
	}

	if fk.OnDelete != "" {
		fmt.Fprintf(&buf, " ON DELETE %s", fk.OnDelete)
	}

	if fk.OnUpdate != "" {
		fmt.Fprintf(&buf, " ON UPDATE %s", fk.OnUpdate)
	}

	return buf.String()
}

func (s *sqlMigrationGenerator) generateCreateIndexStatement(idx *aegis.IndexDefinition, tableName aegis.SchemaTableName) string {
	if idx.Unique {
		return fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s);",
			idx.Name, tableName, strings.Join(idx.Columns, ", "))
	}

	return fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s);",
		idx.Name, tableName, strings.Join(idx.Columns, ", "))
}

func (s *sqlMigrationGenerator) generateDropColumnStatements(tableName aegis.SchemaTableName, columns []aegis.ColumnDefinition) []string {
	statements := make([]string, 0, len(columns))
	for _, col := range columns {
		dropOp := s.driver.DropColumnSQL(string(tableName), col.Name)
		statements = append(statements, dropOp)
	}
	return statements
}

func (s *sqlMigrationGenerator) generateDropIndexStatements(tableName aegis.SchemaTableName, indexes []aegis.IndexDefinition) []string {
	statements := make([]string, 0, len(indexes))
	for _, idx := range indexes {
		dropSQL := s.driver.DropIndexSQL(string(tableName), idx.Name)
		statements = append(statements, dropSQL)
	}
	return statements
}

func (s *sqlMigrationGenerator) generateDropForeignKeyStatements(tableName aegis.SchemaTableName, foreignKeys []aegis.ForeignKeyDefinition) []string {
	statements := make([]string, 0, len(foreignKeys))
	for _, fk := range foreignKeys {
		dropOp := s.driver.DropForeignKeySQL(string(tableName), fk.Name)
		statements = append(statements, dropOp)
	}
	return statements
}
