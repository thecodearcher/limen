package aegis

import "time"

type RateLimit struct {
	ID            any
	Key           string
	Count         int
	LastRequestAt int64
	raw           map[string]any
}

func (r *RateLimit) Touch() {
	r.LastRequestAt = time.Now().UnixMilli()
	r.Count = r.Count + 1
}

func (r *RateLimit) ResetCounter() {
	r.Count = 0
}

func (r RateLimit) Raw() map[string]any {
	return r.raw
}

type RateLimitSchema struct {
	// name of the table in the database
	TableName TableName

	// mapping of the rate limit schema to the database columns
	Fields RateLimitFields
}

type RateLimitFields struct {
	ID            string
	Key           string
	Count         string
	LastRequestAt string
}

func (r *RateLimitSchema) GetTableName() TableName {
	if r.TableName == "" {
		return RateLimitSchemaTableName
	}
	return r.TableName
}

func (r *RateLimitSchema) GetAdditionalFields() AdditionalFieldsFunc {
	return nil
}

func (r *RateLimitSchema) GetSoftDeleteField() string {
	return ""
}

func (r *RateLimitSchema) GetIDField() string {
	return getFieldOrDefault(r.Fields.ID, SchemaIDField)
}

func (r *RateLimitSchema) GetKeyField() string {
	return getFieldOrDefault(r.Fields.Key, RateLimitSchemaKeyField)
}

func (r *RateLimitSchema) GetCountField() string {
	return getFieldOrDefault(r.Fields.Count, RateLimitSchemaCountField)
}

func (r *RateLimitSchema) GetLastRequestAtField() string {
	return getFieldOrDefault(r.Fields.LastRequestAt, RateLimitSchemaLastRequestAtField)
}

func (r *RateLimitSchema) FromStorage(data map[string]any) *RateLimit {
	return &RateLimit{
		ID:            data[r.GetIDField()],
		Key:           data[r.GetKeyField()].(string),
		Count:         int(data[r.GetCountField()].(int32)),
		LastRequestAt: data[r.GetLastRequestAtField()].(int64),
		raw:           data,
	}
}

func (r *RateLimitSchema) ToStorage(data *RateLimit) map[string]any {
	return map[string]any{
		r.GetKeyField():           data.Key,
		r.GetCountField():         data.Count,
		r.GetLastRequestAtField(): data.LastRequestAt,
	}
}
