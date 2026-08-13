package limen

import (
	"context"
	"fmt"
	"maps"
	reflect "reflect"
	"strings"
	"time"
)

type UpdateOption func(*updateOptions)

func WithUpdateAdditionalFields(fields map[string]any) UpdateOption {
	return func(options *updateOptions) {
		options.additionalFields = fields
	}
}

func newUpdateOptions(opts []UpdateOption) *updateOptions {
	options := &updateOptions{}
	for _, opt := range opts {
		opt(options)
	}
	return options
}

type updateOptions struct {
	additionalFields map[string]any
}

// getDB returns the database adapter to use, checking in this order:
//
//  1. Transaction from context (if in a transaction )
//  2. Default database adapter
func (core *LimenCore) getDB(ctx context.Context) DatabaseAdapter {
	if tx := getTxFromContext(ctx); tx != nil {
		return tx
	}
	return core.db
}

func (core *LimenCore) FindOne(ctx context.Context, schema Schema, conditions []Where, orderBy []OrderBy) (Model, error) {
	conditions, err := core.rewritePublicIDConditions(schema, conditions)
	if err != nil {
		return nil, err
	}
	conditions = applySoftDeleteFilter(schema, conditions)
	db := core.getDB(ctx)
	result, err := db.FindOne(ctx, schema.GetTableName(), conditions, orderBy)
	if err != nil {
		return nil, err
	}

	model := schema.FromStorage(result)
	return model, nil
}

// resolveAdditionalFields runs the global and schema AdditionalFieldsFunc for the write,
// with the schema fields taking precedence over the global ones.
func (core *LimenCore) resolveAdditionalFields(ctx context.Context, schema Schema, operation Operation) (map[string]any, error) {
	additionalFieldsContext := getAdditionalFieldsContext(ctx).forWrite(schema, operation)

	fields := make(map[string]any)
	if core.Schema.AdditionalFields != nil {
		globalFields, err := core.Schema.AdditionalFields(additionalFieldsContext)
		if err != nil {
			return nil, err
		}
		maps.Copy(fields, globalFields)
	}

	if schema.GetAdditionalFields() != nil {
		schemaFields, err := schema.GetAdditionalFields()(additionalFieldsContext)
		if err != nil {
			return nil, err
		}
		maps.Copy(fields, schemaFields)
	}

	return fields, nil
}

func (core *LimenCore) buildCreatePayload(ctx context.Context, schema Schema, data any, additionalFields map[string]any) (map[string]any, error) {
	// the order of the copy of the fields is important here!
	// global additional fields -> schema additional fields -> directly passed additional fields -> data
	payload, err := core.resolveAdditionalFields(ctx, schema, OperationCreate)
	if err != nil {
		return nil, err
	}
	maps.Copy(payload, additionalFields)

	payloadData, err := buildWritePayload(schema, data, true)
	if err != nil {
		return nil, err
	}
	maps.Copy(payload, payloadData)

	if err := core.assignID(ctx, schema, payload); err != nil {
		return nil, err
	}
	if err := core.assignPublicID(ctx, schema, payload); err != nil {
		return nil, err
	}

	applyUpdatedAtTimestamp(schema, payload, true)

	return payload, nil
}

func (core *LimenCore) Create(ctx context.Context, schema Schema, data any, additionalFields map[string]any) error {
	_, err := core.CreateWithResult(ctx, schema, data, additionalFields)
	return err
}

func (core *LimenCore) CreateWithResult(ctx context.Context, schema Schema, data any, additionalFields map[string]any) (*DatabaseResult, error) {
	payload, err := core.buildCreatePayload(ctx, schema, data, additionalFields)
	if err != nil {
		return nil, err
	}

	db := core.getDB(ctx)
	result, err := db.Create(ctx, schema.GetTableName(), payload)
	return &result, err
}

// CreateAndReturn inserts data and returns the stored row, located by the unique lookupColumns set on the model.
func (core *LimenCore) CreateAndReturn(ctx context.Context, schema Schema, data any, additionalFields map[string]any, lookupColumns ...SchemaField) (Model, error) {
	if len(lookupColumns) == 0 {
		return nil, fmt.Errorf("CreateAndReturn: at least one lookup column is required")
	}

	payload, err := core.buildCreatePayload(ctx, schema, data, additionalFields)
	if err != nil {
		return nil, err
	}

	db := core.getDB(ctx)
	if _, err := db.Create(ctx, schema.GetTableName(), payload); err != nil {
		return nil, err
	}

	conditions := make([]Where, 0, len(lookupColumns))
	for _, field := range lookupColumns {
		col := schema.GetField(field)
		conditions = append(conditions, Eq(col, payload[col]))
	}

	return core.FindOne(ctx, schema, conditions, nil)
}

