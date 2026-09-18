package limen

import (
	"context"
	"fmt"
	"slices"
	"strings"
)

func ValidateClientIDValue(core *LimenCore, schema Schema, value any) error {
	id, ok := value.(string)
	if !ok {
		return fmt.Errorf("must be a string")
	}

	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("must be a non-empty string")
	}

	schemaName, config, enabled := core.getPublicIDConfig(schema)
	if enabled && !config.Matcher(schemaName, id) {
		return ErrRecordNotFound
	}

	return nil
}

func (core *LimenCore) assignPublicID(ctx context.Context, schema Schema, payload map[string]any) error {
	_, config, enabled := core.getPublicIDConfig(schema)
	if !enabled {
		return nil
	}

	field := core.PublicIDColumn(schema)
	if field == "" {
		return fmt.Errorf("failed to resolve public-ID field for schema %q", schema.GetTableName())
	}

	if _, exists := payload[field]; exists {
		return nil
	}

	if config.Generator == nil {
		return nil
	}

	value, err := core.GeneratePublicID(ctx, schema)
	if err != nil {
		return err
	}

	payload[field] = value
	return nil
}

func (core *LimenCore) rewritePublicIDConditions(schema Schema, conditions []Where) ([]Where, error) {
	schemaName, config, enabled := core.getPublicIDConfig(schema)
	if !enabled || len(conditions) == 0 {
		return conditions, nil
	}

	idColumn := schema.GetIDField()
	publicIDColumn := schema.GetField(config.field)

	var err error
	for i, condition := range conditions {
		if condition.Column != idColumn {
			continue
		}

		conditions[i], err = rewritePublicIDCondition(schemaName, config, publicIDColumn, condition)
		if err != nil {
			return nil, err
		}
	}
	return conditions, nil
}

func rewritePublicIDCondition(schemaName SchemaName, config *PublicIDConfig, publicIDColumn string, condition Where) (Where, error) {
	if condition.Operator == OpIn || condition.Operator == OpNotIn {
		return rewritePublicIDSliceCondition(schemaName, config, publicIDColumn, condition)
	}

	value, ok := condition.Value.(string)
	if !ok || !config.Matcher(schemaName, value) || !slices.Contains([]Operator{OpEq, OpNe}, condition.Operator) {
		return condition, nil
	}

	decoded, err := config.Decoder(schemaName, value)
	if err != nil {
		return Where{}, err
	}
	condition.Column = publicIDColumn
	condition.Value = decoded
	return condition, nil
}

func rewritePublicIDSliceCondition(schemaName SchemaName, config *PublicIDConfig, publicIDColumn string, condition Where) (Where, error) {
	values, ok := publicIDStrings(condition.Value)
	if !ok || !config.Matcher(schemaName, values[0]) {
		return condition, nil
	}

	decoded := make([]any, 0, len(values))
	for _, raw := range values {
		decodedValue, err := config.Decoder(schemaName, raw)
		if err != nil {
			return Where{}, err
		}
		decoded = append(decoded, decodedValue)
	}

	condition.Column = publicIDColumn
	condition.Value = decoded
	return condition, nil
}

func publicIDStrings(value any) ([]string, bool) {
	switch v := value.(type) {
	case []string:
		if len(v) == 0 {
			return nil, false
		}
		return v, true
	case []any:
		if len(v) == 0 {
			return nil, false
		}
		out := make([]string, 0, len(v))
		for _, item := range v {
			s, ok := item.(string)
			if !ok {
				return nil, false
			}
			out = append(out, s)
		}
		return out, true
	default:
		return nil, false
	}
}
