package database

import (
	"context"
	"fmt"
	"maps"
	"strings"

	"github.com/thecodearcher/aegis"
	"github.com/thecodearcher/aegis/schemas"
)

func FindOne[T schemas.Model](ctx context.Context, db aegis.DatabaseAdapter, schema schemas.Schema[T], conditions []aegis.Where, orderBy []aegis.OrderBy) (*T, error) {
	if schema.GetSoftDeleteField() != "" {
		conditions = append(conditions, aegis.IsNull(string(schema.GetSoftDeleteField())))
	}

	result, err := db.FindOne(ctx, schema.GetTableName(), conditions, orderBy)
	if err != nil {
		return nil, err
	}

	model := schema.FromStorage(result)
	return model, nil
}

func Create[T schemas.Model](ctx context.Context, core *aegis.AegisCore, schema schemas.Schema[T], data *T, additionalFields map[string]any) error {
	payload := make(map[string]any)
	// the order of the copy of the fields is important here!
	// global -> schema -> additional fields -> data
	if core.Schema.AdditionalFields != nil {
		maps.Copy(payload, core.Schema.AdditionalFields(ctx))
	}

	if schema.GetAdditionalFields() != nil {
		maps.Copy(payload, schema.GetAdditionalFields()(ctx))
	}
	maps.Copy(payload, additionalFields)
	maps.Copy(payload, schema.ToStorage(data))

	for key, value := range payload {
		// empty strings are converted to nil to avoid empty strings in the database
		if value == "" {
			payload[key] = nil
		}
	}

	err := core.DB.Create(ctx, schema.GetTableName(), payload)
	if err != nil {
		return err
	}

	return nil
}

func Exists[T schemas.Model](ctx context.Context, db aegis.DatabaseAdapter, schema schemas.Schema[T], conditions []aegis.Where) (bool, error) {
	if schema.GetSoftDeleteField() != "" {
		conditions = append(conditions, aegis.IsNull(string(schema.GetSoftDeleteField())))
	}

	return db.Exists(ctx, schema.GetTableName(), conditions)
}

func GenerateVerificationAction(action string, identifier string) string {
	return fmt.Sprintf("%s::%s", action, identifier)
}

func ParseVerificationAction(action string) (string, string) {
	parts := strings.Split(action, "::")
	return parts[0], parts[1]
}

func Update[T schemas.Model](ctx context.Context, db aegis.DatabaseAdapter, schema schemas.Schema[T], data *T, conditions []aegis.Where) error {
	payload := make(map[string]any)
	maps.Copy(payload, schema.ToStorage(data))

	for key, value := range payload {
		//we remove any empty strings or zeros to avoid accidental NULL updates
		if value == "" || value == 0 || value == nil {
			delete(payload, key)
		}
	}

	return db.Update(ctx, schema.GetTableName(), conditions, payload)
}
