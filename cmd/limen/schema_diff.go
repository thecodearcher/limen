package main

import (
	"strings"

	"github.com/thecodearcher/limen"
)

// schemaDiff represents the differences between an existing schema and a required schema
type schemaDiff struct {
	AddedColumns     []limen.ColumnDefinition
	AddedIndexes     []limen.IndexDefinition
	AddedForeignKeys []limen.ForeignKeyDefinition
}

// compareSchemas compares an existing schema with a required schema and returns the differences
func compareSchemas(existing, schemas *limen.SchemaDefinition) schemaDiff {
	diff := schemaDiff{
		AddedColumns:     []limen.ColumnDefinition{},
		AddedIndexes:     []limen.IndexDefinition{},
		AddedForeignKeys: []limen.ForeignKeyDefinition{},
	}

	existingCols := make(map[string]limen.ColumnDefinition)
	for _, col := range existing.Columns {
		existingCols[col.Name] = col
	}

	requiredCols := make(map[string]limen.ColumnDefinition)
	for _, col := range schemas.Columns {
		requiredCols[col.Name] = col
	}

	for name, requiredCol := range requiredCols {
		_, exists := existingCols[name]
		if !exists {
			diff.AddedColumns = append(diff.AddedColumns, requiredCol)
		}
	}

	// Compare indexes
	existingIndexes := make(map[string]limen.IndexDefinition)
	for _, idx := range existing.Indexes {
		existingIndexes[indexKey(idx)] = idx
	}

	requiredIndexes := make(map[string]limen.IndexDefinition)
	for _, idx := range schemas.Indexes {
		requiredIndexes[indexKey(idx)] = idx
	}

	for key, requiredIdx := range requiredIndexes {
		if _, exists := existingIndexes[key]; !exists {
			diff.AddedIndexes = append(diff.AddedIndexes, requiredIdx)
		}
	}

	// Compare foreign keys
	existingFKs := make(map[string]limen.ForeignKeyDefinition)
	for _, fk := range existing.ForeignKeys {
		existingFKs[fk.Name] = fk
	}

	requiredFKs := make(map[string]limen.ForeignKeyDefinition)
	for _, fk := range schemas.ForeignKeys {
		requiredFKs[fk.Name] = fk
	}

	for name, requiredFK := range requiredFKs {
		if _, exists := existingFKs[name]; !exists {
			diff.AddedForeignKeys = append(diff.AddedForeignKeys, requiredFK)
		}
	}

	return diff
}

// indexKey creates a unique key for an index based on its columns and uniqueness
func indexKey(idx limen.IndexDefinition) string {
	var key strings.Builder
	if idx.Unique {
		key.WriteString("unique:")
	}
	for _, col := range idx.Columns {
		key.WriteString(string(col) + ",")
	}
	return key.String()
}

// HasChanges returns true if the diff contains any changes
func (d *schemaDiff) HasChanges() bool {
	return len(d.AddedColumns) > 0 ||
		len(d.AddedIndexes) > 0 ||
		len(d.AddedForeignKeys) > 0
}
