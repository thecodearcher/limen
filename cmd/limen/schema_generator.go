package main

import (
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/thecodearcher/limen"
)

type GenerateOptions struct {
	PackageName string              // Package name for generated code
	Tags        []string            // Tags to include (json, gorm, sql, etc.)
	FieldNaming func(string) string // Function to convert field names to Go field names
}

// GenerateGoStructs generates Go struct definitions from schema definitions
func GenerateGoStructs(schemas limen.SchemaDefinitionMap, opts GenerateOptions) string {
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

		buf.WriteString("}\n\n")
	}

	return buf.String()
}

func generateSchemaStruct(buf *strings.Builder, schema limen.SchemaDefinition, opts GenerateOptions) {
	tableName := schema.GetTableName()
	structName := opts.FieldNaming(string(tableName))

	fmt.Fprintf(buf, "// %s represents the %s table\n", structName, tableName)
	if schema.PluginName != "" {
		fmt.Fprintf(buf, "// This schema is provided by plugin: %s\n", schema.PluginName)
	}

	fmt.Fprintf(buf, "type %s struct {\n", structName)
}

func generateStructField(buf *strings.Builder, field limen.ColumnDefinition, opts GenerateOptions) {
	goFieldName := opts.FieldNaming(field.Name)

	tagParts := make([]string, 0, len(opts.Tags))
	for _, tagName := range opts.Tags {
		tagValue := field.Tags[tagName]
		if tagValue == "" {
			tagValue = field.Name
		}
		tagParts = append(tagParts, fmt.Sprintf("%s:%q", tagName, tagValue))
	}

	tags := strings.Join(tagParts, " ")

	goType := string(field.Type)
	if field.IsNullable && !strings.HasPrefix(goType, "*") && goType != "any" {
		goType = "*" + goType
	}

	fmt.Fprintf(buf, "\t%s %s `%s`", goFieldName, goType, tags)
	if field.IsPrimaryKey {
		buf.WriteString(" // primary key")
	}
	buf.WriteString("\n")
}
