package aegis

import (
	"fmt"
	"strings"
)

// GenerateOptions configures how Go structs are generated
type GenerateOptions struct {
	PackageName string              // Package name for generated code
	Tags        []string            // Tags to include (json, gorm, sql, etc.)
	FieldNaming func(string) string // Function to convert field names to Go field names
}

// DefaultGenerateOptions returns default options for struct generation
func DefaultGenerateOptions() GenerateOptions {
	return GenerateOptions{
		PackageName: "models",
		Tags:        []string{"json"},
		FieldNaming: defaultFieldNaming,
	}
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
func GenerateGoStructs(schemas map[string]SchemaDefinition, opts GenerateOptions) (string, error) {
	if opts.PackageName == "" {
		opts.PackageName = "models"
	}
	if opts.FieldNaming == nil {
		opts.FieldNaming = defaultFieldNaming
	}
	if len(opts.Tags) == 0 {
		opts.Tags = []string{"json"}
	}

	var buf strings.Builder

	// Write package declaration
	buf.WriteString(fmt.Sprintf("package %s\n\n", opts.PackageName))

	// Write imports
	buf.WriteString("import (\n")
	buf.WriteString("\t\"time\"\n")
	buf.WriteString(")\n\n")

	// Generate struct for each schema
	for schemaName, schema := range schemas {
		structName := opts.FieldNaming(string(schema.TableName))
		if structName == "" {
			structName = opts.FieldNaming(schemaName)
		}

		buf.WriteString(fmt.Sprintf("// %s represents the %s table\n", structName, schema.TableName))
		if schema.PluginName != "" {
			buf.WriteString(fmt.Sprintf("// This schema is provided by plugin: %s\n", schema.PluginName))
		}
		if schema.Extends != nil {
			buf.WriteString(fmt.Sprintf("// This schema extends: %s\n", string(*schema.Extends)))
		}
		buf.WriteString(fmt.Sprintf("type %s struct {\n", structName))

		// Generate fields
		for _, field := range schema.Columns {
			goFieldName := opts.FieldNaming(field.Name)
			if goFieldName == "" {
				goFieldName = field.Name
			}

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
			buf.WriteString(fmt.Sprintf("\t%s %s `%s`", goFieldName, goType, tags))
			if field.IsPrimaryKey {
				buf.WriteString(" // primary key")
			}
			buf.WriteString("\n")
		}

		buf.WriteString("}\n\n")
	}

	return buf.String(), nil
}
