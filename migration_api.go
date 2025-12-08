package aegis

import (
	"context"
	"fmt"
	"time"
)

// IntrospectSchemas introspects all schemas from a Config struct
func IntrospectSchemas(config *Config) (map[string]SchemaDefinition, error) {
	return DiscoverAllSchemasFromConfig(config)
}

// GenerateGoStructsFromConfig generates Go struct definitions from a Config
func GenerateGoStructsFromConfig(config *Config, opts GenerateOptions) (string, error) {
	schemas, err := IntrospectSchemas(config)
	if err != nil {
		return "", fmt.Errorf("failed to introspect schemas: %w", err)
	}

	return GenerateGoStructs(schemas, opts)
}

// GenerateMigrations generates migration SQL from schema definitions
func GenerateMigrations(config *Config, generator MigrationGenerator) ([]Migration, error) {
	schemas, err := IntrospectSchemas(config)
	if err != nil {
		return nil, fmt.Errorf("failed to introspect schemas: %w", err)
	}

	migrations := make([]Migration, 0, len(schemas))
	timestamp := time.Now().Format("20060102150405")

	for schemaName, schema := range schemas {
		upSQL, err := generator.GenerateUpMigration(schema)
		if err != nil {
			return nil, fmt.Errorf("failed to generate up migration for %s: %w", schemaName, err)
		}

		downSQL, err := generator.GenerateDownMigration(schema)
		if err != nil {
			return nil, fmt.Errorf("failed to generate down migration for %s: %w", schemaName, err)
		}

		migration := Migration{
			Version: fmt.Sprintf("%s_%s", timestamp, schemaName),
			Name:    fmt.Sprintf("create_%s", schemaName),
			UpSQL:   upSQL,
			DownSQL: downSQL,
		}

		migrations = append(migrations, migration)
	}

	return migrations, nil
}

// ApplyMigrations applies a list of migrations to the database
func ApplyMigrations(ctx context.Context, config *Config, migrations []Migration, applier MigrationApplier) error {
	// Get already applied migrations
	applied, err := applier.GetAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	appliedSet := make(map[string]bool)
	for _, v := range applied {
		appliedSet[v] = true
	}

	// Apply pending migrations
	for _, migration := range migrations {
		if appliedSet[migration.Version] {
			continue // Already applied
		}

		if err := applier.ApplyMigration(ctx, migration.UpSQL); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.Version, err)
		}

		now := time.Now()
		migration.AppliedAt = &now
		if err := applier.RecordMigration(ctx, migration); err != nil {
			return fmt.Errorf("failed to record migration %s: %w", migration.Version, err)
		}
	}

	return nil
}

// RollbackLastMigration rolls back the last applied migration
func RollbackLastMigration(ctx context.Context, config *Config, applier MigrationApplier) error {
	applied, err := applier.GetAppliedMigrations(ctx)
	if err != nil {
		return fmt.Errorf("failed to get applied migrations: %w", err)
	}

	if len(applied) == 0 {
		return fmt.Errorf("no migrations to rollback")
	}

	// Find the migration definition (this would need to be stored or retrieved)
	// For now, we'll need the migration to be passed in or retrieved from storage
	// lastVersion := applied[len(applied)-1] // Reserved for future use
	return fmt.Errorf("rollback requires migration definition - not yet implemented")
}
