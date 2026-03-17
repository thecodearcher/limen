package main

import (
	"fmt"
	"strings"

	"github.com/thecodearcher/limen"
)

type baseDriver struct{}

func (d *baseDriver) ParseIndexRow(scan func(dest ...any) error) (limen.IndexDefinition, error) {
	var idxName string
	var columnsStr string
	var isUnique bool

	if err := scan(&idxName, &columnsStr, &isUnique); err != nil {
		return limen.IndexDefinition{}, err
	}

	columns := strings.Split(columnsStr, ",")
	columnsFields := make([]limen.SchemaField, len(columns))
	for i, col := range columns {
		columnsFields[i] = limen.SchemaField(col)
	}
	return limen.IndexDefinition{
		Name:    idxName,
		Columns: columnsFields,
		Unique:  isUnique,
	}, nil
}

func (d *baseDriver) ParseForeignKeyRow(scan func(dest ...any) error) (limen.ForeignKeyDefinition, error) {
	var fkName string

	if err := scan(&fkName); err != nil {
		return limen.ForeignKeyDefinition{}, err
	}

	return limen.ForeignKeyDefinition{
		Name: fkName,
	}, nil
}

func (d *baseDriver) DropColumnSQL(tableName, columnName string) string {
	return fmt.Sprintf("DROP COLUMN %s", columnName)
}

func (d *baseDriver) ParseColumnRow(scan func(dest ...any) error) (limen.ColumnDefinition, error) {
	var colName string

	if err := scan(&colName); err != nil {
		return limen.ColumnDefinition{}, err
	}

	return limen.ColumnDefinition{
		Name:         colName,
		LogicalField: limen.SchemaField(colName),
		Type:         limen.ColumnTypeString,
		IsNullable:   false,
		IsPrimaryKey: false,
		DefaultValue: "",
	}, nil
}

func checkForSpecialSyntaxPatterns(defaultValue string) bool {
	return strings.HasPrefix(defaultValue, limen.DatabaseDefaultValuePrefix)
}
