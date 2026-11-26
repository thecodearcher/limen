package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type SqlFileWriter interface {
	WriteSqlFile(outputPath string, migration Migration) error
}

type GolangMigrateCompatibleSqlWriter struct{}

func (w *GolangMigrateCompatibleSqlWriter) WriteSqlFile(outputPath string, migration Migration) error {
	upFile := filepath.Join(outputPath, fmt.Sprintf("%s.up.sql", migration.Version))
	downFile := filepath.Join(outputPath, fmt.Sprintf("%s.down.sql", migration.Version))

	if err := os.WriteFile(upFile, []byte(migration.UpSQL), 0644); err != nil {
		return fmt.Errorf("error writing migration file: %w", err)
	}

	if err := os.WriteFile(downFile, []byte(migration.DownSQL), 0644); err != nil {
		return fmt.Errorf("error writing migration file: %w", err)
	}

	return nil
}

type GooseCompatibleSqlWriter struct{}

const gooseMigrationUpAnnotation string = "-- +goose Up"
const gooseMigrationDownAnnotation string = "-- +goose Down"

func (w *GooseCompatibleSqlWriter) WriteSqlFile(outputPath string, migration Migration) error {
	file := filepath.Join(outputPath, fmt.Sprintf("%s.sql", migration.Version))

	sql := strings.Join(
		[]string{
			gooseMigrationUpAnnotation,
			migration.UpSQL,
			gooseMigrationDownAnnotation,
			migration.DownSQL,
		},
		"\n",
	)

	sql += "\n" // Adding endline at the end of the file as a good practice

	if err := os.WriteFile(file, []byte(sql), 0644); err != nil {
		return fmt.Errorf("error writing migration file: %w", err)
	}

	return nil
}
