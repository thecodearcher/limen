package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"

	"github.com/thecodearcher/aegis"
	gormadapter "github.com/thecodearcher/aegis/adapters/gorm"

	"gorm.io/driver/postgres"
	gormlib "gorm.io/gorm"
)

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	command := os.Args[1]

	switch command {
	case "introspect":
		runIntrospect()
	case "migrate":
		if len(os.Args) < 3 {
			fmt.Fprintf(os.Stderr, "Error: migrate command requires a subcommand (generate, up, down, status)\n")
			os.Exit(1)
		}
		subcommand := os.Args[2]
		switch subcommand {
		case "generate":
			runMigrateGenerate()
		case "up":
			runMigrateUp()
		case "down":
			runMigrateDown()
		case "status":
			runMigrateStatus()
		default:
			fmt.Fprintf(os.Stderr, "Error: unknown migrate subcommand: %s\n", subcommand)
			os.Exit(1)
		}
	default:
		fmt.Fprintf(os.Stderr, "Error: unknown command: %s\n", command)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Fprintf(os.Stderr, `Aegis CLI Tool

Usage:
  aegis <command> [options]

Commands:
  introspect          Generate Go structs from schemas
  migrate generate    Generate migration files
  migrate up          Apply pending migrations
  migrate down        Rollback last migration
  migrate status      Show migration status

Examples:
  aegis introspect --config ./config.go --output ./models
  aegis migrate generate --config ./config.go --output ./migrations
  aegis migrate up --config ./config.go
`)
}

func runIntrospect() {
	var (
		configPath  = flag.String("config", "", "Path to Go file containing Config struct")
		outputPath  = flag.String("output", "./models", "Output directory for generated structs")
		packageName = flag.String("package", "models", "Package name for generated code")
	)
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --config is required\n")
		os.Exit(1)
	}

	// Load config from file
	config, err := loadConfigFromFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Generate structs
	opts := aegis.GenerateOptions{
		PackageName: *packageName,
		Tags:        []string{"json", "gorm"},
	}

	code, err := aegis.GenerateGoStructsFromConfig(config, opts)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating structs: %v\n", err)
		os.Exit(1)
	}

	// Write to file
	outputFile := filepath.Join(*outputPath, "models.go")
	if err := os.MkdirAll(*outputPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	if err := os.WriteFile(outputFile, []byte(code), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "Error writing output file: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Generated structs written to %s\n", outputFile)
}

func runMigrateGenerate() {
	var (
		configPath = flag.String("config", "", "Path to Go file containing Config struct")
		outputPath = flag.String("output", "./migrations", "Output directory for migration files")
		driver     = flag.String("driver", "postgres", "Database driver (postgres, mysql, sqlite)")
	)
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --config is required\n")
		os.Exit(1)
	}

	// Load config
	config, err := loadConfigFromFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Create migration generator
	generator := gormadapter.NewMigrationGenerator(*driver)

	// Generate migrations
	migrations, err := aegis.GenerateMigrations(config, generator)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating migrations: %v\n", err)
		os.Exit(1)
	}

	// Write migration files
	if err := os.MkdirAll(*outputPath, 0755); err != nil {
		fmt.Fprintf(os.Stderr, "Error creating output directory: %v\n", err)
		os.Exit(1)
	}

	for _, migration := range migrations {
		upFile := filepath.Join(*outputPath, fmt.Sprintf("%s.up.sql", migration.Version))
		downFile := filepath.Join(*outputPath, fmt.Sprintf("%s.down.sql", migration.Version))

		if err := os.WriteFile(upFile, []byte(migration.UpSQL), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing migration file: %v\n", err)
			os.Exit(1)
		}

		if err := os.WriteFile(downFile, []byte(migration.DownSQL), 0644); err != nil {
			fmt.Fprintf(os.Stderr, "Error writing migration file: %v\n", err)
			os.Exit(1)
		}

		fmt.Printf("Generated migration: %s\n", migration.Version)
	}
}

