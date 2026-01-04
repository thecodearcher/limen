package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/thecodearcher/aegis"
)

// GenerateOptions configures how Go structs are generated
type GenerateOptions struct {
	PackageName string              // Package name for generated code
	Tags        []string            // Tags to include (json, gorm, sql, etc.)
	FieldNaming func(string) string // Function to convert field names to Go field names
}

func defaultFieldNaming(dbName string) string {
	// Convert snake_case to PascalCase
	parts := strings.Split(dbName, "_")
	var result strings.Builder
	for _, part := range parts {
		if len(part) > 0 {
			result.WriteString(strings.ToUpper(part[:1]) + part[1:])
		}
	}
	return result.String()
}

// GenerateGoStructs generates Go struct definitions from schema definitions
func GenerateGoStructs(schemas map[string]aegis.SchemaDefinition, opts GenerateOptions) string {
	var buf strings.Builder

	fmt.Fprintf(&buf, "package %s\n\n", opts.PackageName)

	buf.WriteString("import (\n")
	buf.WriteString("\t\"time\"\n")
	buf.WriteString(")\n\n")

	// this helps give a deterministic order to the models
	sortedKeys := slices.Sorted(maps.Keys(schemas))
	for _, key := range sortedKeys {
		schema := schemas[key]
		generateSchemaStruct(&buf, schema, opts)
		for _, field := range schema.Columns {
			generateStructField(&buf, field, opts)
		}

		// close the struct
		buf.WriteString("}\n\n")
	}

	return buf.String()
}

func generateSchemaStruct(buf *strings.Builder, schema aegis.SchemaDefinition, opts GenerateOptions) {
	tableName := schema.GetTableName()
	structName := opts.FieldNaming(string(tableName))

	fmt.Fprintf(buf, "// %s represents the %s table\n", structName, tableName)
	if schema.PluginName != "" {
		fmt.Fprintf(buf, "// This schema is provided by plugin: %s\n", schema.PluginName)
	}

	fmt.Fprintf(buf, "type %s struct {\n", structName)
}

func generateStructField(buf *strings.Builder, field aegis.ColumnDefinition, opts GenerateOptions) {
	goFieldName := opts.FieldNaming(field.Name)

	// Build tags
	tagParts := make([]string, 0, len(opts.Tags))
	for _, tagName := range opts.Tags {
		if tagValue, exists := field.Tags[tagName]; exists {
			tagParts = append(tagParts, fmt.Sprintf("%s:%q", tagName, tagValue))
		} else {
			// Default tag value
			tagParts = append(tagParts, fmt.Sprintf("%s:%q", tagName, field.Name))
		}
	}

	tags := strings.Join(tagParts, " ")

	// Determine Go type
	goType := string(field.Type)
	if field.IsNullable && !strings.HasPrefix(goType, "*") && goType != "any" {
		goType = "*" + goType
	}

	// Write field
	fmt.Fprintf(buf, "\t%s %s `%s`", goFieldName, goType, tags)
	if field.IsPrimaryKey {
		buf.WriteString(" // primary key")
	}
	buf.WriteString("\n")
}