func (core *LimenCore) Exists(ctx context.Context, schema Schema, conditions []Where) (bool, error) {
	conditions, err := core.rewritePublicIDConditions(schema, conditions)
	if err != nil {
		return false, err
	}
	conditions = applySoftDeleteFilter(schema, conditions)
	db := core.getDB(ctx)
	return db.Exists(ctx, schema.GetTableName(), conditions)
}

func GenerateVerificationAction(action, identifier string) string {
	return fmt.Sprintf("%s::%s", action, identifier)
}

func ParseVerificationAction(action string) (string, string) {
	parts := strings.Split(action, "::")
	return parts[0], parts[1]
}

// Update changes matching rows. Use Model for patch-style updates,
// or map[SchemaField]any when you need exact column control.
func (core *LimenCore) Update(ctx context.Context, schema Schema, data any, conditions []Where, opts ...UpdateOption) error {
	_, err := core.UpdateWithResult(ctx, schema, data, conditions, opts...)
	return err
}

func (core *LimenCore) UpdateWithResult(ctx context.Context, schema Schema, data any, conditions []Where, opts ...UpdateOption) (*DatabaseResult, error) {
	if len(conditions) == 0 {
		return nil, fmt.Errorf("%w: conditions required to prevent accidental table-wide update", ErrMissingConditions)
	}

	payload, err := core.buildUpdatePayload(ctx, schema, data, newUpdateOptions(opts))
	if err != nil {
		return nil, err
	}
	if len(payload) == 0 {
		return nil, nil
	}
	if err := validateUpdatePayload(payload); err != nil {
		return nil, err
	}

	conditions, err = core.rewritePublicIDConditions(schema, conditions)
	if err != nil {
		return nil, err
	}
	applyUpdatedAtTimestamp(schema, payload, false)

	conditions = applySoftDeleteFilter(schema, conditions)
	db := core.getDB(ctx)

	result, err := db.Update(ctx, schema.GetTableName(), conditions, payload)
	return &result, err
}

func (core *LimenCore) buildUpdatePayload(ctx context.Context, schema Schema, data any, options *updateOptions) (map[string]any, error) {
	// the order of the copy of the fields is important here!
	// global additional fields -> schema additional fields -> directly passed additional fields -> data
	payload, err := core.resolveAdditionalFields(ctx, schema, OperationUpdate)
	if err != nil {
		return nil, err
	}
	maps.Copy(payload, options.additionalFields)
	core.dropImmutableFields(schema, payload)

	payloadData, err := buildWritePayload(schema, data, false)
	if err != nil {
		return nil, err
	}
	maps.Copy(payload, payloadData)

	return payload, nil
}

func (core *LimenCore) dropImmutableFields(schema Schema, fields map[string]any) {
	if len(fields) == 0 {
		return
	}

	immutable := []string{
		schema.GetIDField(),
		schema.GetField(SchemaCreatedAtField),
	}
	if _, config, enabled := core.getPublicIDConfig(schema); enabled {
		immutable = append(immutable, schema.GetField(config.field))
	}

	for _, column := range immutable {
		delete(fields, column)
	}
}

func (core *LimenCore) UpdateAndReturn(ctx context.Context, schema Schema, data any, conditions []Where, id any, opts ...UpdateOption) (Model, error) {
	if err := core.Update(ctx, schema, data, conditions, opts...); err != nil {
		return nil, err
	}

	return core.FindOne(ctx, schema, []Where{Eq(schema.GetIDField(), id)}, nil)
}

func buildWritePayload(schema Schema, data any, forInsert bool) (map[string]any, error) {
	switch v := data.(type) {
	case map[SchemaField]any:
		payload := make(map[string]any, len(v))
		for field, value := range v {
			col := schema.GetField(field)
			if col == "" {
				return nil, fmt.Errorf("Update: unknown field %q for schema %q", field, schema.GetTableName())
			}
			payload[col] = value
		}
		return payload, nil
	case Model:
		payload := make(map[string]any)
		maps.Copy(payload, schema.ToStorage(v))
		if !forInsert {
			for key, value := range payload {
				concreteValue := reflect.ValueOf(value)
				// drop empty strings and zeros to avoid clobbering unset columns
				if !concreteValue.IsValid() || concreteValue.IsZero() {
					delete(payload, key)
				}
			}
		}
		return payload, nil
	default:
		return nil, fmt.Errorf("unsupported data type %T, want limen.Model or map[limen.SchemaField]any", data)
	}
}