func runMigrateUp() {
	var (
		configPath = flag.String("config", "", "Path to Go file containing Config struct")
		dsn        = flag.String("dsn", "", "Database connection string (overrides config)")
		driver     = flag.String("driver", "postgres", "Database driver")
	)
	flag.Parse()

	if *configPath == "" {
		fmt.Fprintf(os.Stderr, "Error: --config is required\n")
		os.Exit(1)
	}

	// Load config
	config, err := loadConfigFromFile(*configPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Connect to database
	db, err := connectDatabase(*driver, *dsn)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error connecting to database: %v\n", err)
		os.Exit(1)
	}

	adapter := gormadapter.New(db)
	applier := &gormMigrationApplier{adapter: adapter, db: db}

	// Generate migrations
	generator := gormadapter.NewMigrationGenerator(*driver)
	migrations, err := aegis.GenerateMigrations(config, generator)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error generating migrations: %v\n", err)
		os.Exit(1)
	}

	// Apply migrations
	ctx := context.Background()
	if err := aegis.ApplyMigrations(ctx, config, migrations, applier); err != nil {
		fmt.Fprintf(os.Stderr, "Error applying migrations: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Migrations applied successfully")
}

func runMigrateDown() {
	fmt.Fprintf(os.Stderr, "Error: migrate down is not yet implemented\n")
	os.Exit(1)
}

func runMigrateStatus() {
	fmt.Fprintf(os.Stderr, "Error: migrate status is not yet implemented\n")
	os.Exit(1)
}

// loadConfigFromFile loads a Config from a Go file
// This is a simplified version - in production, you'd use go/parser and go/types
// For now, we'll require users to provide a function that returns the config
func loadConfigFromFile(filePath string) (*aegis.Config, error) {
	// Note: This is a placeholder. In a real implementation, you would:
	// 1. Parse the Go file using go/parser
	// 2. Extract the Config struct initialization
	// 3. Evaluate it to get the actual Config value
	// For now, we'll return an error asking users to use the programmatic API
	return nil, fmt.Errorf("loading config from file is not yet implemented. Please use the programmatic API or provide a function that returns the config")
}

// connectDatabase connects to a database using the specified driver and DSN
func connectDatabase(driver, dsn string) (*gormlib.DB, error) {
	var dialector gormlib.Dialector

	switch driver {
	case "postgres":
		dialector = postgres.Open(dsn)
	case "mysql":
		// MySQL driver would be imported here if needed
		return nil, fmt.Errorf("mysql driver not yet supported in CLI")
	case "sqlite":
		// SQLite driver would be imported here if needed
		return nil, fmt.Errorf("sqlite driver not yet supported in CLI")
	default:
		return nil, fmt.Errorf("unsupported driver: %s", driver)
	}

	return gormlib.Open(dialector, &gormlib.Config{})
}

// gormMigrationApplier implements MigrationApplier for GORM
type gormMigrationApplier struct {
	adapter *gormadapter.Adapter
	db      *gormlib.DB
}

func (g *gormMigrationApplier) ApplyMigration(ctx context.Context, sql string) error {
	return g.db.WithContext(ctx).Exec(sql).Error
}

func (g *gormMigrationApplier) RollbackMigration(ctx context.Context, sql string) error {
	return g.db.WithContext(ctx).Exec(sql).Error
}

func (g *gormMigrationApplier) GetAppliedMigrations(ctx context.Context) ([]string, error) {
	// Ensure migrations table exists
	if err := g.EnsureMigrationTable(ctx); err != nil {
		return nil, err
	}

	var versions []string
	err := g.db.WithContext(ctx).
		Table("aegis_migrations").
		Select("version").
		Order("version ASC").
		Pluck("version", &versions).Error

	return versions, err
}

func (g *gormMigrationApplier) RecordMigration(ctx context.Context, migration aegis.Migration) error {
	if err := g.EnsureMigrationTable(ctx); err != nil {
		return err
	}

	return g.db.WithContext(ctx).Table("aegis_migrations").Create(map[string]any{
		"version":    migration.Version,
		"name":       migration.Name,
		"applied_at": migration.AppliedAt,
	}).Error
}

func (g *gormMigrationApplier) RemoveMigration(ctx context.Context, version string) error {
	return g.db.WithContext(ctx).
		Table("aegis_migrations").
		Where("version = ?", version).
		Delete(nil).Error
}

func (g *gormMigrationApplier) EnsureMigrationTable(ctx context.Context) error {
	return g.db.WithContext(ctx).Exec(`
		CREATE TABLE IF NOT EXISTS aegis_migrations (
			version VARCHAR(255) PRIMARY KEY,
			name VARCHAR(255) NOT NULL,
			applied_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`).Error
}
