package aegis

import "time"

type Verification struct {
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
	// name of the table in the database
	TableName TableName
	// mapping of the verification schema to the database columns
	Fields VerificationFields
	// A function to return a map of additional fields to be added to the schema when creating a record
	AdditionalFields AdditionalFieldsFunc
}

type VerificationFields struct {
	ID              string
	Subject         string // ${action}:${identifier} (e.g. email_verification:john.doe@example.com)
	Value           string // token/code
	ExpiresAt       string
	CreatedAt       string
	UpdatedAt       string
	SoftDeleteField string
}

func (v *VerificationSchema) GetTableName() TableName {
	if v.TableName == "" {
		return VerificationSchemaTableName
	}
	return v.TableName
}

func (v *VerificationSchema) GetAdditionalFields() AdditionalFieldsFunc {
	return v.AdditionalFields
}

func (v *VerificationSchema) GetIDField() string {
	return getFieldOrDefault(v.Fields.ID, SchemaIDField)
}

func (v *VerificationSchema) GetSubjectField() string {
	return getFieldOrDefault(v.Fields.Subject, VerificationSchemaSubjectField)
}

func (v *VerificationSchema) GetValueField() string {
	return getFieldOrDefault(v.Fields.Value, VerificationSchemaValueField)
}

func (v *VerificationSchema) GetExpiresAtField() string {
	return getFieldOrDefault(v.Fields.ExpiresAt, VerificationSchemaExpiresAtField)
}

func (v *VerificationSchema) GetCreatedAtField() string {
	return getFieldOrDefault(v.Fields.CreatedAt, VerificationSchemaCreatedAtField)
}

func (v *VerificationSchema) GetUpdatedAtField() string {
	return getFieldOrDefault(v.Fields.UpdatedAt, VerificationSchemaUpdatedAtField)
}

func (v *VerificationSchema) GetSoftDeleteField() string {
	return getFieldOrDefault(v.Fields.SoftDeleteField, "")
}

func (v *VerificationSchema) FromStorage(data map[string]any) *Verification {
	return &Verification{
		Subject:   data[v.GetSubjectField()].(string),
		Value:     data[v.GetValueField()].(string),
		ExpiresAt: data[v.GetExpiresAtField()].(time.Time),
		CreatedAt: data[v.GetCreatedAtField()].(time.Time),
		UpdatedAt: data[v.GetUpdatedAtField()].(time.Time),
		raw:       data,
	}
}

func (v *VerificationSchema) ToStorage(data *Verification) map[string]any {
	return map[string]any{
		v.GetSubjectField():   data.Subject,
		v.GetValueField():     data.Value,
		v.GetExpiresAtField(): data.ExpiresAt,
		v.GetCreatedAtField(): data.CreatedAt,
		v.GetUpdatedAtField(): data.UpdatedAt,
	}
}
