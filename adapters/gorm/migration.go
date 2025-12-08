package gorm

import (
	"fmt"
	"strings"

	"github.com/thecodearcher/aegis"
)

// gormMigrationGenerator implements MigrationGenerator for GORM
type gormMigrationGenerator struct {
	driver string // Database driver name (postgres, mysql, sqlite, etc.)
}

// NewMigrationGenerator creates a new GORM migration generator
func NewMigrationGenerator(driver string) aegis.MigrationGenerator {
	return &gormMigrationGenerator{driver: driver}
}

func (g *gormMigrationGenerator) GenerateUpMigration(schema aegis.SchemaDefinition) (string, error) {
	return g.GenerateCreateTable(schema)
}

func (g *gormMigrationGenerator) GenerateDownMigration(schema aegis.SchemaDefinition) (string, error) {
	return fmt.Sprintf("DROP TABLE IF EXISTS %s;", schema.TableName), nil
}

func (g *gormMigrationGenerator) GenerateCreateTable(schema aegis.SchemaDefinition) (string, error) {
	var buf strings.Builder

	buf.WriteString(fmt.Sprintf("CREATE TABLE IF NOT EXISTS %s (\n", schema.TableName))

	// Generate columns
	columns := make([]string, 0, len(schema.Fields))
	for _, field := range schema.Fields {
		colDef := g.generateColumnDefinition(field)
		columns = append(columns, fmt.Sprintf("  %s", colDef))
	}

	buf.WriteString(strings.Join(columns, ",\n"))

	// Generate primary key
	pkFields := make([]string, 0)
	for _, field := range schema.Fields {
		if field.IsPrimaryKey {
			pkFields = append(pkFields, field.Name)
		}
	}
	if len(pkFields) > 0 {
		buf.WriteString(fmt.Sprintf(",\n  PRIMARY KEY (%s)", strings.Join(pkFields, ", ")))
	}

	// Generate foreign keys
	for _, fk := range schema.ForeignKeys {
		buf.WriteString(fmt.Sprintf(",\n  CONSTRAINT %s FOREIGN KEY (%s) REFERENCES %s (%s)",
			fk.Name, fk.Column, fk.ReferencedTable, fk.ReferencedColumn))
		if fk.OnDelete != "" {
			buf.WriteString(fmt.Sprintf(" ON DELETE %s", fk.OnDelete))
		}
		if fk.OnUpdate != "" {
			buf.WriteString(fmt.Sprintf(" ON UPDATE %s", fk.OnUpdate))
		}
	}

	buf.WriteString("\n);\n")

	// Generate indexes
	for _, idx := range schema.Indexes {
		if idx.Unique {
			buf.WriteString(fmt.Sprintf("CREATE UNIQUE INDEX IF NOT EXISTS %s ON %s (%s);\n",
				idx.Name, schema.TableName, strings.Join(idx.Columns, ", ")))
		} else {
			buf.WriteString(fmt.Sprintf("CREATE INDEX IF NOT EXISTS %s ON %s (%s);\n",
				idx.Name, schema.TableName, strings.Join(idx.Columns, ", ")))
		}
	}

	return buf.String(), nil
}

func (g *gormMigrationGenerator) GenerateAlterTable(tableName aegis.TableName, newFields []aegis.FieldDefinition) (string, error) {
	if len(newFields) == 0 {
		return "", nil
	}

	var buf strings.Builder
	buf.WriteString(fmt.Sprintf("ALTER TABLE %s\n", tableName))

	alterStatements := make([]string, 0, len(newFields))
	for _, field := range newFields {
		colDef := g.generateColumnDefinition(field)
		alterStatements = append(alterStatements, fmt.Sprintf("  ADD COLUMN %s", colDef))
	}

	buf.WriteString(strings.Join(alterStatements, ",\n"))
	buf.WriteString(";\n")

	return buf.String(), nil
}

func (g *gormMigrationGenerator) generateColumnDefinition(field aegis.FieldDefinition) string {
	var parts []string

	parts = append(parts, field.Name)

	// Map Go types to SQL types based on driver
	sqlType := g.mapGoTypeToSQL(field.Type, field.IsNullable)
	parts = append(parts, sqlType)

	if !field.IsNullable && !field.IsPrimaryKey {
		parts = append(parts, "NOT NULL")
	}

	if field.DefaultValue != "" {
		parts = append(parts, fmt.Sprintf("DEFAULT %s", field.DefaultValue))
	}

	return strings.Join(parts, " ")
}

func (g *gormMigrationGenerator) mapGoTypeToSQL(goType string, nullable bool) string {
	// Remove pointer prefix if present
	baseType := strings.TrimPrefix(goType, "*")

	switch baseType {
	case "string":
		if g.driver == "postgres" {
			return "VARCHAR(255)"
		} else if g.driver == "mysql" {
			return "VARCHAR(255)"
		} else {
			return "TEXT"
		}
	case "int", "int32":
		return "INTEGER"
	case "int64":
		return "BIGINT"
	case "bool":
		return "BOOLEAN"
	case "time.Time":
		if g.driver == "postgres" {
			return "TIMESTAMP"
		} else {
			return "DATETIME"
		}
	case "any":
		if g.driver == "postgres" {
			return "BIGINT" // Common for IDs
		} else {
			return "BIGINT"
		}
	default:
		// For map[string]any and other complex types, use JSON/TEXT
		if strings.Contains(baseType, "map") || strings.Contains(baseType, "[]") {
			if g.driver == "postgres" {
				return "JSONB"
			} else if g.driver == "mysql" {
				return "JSON"
			} else {
				return "TEXT"
			}
		}
		return "TEXT"
	}
}
