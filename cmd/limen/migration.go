package main

import (
	"database/sql"
	"fmt"
	"maps"
	"os"
	"slices"
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
	baseTime := time.Now().UTC()

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

	for _, schemaName := range orderSchemasByDependency(config.Schemas) {
		schemaDef := config.Schemas[schemaName]

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
			Version: migrationVersion(baseTime, len(migrations), string(schemaName)),
			UpSQL:   upSQL,
			DownSQL: downSQL,
		}

		migrations = append(migrations, migration)
	}

	return migrations, nil
}

// migrationVersion builds a unique, sortable version prefix for a migration. seq
// is the zero-based position of the migration within a single run; each increment
// advances the timestamp by one second so tools that order by the numeric prefix
// (golang-migrate, goose) see distinct, increasing versions.
func migrationVersion(base time.Time, seq int, schemaName string) string {
	ts := base.Add(time.Duration(seq) * time.Second).Format("20060102150405")
	return fmt.Sprintf("%s_%s", ts, schemaName)
}

func orderSchemasByDependency(schemas limen.SchemaDefinitionMap) []limen.SchemaName {
	deps := schemaDependencies(schemas)

	const (
		visiting = 1
		visited  = 2
	)

	ordered := make([]limen.SchemaName, 0, len(schemas))
	state := make(map[limen.SchemaName]int, len(schemas))

	// Depth-first post-order: a schema is appended only after its dependencies, so
	// referenced tables land earlier in the result.
	var visit func(name limen.SchemaName)
	visit = func(name limen.SchemaName) {
		switch state[name] {
		case visited:
			return
		case visiting:
			fmt.Fprintf(os.Stderr,
				"warning: foreign-key cycle involving schema %q; review its migration order manually\n", name)
			return
		}

		state[name] = visiting
		for _, dep := range deps[name] {
			visit(dep)
		}

		state[name] = visited
		ordered = append(ordered, name)
	}

	// Visit roots alphabetically so independent schemas stay alphabetical.
	for _, name := range slices.Sorted(maps.Keys(schemas)) {
		visit(name)
	}

	return ordered
}

// schemaDependencies maps each schema to the schemas it references via foreign keys,
// sorted alphabetically. External-table references and self-references are dropped.
func schemaDependencies(schemas limen.SchemaDefinitionMap) map[limen.SchemaName][]limen.SchemaName {
	// A resolved foreign key's ReferencedSchema holds the actual table name, so map
	// table names back to their owning schema key.
	tableToName := make(map[limen.SchemaTableName]limen.SchemaName, len(schemas))
	for name, def := range schemas {
		tableToName[def.GetTableName()] = name
	}

	deps := make(map[limen.SchemaName][]limen.SchemaName, len(schemas))
	for name, def := range schemas {
		for _, fk := range def.ForeignKeys {
			ref, ok := tableToName[limen.SchemaTableName(fk.ReferencedSchema)]
			if !ok || ref == name {
				continue
			}

			deps[name] = append(deps[name], ref)
		}

		slices.Sort(deps[name])
	}

	return deps
}

func generateDiffForTable(introspector *schemaIntrospector, schema *limen.SchemaDefinition) (*schemaDiff, error) {
	existingSchema, err := introspector.introspectTable(schema.GetTableName())
	if err != nil {
		return nil, err
	}
	diff := compareSchemas(existingSchema, schema)
	return &diff, nil
}
