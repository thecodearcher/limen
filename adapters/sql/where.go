package sql

import (
	"strings"

	"github.com/thecodearcher/limen"
)

// buildWhere returns a WHERE clause (without "WHERE") and args. Uses ? placeholders; caller must rebind before execute.
// Conditions are grouped by OR runs; groups are then AND'd. E.g. [A, B OR C, D] → (A) AND (B OR C) AND (D).
func (a *Adapter) buildWhere(conditions []limen.Where) (string, []any) {
	if len(conditions) == 0 {
		return "", nil
	}

	if len(conditions) == 1 {
		return a.buildOneCondition(conditions[0])
	}

	groups := limen.GroupConditionsByConnector(conditions)
	buildGroup := func(group []limen.Where) (string, []any) {
		clause, args := limen.BuildGroupClause(group, a.buildOneCondition)
		if clause != "" && len(group) > 1 {
			clause = "(" + clause + ")"
		}
		return clause, args
	}
	return limen.BuildWhereFromGroups(groups, buildGroup)
}

func (a *Adapter) buildOneCondition(c limen.Where) (string, []any) {
	col := a.quoteIdent(c.Column)
	const ph = "?"

	switch c.Operator {
	case limen.OpEq:
		return col + " = " + ph, []any{c.Value}
	case limen.OpNe:
		return col + " != " + ph, []any{c.Value}
	case limen.OpLt:
		return col + " < " + ph, []any{c.Value}
	case limen.OpLte:
		return col + " <= " + ph, []any{c.Value}
	case limen.OpGt:
		return col + " > " + ph, []any{c.Value}
	case limen.OpGte:
		return col + " >= " + ph, []any{c.Value}
	case limen.OpIn:
		vals := c.Value.([]any)
		placeholders := strings.Repeat(ph+", ", len(vals)-1) + ph
		return col + " IN (" + placeholders + ")", vals
	case limen.OpNotIn:
		vals := c.Value.([]any)
		placeholders := strings.Repeat(ph+", ", len(vals)-1) + ph
		return col + " NOT IN (" + placeholders + ")", vals
	case limen.OpContains:
		s, ok := c.Value.(string)
		if !ok {
			return "", nil
		}
		return col + " LIKE " + ph + " ESCAPE '\\'", []any{"%" + escapeLike(s) + "%"}
	case limen.OpStartsWith:
		s, ok := c.Value.(string)
		if !ok {
			return "", nil
		}
		return col + " LIKE " + ph + " ESCAPE '\\'", []any{escapeLike(s) + "%"}
	case limen.OpEndsWith:
		s, ok := c.Value.(string)
		if !ok {
			return "", nil
		}
		return col + " LIKE " + ph + " ESCAPE '\\'", []any{"%" + escapeLike(s)}
	case limen.OpIsNull:
		return col + " IS NULL", nil
	case limen.OpIsNotNull:
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
