package aegis

import "time"

type Verification struct {
	ID        any
	Subject   string
	Value     string
	ExpiresAt time.Time
	CreatedAt time.Time
	UpdatedAt time.Time
	raw       map[string]any
}

func (v Verification) Raw() map[string]any {
	return v.raw
}

type VerificationSchema struct {
	BaseSchema
}

type SchemaConfigVerificationOption func(*SchemaConfig, *VerificationSchema)

func newDefaultVerificationSchema(c *SchemaConfig, opts ...SchemaConfigVerificationOption) *VerificationSchema {
	schema := &VerificationSchema{
		BaseSchema: BaseSchema{},
	}

	for _, opt := range opts {
		opt(c, schema)
	}

	return schema
}

func (v *VerificationSchema) GetSubjectField() string {
	return v.GetField(VerificationSchemaSubjectField)
}

func (v *VerificationSchema) GetValueField() string {
	return v.GetField(VerificationSchemaValueField)
}

func (v *VerificationSchema) GetExpiresAtField() string {
	return v.GetField(VerificationSchemaExpiresAtField)
}

func (v *VerificationSchema) GetCreatedAtField() string {
	return v.GetField(SchemaCreatedAtField)
}

func (v *VerificationSchema) GetUpdatedAtField() string {
	return v.GetField(SchemaUpdatedAtField)
}

func (v *VerificationSchema) FromStorage(data map[string]any) Model {
	return &Verification{
		ID:        data[v.GetIDField()],
		Subject:   data[v.GetSubjectField()].(string),
		Value:     data[v.GetValueField()].(string),
		ExpiresAt: data[v.GetExpiresAtField()].(time.Time),
		CreatedAt: data[v.GetCreatedAtField()].(time.Time),
		UpdatedAt: data[v.GetUpdatedAtField()].(time.Time),
		raw:       data,
	}
}

func (v *VerificationSchema) ToStorage(data Model) map[string]any {
	verification := data.(*Verification)
	return map[string]any{
		v.GetSubjectField():   verification.Subject,
		v.GetValueField():     verification.Value,
		v.GetExpiresAtField(): verification.ExpiresAt,
		v.GetCreatedAtField(): verification.CreatedAt,
		v.GetUpdatedAtField(): verification.UpdatedAt,
	}
}
