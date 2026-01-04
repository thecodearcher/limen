package main

import (
	"context"
	"time"

	"github.com/thecodearcher/aegis"
)

// Migration represents a single database migration
type Migration struct {
	Version   string     // Migration version (timestamp or version number)
	Name      string     // Migration name
	UpSQL     string     // SQL to apply the migration
	DownSQL   string     // SQL to rollback the migration
	AppliedAt *time.Time // When this migration was applied (nil if not applied)
}

// MigrationGenerator generates migration SQL from schema definitions
type MigrationGenerator interface {
	// GenerateUpMigration generates SQL to create/alter a table based on schema definition
	GenerateUpMigration(schema aegis.SchemaDefinition) (string, error)
	// GenerateDownMigration generates SQL to drop/alter a table back
	GenerateDownMigration(schema aegis.SchemaDefinition) (string, error)
	// GenerateCreateTable generates CREATE TABLE SQL
	GenerateCreateTable(schema aegis.SchemaDefinition) (string, error)
	// GenerateAlterTable generates ALTER TABLE SQL for adding columns
	GenerateAlterTable(tableName aegis.SchemaTableName, newFields []aegis.ColumnDefinition) (string, error)
}

// MigrationApplier applies migrations to the database
type MigrationApplier interface {
	// ApplyMigration applies a migration SQL to the database
	ApplyMigration(ctx context.Context, sql string) error
	// RollbackMigration rolls back a migration SQL
	RollbackMigration(ctx context.Context, sql string) error
	// GetAppliedMigrations returns list of applied migration versions
	GetAppliedMigrations(ctx context.Context) ([]string, error)
	// RecordMigration records that a migration was applied
	RecordMigration(ctx context.Context, migration Migration) error
	// RemoveMigration removes a migration record
	RemoveMigration(ctx context.Context, version string) error
}

// MigrationTracker manages migration state in the database
type MigrationTracker interface {
	// EnsureMigrationTable ensures the migrations table exists
	EnsureMigrationTable(ctx context.Context) error
	// GetAppliedMigrations returns all applied migrations
	GetAppliedMigrations(ctx context.Context) ([]Migration, error)
	// RecordMigration records a migration as applied
	RecordMigration(ctx context.Context, migration Migration) error
	// RemoveMigration removes a migration record
	RemoveMigration(ctx context.Context, version string) error
}
