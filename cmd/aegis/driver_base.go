package main

import (
	"fmt"
	"strings"

	"github.com/thecodearcher/aegis"
)

type baseDriver struct{}

func (d *baseDriver) ParseIndexRow(scan func(dest ...any) error) (aegis.IndexDefinition, error) {
	var idxName string
	var columnsStr string
	var isUnique bool

	if err := scan(&idxName, &columnsStr, &isUnique); err != nil {
		return aegis.IndexDefinition{}, err
	}

	columns := strings.Split(columnsStr, ",")
	columnsFields := make([]aegis.SchemaField, len(columns))
	for i, col := range columns {
		columnsFields[i] = aegis.SchemaField(col)
	}
	return aegis.IndexDefinition{
		Name:    idxName,
		Columns: columnsFields,
		Unique:  isUnique,
	}, nil
}

func (d *baseDriver) ParseForeignKeyRow(scan func(dest ...any) error) (aegis.ForeignKeyDefinition, error) {
	var fkName string

	if err := scan(&fkName); err != nil {
		return aegis.ForeignKeyDefinition{}, err
	}

	return aegis.ForeignKeyDefinition{
		Name: fkName,
	}, nil
}

func (d *baseDriver) DropColumnSQL(tableName, columnName string) string {
	return fmt.Sprintf("DROP COLUMN %s", columnName)
}

func (d *baseDriver) ParseColumnRow(scan func(dest ...any) error) (aegis.ColumnDefinition, error) {
	var colName string

	if err := scan(&colName); err != nil {
		return aegis.ColumnDefinition{}, err
	}

	return aegis.ColumnDefinition{
		Name:         colName,
		LogicalField: aegis.SchemaField(colName),
		Type:         aegis.ColumnTypeString,
		IsNullable:   false,
		IsPrimaryKey: false,
		DefaultValue: "",
	}, nil
}

func checkForSpecialSyntaxPatterns(defaultValue string) bool {
	return strings.HasPrefix(defaultValue, aegis.DatabaseDefaultValuePrefix)
}
