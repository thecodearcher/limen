package main

import (
	"database/sql"
	"fmt"
	"time"

	"github.com/thecodearcher/limen"
)

type Migration struct {
	Version string // Migration version timestamp (YYYYMMDDHHMMSS)
	UpSQL   string // SQL to apply the migration
	DownSQL string // SQL to rollback the migration
}

func generateMigrations(db *sql.DB, driver Driver, config *cliConfig) ([]Migration, error) {
	migrations := make([]Migration, 0, len(config.Schemas))
	timestamp := time.Now().Format("20060102150405")

	introspector := newSchemaIntrospector(db, driver)
	tableNames := make([]string, 0, len(config.Schemas))
	for _, schema := range config.Schemas {
		tableNames = append(tableNames, string(schema.GetTableName()))
	}

	existingTables, err := introspector.getTables(tableNames)
	if err != nil {
		return nil, fmt.Errorf("failed to get existing tables: %w", err)
	}

	generator, err := newSQLMigrationGenerator(driver, config)
	if err != nil {
		return nil, fmt.Errorf("failed to create migration generator: %w", err)
	}

	for schemaName, schemaDef := range config.Schemas {
		var diff *schemaDiff

		if existingTables[string(schemaDef.GetTableName())] {
			diff, err = generateDiffForTable(introspector, &schemaDef)
			if err != nil {
				return nil, fmt.Errorf("failed to generate schema diff for %s: %w", schemaName, err)
			}

			if !diff.HasChanges() {
				continue
			}
		}

		upSQL, err := generator.generateUpMigration(&schemaDef, diff)
		if err != nil {
			return nil, fmt.Errorf("failed to generate up migration for %s: %w", schemaName, err)
		}

		downSQL, err := generator.generateDownMigration(&schemaDef, diff)
		if err != nil {
			return nil, fmt.Errorf("failed to generate down migration for %s: %w", schemaName, err)
		}

		migration := Migration{
			Version: fmt.Sprintf("%s_%s", timestamp, schemaName),
			UpSQL:   upSQL,
			DownSQL: downSQL,
		}

		migrations = append(migrations, migration)
	}

	return migrations, nil
}

func generateDiffForTable(introspector *schemaIntrospector, schema *limen.SchemaDefinition) (*schemaDiff, error) {
	existingSchema, err := introspector.introspectTable(schema.GetTableName())
	if err != nil {
		return nil, err
	}
	diff := compareSchemas(existingSchema, schema)
	return &diff, nil
}
