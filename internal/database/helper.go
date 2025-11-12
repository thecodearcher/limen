package database

import (
	"context"
	"fmt"
	"maps"
	"strings"
	"time"

	"github.com/thecodearcher/aegis"
)

func FindOne[T aegis.Model](ctx context.Context, core *aegis.AegisCore, schema aegis.Schema[T], conditions []aegis.Where, orderBy []aegis.OrderBy) (*T, error) {
	conditions = applySoftDeleteFilter(ctx, core, schema, conditions)
	result, err := core.DB.FindOne(ctx, schema.GetTableName(), conditions, orderBy)
	if err != nil {
		return nil, err
	}

	model := schema.FromStorage(result)
	return model, nil
}

func Create[T aegis.Model](ctx context.Context, core *aegis.AegisCore, schema aegis.Schema[T], data *T, additionalFields map[string]any) error {
	payload := make(map[string]any)
	additionalFieldsContext := aegis.NewAdditionalFieldsContext(nil, nil)
	// the order of the copy of the fields is important here!
	// global -> schema -> additional fields -> data
	if core.Schema.AdditionalFields != nil {
		additionalFields, err := core.Schema.AdditionalFields(additionalFieldsContext)
		if err != nil {
			return err
		}
		maps.Copy(payload, additionalFields)
	}

	if schema.GetAdditionalFields() != nil {
		additionalFields, err := schema.GetAdditionalFields()(additionalFieldsContext)
		if err != nil {
			return err
		}
		maps.Copy(payload, additionalFields)
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

func Exists[T aegis.Model](ctx context.Context, core *aegis.AegisCore, schema aegis.Schema[T], conditions []aegis.Where) (bool, error) {
	conditions = applySoftDeleteFilter(ctx, core, schema, conditions)

	return core.DB.Exists(ctx, schema.GetTableName(), conditions)
}

func GenerateVerificationAction(action string, identifier string) string {
	return fmt.Sprintf("%s::%s", action, identifier)
}

func ParseVerificationAction(action string) (string, string) {
	parts := strings.Split(action, "::")
	return parts[0], parts[1]
}

func Update[T aegis.Model](ctx context.Context, core *aegis.AegisCore, schema aegis.Schema[T], data *T, conditions []aegis.Where) error {
	payload := make(map[string]any)
	maps.Copy(payload, schema.ToStorage(data))

	for key, value := range payload {
		//we remove any empty strings or zeros to avoid accidental NULL updates
		if value == "" || value == 0 || value == nil {
			delete(payload, key)
		}
	}
	conditions = applySoftDeleteFilter(ctx, core, schema, conditions)

	return core.DB.Update(ctx, schema.GetTableName(), conditions, payload)
}

func applySoftDeleteFilter[T aegis.Model](ctx context.Context, core *aegis.AegisCore, schema aegis.Schema[T], conditions []aegis.Where) []aegis.Where {
	softDeleteField := core.Schema.SoftDeleteField
	if schema.GetSoftDeleteField() != "" {
		softDeleteField = string(schema.GetSoftDeleteField())
	}

	if softDeleteField != "" {
		conditions = append(conditions, aegis.IsNull(softDeleteField))
	}

	return conditions
}

func Delete[T aegis.Model](ctx context.Context, core *aegis.AegisCore, schema aegis.Schema[T], conditions []aegis.Where) error {
	// if there are conditions, we update the soft delete field to the current time
	// otherwise we delete the record directly
	if schema.GetSoftDeleteField() != "" {
		if err := core.DB.Update(ctx, schema.GetTableName(), conditions, map[string]any{
			string(schema.GetSoftDeleteField()): time.Now().UTC(),
		}); err != nil {
			return err
		}

		return nil
	}

	return core.DB.Delete(ctx, schema.GetTableName(), conditions)
}
