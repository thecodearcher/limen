package sql

import (
	"strings"

	"github.com/thecodearcher/aegis"
)

// buildWhere returns a WHERE clause (without "WHERE") and args. Uses ? placeholders; caller must rebind before execute.
// Conditions are grouped by OR runs; groups are then AND'd. E.g. [A, B OR C, D] → (A) AND (B OR C) AND (D).
func (a *Adapter) buildWhere(conditions []aegis.Where) (string, []any) {
	if len(conditions) == 0 {
		return "", nil
	}

	if len(conditions) == 1 {
		return a.buildOneCondition(conditions[0])
	}

	groups := aegis.GroupConditionsByConnector(conditions)
	buildGroup := func(group []aegis.Where) (string, []any) {
		clause, args := aegis.BuildGroupClause(group, a.buildOneCondition)
		if clause != "" && len(group) > 1 {
			clause = "(" + clause + ")"
		}
		return clause, args
	}
	return aegis.BuildWhereFromGroups(groups, buildGroup)
}

func (a *Adapter) buildOneCondition(c aegis.Where) (string, []any) {
	col := a.quoteIdent(c.Column)
	const ph = "?"

	switch c.Operator {
	case aegis.OpEq:
		return col + " = " + ph, []any{c.Value}
	case aegis.OpNe:
		return col + " != " + ph, []any{c.Value}
	case aegis.OpLt:
		return col + " < " + ph, []any{c.Value}
	case aegis.OpLte:
		return col + " <= " + ph, []any{c.Value}
	case aegis.OpGt:
		return col + " > " + ph, []any{c.Value}
	case aegis.OpGte:
		return col + " >= " + ph, []any{c.Value}
	case aegis.OpIn:
		vals := c.Value.([]any)
		placeholders := strings.Repeat(ph+", ", len(vals)-1) + ph
		return col + " IN (" + placeholders + ")", vals
	case aegis.OpNotIn:
		vals := c.Value.([]any)
		placeholders := strings.Repeat(ph+", ", len(vals)-1) + ph
		return col + " NOT IN (" + placeholders + ")", vals
	case aegis.OpContains:
		s, ok := c.Value.(string)
		if !ok {
			return "", nil
		}
		return col + " LIKE " + ph + " ESCAPE '\\'", []any{"%" + escapeLike(s) + "%"}
	case aegis.OpStartsWith:
		s, ok := c.Value.(string)
		if !ok {
			return "", nil
		}
		return col + " LIKE " + ph + " ESCAPE '\\'", []any{escapeLike(s) + "%"}
	case aegis.OpEndsWith:
		s, ok := c.Value.(string)
		if !ok {
			return "", nil
		}
		return col + " LIKE " + ph + " ESCAPE '\\'", []any{"%" + escapeLike(s)}
	case aegis.OpIsNull:
		return col + " IS NULL", nil
	case aegis.OpIsNotNull:
		return col + " IS NOT NULL", nil
	default:
		return "", nil
	}
}

// escapeLike escapes %, _, and \ for use in LIKE patterns with ESCAPE '\\'.
func escapeLike(s string) string {
	var b strings.Builder
	for _, r := range s {
		switch r {
		case '\\', '%', '_':
			b.WriteRune('\\')
			b.WriteRune(r)
		default:
			b.WriteRune(r)
		}
	}
	return b.String()
}
