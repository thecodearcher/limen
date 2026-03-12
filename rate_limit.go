package aegis

import "time"

type RateLimit struct {
	ID            any    `json:"id,omitempty"`
	Key           string `json:"key"`
	Count         int    `json:"count"`
	LastRequestAt int64  `json:"last_request_at"`
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
	BaseSchema
}

type SchemaConfigRateLimitOption func(*SchemaConfig, *RateLimitSchema)

func newDefaultRateLimitSchema(c *SchemaConfig, opts ...SchemaConfigRateLimitOption) *RateLimitSchema {
	schema := &RateLimitSchema{
		BaseSchema: BaseSchema{},
	}

	for _, opt := range opts {
		opt(c, schema)
	}

	return schema
}

func (r *RateLimitSchema) GetIDField() string {
	return r.GetField(SchemaIDField)
}

func (r *RateLimitSchema) GetKeyField() string {
	return r.GetField(RateLimitSchemaKeyField)
}

func (r *RateLimitSchema) GetCountField() string {
	return r.GetField(RateLimitSchemaCountField)
}

func (r *RateLimitSchema) GetLastRequestAtField() string {
	return r.GetField(RateLimitSchemaLastRequestAtField)
}

func (r *RateLimitSchema) FromStorage(data map[string]any) Model {
	return &RateLimit{
		ID:            data[r.GetIDField()],
		Key:           data[r.GetKeyField()].(string),
		Count:         int(data[r.GetCountField()].(int32)),
		LastRequestAt: data[r.GetLastRequestAtField()].(int64),
		raw:           data,
	}
}

func (r *RateLimitSchema) ToStorage(data Model) map[string]any {
	rateLimit := data.(*RateLimit)
	return map[string]any{
		r.GetKeyField():           rateLimit.Key,
		r.GetCountField():         rateLimit.Count,
		r.GetLastRequestAtField(): rateLimit.LastRequestAt,
	}
}
