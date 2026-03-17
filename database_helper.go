package limen

import (
	"context"
	"fmt"
	"maps"
	reflect "reflect"
	"strings"
	"time"
)

// getDB returns the database adapter to use, checking in this order:
//
//  1. Transaction from context (if in a transaction )
//  2. Default database adapter
//
// if skipTx is true, the default database adapter is returned.
func (core *LimenCore) getDB(ctx context.Context, skipTx ...bool) DatabaseAdapter {
	if len(skipTx) > 0 && skipTx[0] {
		return core.db
	}
	if tx := getTxFromContext(ctx); tx != nil {
		return tx
	}
	return core.db
}

func (core *LimenCore) FindOne(ctx context.Context, schema Schema, conditions []Where, orderBy []OrderBy) (Model, error) {
	conditions = applySoftDeleteFilter(schema, conditions)
	db := core.getDB(ctx)
	result, err := db.FindOne(ctx, schema.GetTableName(), conditions, orderBy)
	if err != nil {
		return nil, err
	}

	model := schema.FromStorage(result)
	return model, nil
}

func (core *LimenCore) Create(ctx context.Context, schema Schema, data Model, additionalFields map[string]any) error {
	payload := make(map[string]any)

	additionalFieldsContext := getAdditionalFieldsContext(ctx)

	// the order of the copy of the fields is important here!
	// global additional fields -> schema additional fields -> directly passed additional fields -> data
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

	if err := core.assignID(ctx, schema, payload); err != nil {
		return err
	}

	db := core.getDB(ctx)
	err := db.Create(ctx, schema.GetTableName(), payload)
	if err != nil {
		return err
	}

	return nil
}

func (core *LimenCore) Exists(ctx context.Context, schema Schema, conditions []Where) (bool, error) {
	conditions = applySoftDeleteFilter(schema, conditions)
	db := core.getDB(ctx)
	return db.Exists(ctx, schema.GetTableName(), conditions)
}

func GenerateVerificationAction(action string, identifier string) string {
	return fmt.Sprintf("%s::%s", action, identifier)
}

func ParseVerificationAction(action string) (string, string) {
	parts := strings.Split(action, "::")
	return parts[0], parts[1]
}

func (core *LimenCore) Update(ctx context.Context, schema Schema, updatedData Model, conditions []Where) error {
	return core.UpdateRaw(ctx, schema, updatedData, conditions, true)
}

func (core *LimenCore) UpdateRaw(ctx context.Context, schema Schema, updatedData Model, conditions []Where, removeEmptyValues bool) error {
	payload := make(map[string]any)

	maps.Copy(payload, schema.ToStorage(updatedData))
	if removeEmptyValues {
		for key, value := range payload {
			concreteValue := reflect.ValueOf(value)
			//we remove any empty strings or zeros to avoid accidental NULL updates
			if !concreteValue.IsValid() || concreteValue.IsZero() {
				delete(payload, key)
			}
		}
	}

	conditions = applySoftDeleteFilter(schema, conditions)
	db := core.getDB(ctx)

	return db.Update(ctx, schema.GetTableName(), conditions, payload)
}

func (core *LimenCore) assignID(ctx context.Context, schema Schema, payload map[string]any) error {
	idField := schema.GetIDField()
	if idField == "" {
		return nil
	}

	if _, exists := payload[idField]; exists || core.Schema.IDGenerator == nil {
		return nil
	}

	id, err := core.Schema.IDGenerator.Generate(ctx)
	if err != nil {
		return fmt.Errorf("failed to generate ID: %w", err)
	}

	if id != nil {
		payload[idField] = id
	}

	return nil
}

func applySoftDeleteFilter(schema Schema, conditions []Where) []Where {
	softDeleteField := schema.GetSoftDeleteField()
	if softDeleteField != "" {
		conditions = append(conditions, IsNull(softDeleteField))
	}
	return conditions
}

func (core *LimenCore) Delete(ctx context.Context, schema Schema, conditions []Where) error {
	db := core.getDB(ctx)
	// if there are conditions, we update the soft delete field to the current time
	// otherwise we delete the record directly
	if schema.GetSoftDeleteField() != "" {
		if err := db.Update(ctx, schema.GetTableName(), conditions, map[string]any{
			string(schema.GetSoftDeleteField()): time.Now(),
		}); err != nil {
			return err
		}

		return nil
	}

	return db.Delete(ctx, schema.GetTableName(), conditions)
}

func (core *LimenCore) FindMany(ctx context.Context, schema Schema, conditions []Where) ([]Model, error) {
	db := core.getDB(ctx)
	list, err := db.FindMany(ctx, schema.GetTableName(), conditions, nil)
	if err != nil {
		return nil, err
	}
	out := make([]Model, 0, len(list))
	for _, m := range list {
		out = append(out, schema.FromStorage(m))
	}
	return out, nil
}

func (core *LimenCore) Count(ctx context.Context, schema Schema, conditions []Where) (int64, error) {
	db := core.getDB(ctx)
	return db.Count(ctx, schema.GetTableName(), conditions)
}