func validateUpdatePayload(payload map[string]any) error {
	for column, value := range payload {
		update, ok := value.(ArithmeticUpdate)
		if !ok {
			continue
		}
		if err := update.Validate(); err != nil {
			return fmt.Errorf("Update: invalid arithmetic update for column %q: %w", column, err)
		}
	}
	return nil
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

// applyUpdatedAtTimestamp sets the schema's updated_at column when the resolver exposes it.
// For inserts (forInsert true), the timestamp is added only if the key is absent.
// For updates (forInsert false), it is always set to time.Now().
func applyUpdatedAtTimestamp(schema Schema, payload map[string]any, forInsert bool) {
	col := schema.GetField(SchemaUpdatedAtField)
	if col == "" {
		return
	}
	now := time.Now()
	if forInsert {
		if _, exists := payload[col]; !exists {
			payload[col] = now
		}
		return
	}
	payload[col] = now
}

func applySoftDeleteFilter(schema Schema, conditions []Where) []Where {
	softDeleteField := schema.GetSoftDeleteField()
	if softDeleteField != "" {
		conditions = append(conditions, IsNull(softDeleteField))
	}
	return conditions
}

func (core *LimenCore) Delete(ctx context.Context, schema Schema, conditions []Where) error {
	_, err := core.DeleteWithResult(ctx, schema, conditions)
	return err
}

func (core *LimenCore) DeleteWithResult(ctx context.Context, schema Schema, conditions []Where) (*DatabaseResult, error) {
	if len(conditions) == 0 {
		return nil, fmt.Errorf("%w: conditions required to prevent accidental table-wide delete", ErrMissingConditions)
	}

	conditions, err := core.rewritePublicIDConditions(schema, conditions)
	if err != nil {
		return nil, err
	}

	db := core.getDB(ctx)
	// if there are conditions, we update the soft delete field to the current time
	// otherwise we delete the record directly
	if schema.GetSoftDeleteField() != "" {
		result, err := db.Update(ctx, schema.GetTableName(), conditions, map[string]any{
			schema.GetSoftDeleteField(): time.Now(),
		})
		return &result, err
	}

	result, err := db.Delete(ctx, schema.GetTableName(), conditions)
	return &result, err
}

func (core *LimenCore) FindMany(ctx context.Context, schema Schema, conditions []Where) ([]Model, error) {
	db := core.getDB(ctx)
	conditions, err := core.rewritePublicIDConditions(schema, conditions)
	if err != nil {
		return nil, err
	}
	conditions = applySoftDeleteFilter(schema, conditions)

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
	conditions, err := core.rewritePublicIDConditions(schema, conditions)
	if err != nil {
		return 0, err
	}
	db := core.getDB(ctx)
	return db.Count(ctx, schema.GetTableName(), conditions)
}

const (
	DefaultPage    = 1
	DefaultPerPage = 20
	MaxPerPage     = 100
)

type Page[T any] struct {
	Items      []T   `json:"items"`
	Total      int64 `json:"total"`
	Page       int   `json:"page"`
	PerPage    int   `json:"per_page"`
	TotalPages int   `json:"total_pages"`
}

func (core *LimenCore) FindWithOptions(ctx context.Context, schema Schema, conditions []Where, opts *QueryOptions) (*Page[Model], error) {
	if opts == nil {
		opts = &QueryOptions{Limit: DefaultPerPage}
	}

	conditions, err := core.rewritePublicIDConditions(schema, conditions)
	if err != nil {
		return nil, err
	}
	conditions = applySoftDeleteFilter(schema, conditions)
	db := core.getDB(ctx)

	total, err := db.Count(ctx, schema.GetTableName(), conditions)
	if err != nil {
		return nil, err
	}

	if opts.OrderBy == nil {
		field := schema.GetField(SchemaCreatedAtField)
		if field == "" {
			return nil, fmt.Errorf("no order by column found, use QueryOptions.OrderBy to specify one")
		}
		opts.OrderBy = []OrderBy{{Column: field, Direction: OrderByDesc}}
	}

	list, err := db.FindMany(ctx, schema.GetTableName(), conditions, opts)
	if err != nil {
		return nil, err
	}

	out := make([]Model, 0, len(list))
	for _, m := range list {
		out = append(out, schema.FromStorage(m))
	}

	return newPage(out, total, opts), nil
}

func newPage[T any](items []T, total int64, opts *QueryOptions) *Page[T] {
	perPage := opts.Limit
	if perPage <= 0 {
		perPage = DefaultPerPage
	}

	return &Page[T]{
		Items:      items,
		Total:      total,
		Page:       opts.Offset/perPage + 1,
		PerPage:    perPage,
		TotalPages: int((total + int64(perPage) - 1) / int64(perPage)),
	}
}
