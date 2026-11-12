package admin

import (
	"github.com/thecodearcher/limen"
)

const SessionSchemaImpersonatedByField limen.SchemaField = "impersonated_by"

type sessionSchema struct {
	*limen.SessionSchema
}

func buildSessionImpersonationExtension(schemaConfig *limen.SchemaConfig) *limen.SchemaDefinition {
	return limen.NewSchemaDefinitionForExtension(
		limen.CoreSchemaSessions,
		&sessionSchema{SessionSchema: schemaConfig.Session},
		limen.WithSchemaField(SessionSchemaImpersonatedByField, schemaConfig.GetIDColumnType(), limen.WithNullable(true)),
		limen.WithSchemaIndex("idx_sessions_impersonated_by", []limen.SchemaField{SessionSchemaImpersonatedByField}),
	)
}
