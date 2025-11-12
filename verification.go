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

func (v Verification) TableName() string {
	return string(VerificationSchemaTableName)
}

func (v Verification) Raw() map[string]any {
	return v.raw
}

type VerificationSchema struct {
	// name of the table in the database
	TableName TableName
	Fields    VerificationFields
}

type VerificationFields struct {
	ID        string
	Subject   string // (email_verification, password_reset, etc.):${email,id}
	Value     string // token/code
	ExpiresAt string
	CreatedAt string
	UpdatedAt string
}

func (c *VerificationSchema) GetTableName() TableName {
	if c.TableName == "" {
		return VerificationSchemaTableName
	}
	return c.TableName
}

func (c *VerificationSchema) GetAdditionalFields() AdditionalFieldsFunc {
	return nil
}

func (c *VerificationSchema) GetIDField() string {
	return getFieldOrDefault(c.Fields.ID, SchemaIDField)
}

func (c *VerificationSchema) GetSubjectField() string {
	return getFieldOrDefault(c.Fields.Subject, VerificationSchemaSubjectField)
}

func (c *VerificationSchema) GetValueField() string {
	return getFieldOrDefault(c.Fields.Value, VerificationSchemaValueField)
}

func (c *VerificationSchema) GetExpiresAtField() string {
	return getFieldOrDefault(c.Fields.ExpiresAt, VerificationSchemaExpiresAtField)
}

func (c *VerificationSchema) GetCreatedAtField() string {
	return getFieldOrDefault(c.Fields.CreatedAt, VerificationSchemaCreatedAtField)
}

func (c *VerificationSchema) GetUpdatedAtField() string {
	return getFieldOrDefault(c.Fields.UpdatedAt, VerificationSchemaUpdatedAtField)
}

func (c *VerificationSchema) GetSoftDeleteField() SchemaField {
	return ""
}

func (c *VerificationSchema) FromStorage(data map[string]any) *Verification {
	return &Verification{
		Subject:   data[c.GetSubjectField()].(string),
		Value:     data[c.GetValueField()].(string),
		ExpiresAt: data[c.GetExpiresAtField()].(time.Time),
		CreatedAt: data[c.GetCreatedAtField()].(time.Time),
		UpdatedAt: data[c.GetUpdatedAtField()].(time.Time),
		raw:       data,
	}
}

func (c *VerificationSchema) ToStorage(data *Verification) map[string]any {
	return map[string]any{
		c.GetSubjectField():   data.Subject,
		c.GetValueField():     data.Value,
		c.GetExpiresAtField(): data.ExpiresAt,
		c.GetCreatedAtField(): data.CreatedAt,
		c.GetUpdatedAtField(): data.UpdatedAt,
	}
}
