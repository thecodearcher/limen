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

func (s *sqlMigrationGenerator) generateMigrationForExistingTable(tableName aegis.SchemaTableName, diff *schemaDiff) (string, error) {
	var buf strings.Builder
	statements := []string{}

	alterTableStatement, err := s.generateUpAlterTableStatement(tableName, diff)
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

	for _, idx := range diff.AddedIndexes {
		dropSQL := s.driver.DropIndexSQL(string(tableName), idx.Name)
		statements = append(statements, dropSQL)
	}

	downAlterTableStatement, err := s.generateDownAlterTableStatement(tableName, diff)
	if err != nil {
		return "", err
	}

	if downAlterTableStatement != "" {
		statements = append(statements, downAlterTableStatement)
	}

	if len(statements) > 0 {
		buf.WriteString(strings.Join(statements, "\n"))
	}

	return buf.String(), nil
}

func (s *sqlMigrationGenerator) generateUpAlterTableStatement(tableName aegis.SchemaTableName, diff *schemaDiff) (string, error) {
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

func (s *sqlMigrationGenerator) generateDownAlterTableStatement(tableName aegis.SchemaTableName, diff *schemaDiff) (string, error) {
	if len(diff.AddedColumns) == 0 && len(diff.AddedForeignKeys) == 0 {
		return "", nil
	}

	var buf strings.Builder
	statements := []string{}

	fmt.Fprintf(&buf, "ALTER TABLE %s\n", tableName)

	for _, fk := range diff.AddedForeignKeys {
		dropOp := s.driver.DropForeignKeySQL(string(tableName), fk.Name)
		statements = append(statements, dropOp)
	}

	for _, col := range diff.AddedColumns {
		dropOp := s.driver.DropColumnSQL(string(tableName), col.Name)
		statements = append(statements, dropOp)
	}

	if len(statements) > 0 {
		buf.WriteString(strings.Join(statements, ",\n"))
		buf.WriteString(";\n")
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
		defaultValue := field.DefaultValue
		if checkForSpecialSyntaxPatterns(defaultValue) {
			defaultValue = strings.TrimPrefix(s.driver.FormatDefaultValue(defaultValue), aegis.DatabaseDefaultValuePrefix)
		}
		parts = append(parts, fmt.Sprintf("DEFAULT %s", defaultValue))
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
		return fmt.Sprintf("CREATE UNIQUE INDEX %s ON %s (%s);",
			idx.Name, tableName, strings.Join(idx.Columns, ", "))
	}

	return fmt.Sprintf("CREATE INDEX %s ON %s (%s);",
		idx.Name, tableName, strings.Join(idx.Columns, ", "))
}
