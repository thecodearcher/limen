package main

import (
	"database/sql"
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
	return aegis.IndexDefinition{
		Name:    idxName,
		Columns: columns,
		Unique:  isUnique,
	}, nil
}

func (d *baseDriver) ParseForeignKeyRow(scan func(dest ...any) error) (aegis.ForeignKeyDefinition, error) {
	var fkName, colName, refTable, refCol, deleteRule, updateRule string

	if err := scan(&fkName, &colName, &refTable, &refCol, &deleteRule, &updateRule); err != nil {
		return aegis.ForeignKeyDefinition{}, err
	}

	return aegis.ForeignKeyDefinition{
		Name:             fkName,
		Column:           colName,
		ReferencedSchema: aegis.SchemaTableName(refTable),
		ReferencedField:  aegis.SchemaField(refCol),
		OnDelete:         aegis.ForeignKeyAction(deleteRule),
		OnUpdate:         aegis.ForeignKeyAction(updateRule),
	}, nil
}

func (d *baseDriver) DropColumnSQL(tableName, columnName string) string {
	return fmt.Sprintf("DROP COLUMN %s", columnName)
}

// normalizeDefault normalizes default values from the database
// Handles both PostgreSQL (nextval) and MySQL (CURRENT_TIMESTAMP) cases
func (d *baseDriver) normalizeDefault(defaultValue string) string {
	if defaultValue == "" {
		return ""
	}

	// Remove quotes from string defaults
	if len(defaultValue) > 0 && (defaultValue[0] == '\'' || defaultValue[0] == '"') {
		defaultValue = defaultValue[1 : len(defaultValue)-1]
	}

	return defaultValue
}

func (d *baseDriver) parseColumnRowCommon(
	colName, isNullable string,
	colDefault sql.NullString,
	isPK bool,
) (aegis.ColumnDefinition, error) {
	defaultValue := d.normalizeDefault(colDefault.String)

	return aegis.ColumnDefinition{
		Name:         colName,
		LogicalField: colName,
		IsNullable:   isNullable == "YES",
		IsPrimaryKey: isPK,
		DefaultValue: defaultValue,
	}, nil
}

func checkForSpecialSyntaxPatterns(defaultValue string) bool {
	return strings.HasPrefix(defaultValue, aegis.DatabaseDefaultValuePrefix)
}
