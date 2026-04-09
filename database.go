package limen

import (
	"context"
	"strings"
)

type DatabaseAdapter interface {
	Create(ctx context.Context, tableName SchemaTableName, data map[string]any) error
	FindOne(ctx context.Context, tableName SchemaTableName, conditions []Where, orderBy []OrderBy) (map[string]any, error)
	FindMany(ctx context.Context, tableName SchemaTableName, conditions []Where, options *QueryOptions) ([]map[string]any, error)
	Update(ctx context.Context, tableName SchemaTableName, conditions []Where, updates map[string]any) error
	Delete(ctx context.Context, tableName SchemaTableName, conditions []Where) error
	Exists(ctx context.Context, tableName SchemaTableName, conditions []Where) (bool, error)
	Count(ctx context.Context, tableName SchemaTableName, conditions []Where) (int64, error)
}

// DatabaseTx represents a database transaction
type DatabaseTx interface {
	DatabaseAdapter
	Commit() error
	Rollback() error
}

// TransactionalAdapter is implemented by adapters that support transactions
type TransactionalAdapter interface {
	BeginTx(ctx context.Context) (DatabaseTx, error)
}

// Where represents a typed condition for database queries
type Where struct {
	Column    string    `json:"column"`
	Operator  Operator  `json:"operator"`  // "eq" by default
	Value     any       `json:"value"`     // string | number | boolean | []string | []number | time.Time | nil
	Connector Connector `json:"connector"` // "AND" by default, "OR" for multiple conditions
}

// Operator defines the comparison operation
type Operator string

const (
	OpEq         Operator = "eq"          // equals
	OpNe         Operator = "ne"          // not equals
	OpLt         Operator = "lt"          // less than
	OpLte        Operator = "lte"         // less than or equal
	OpGt         Operator = "gt"          // greater than
	OpGte        Operator = "gte"         // greater than or equal
	OpIn         Operator = "in"          // in array
	OpNotIn      Operator = "not_in"      // not in array
	OpContains   Operator = "contains"    // contains substring
	OpStartsWith Operator = "starts_with" // starts with
	OpEndsWith   Operator = "ends_with"   // ends with
	OpIsNull     Operator = "is_null"     // is null
	OpIsNotNull  Operator = "is_not_null" // is not null
)

// Connector defines how conditions are joined
type Connector string

const (
	ConnectorAnd Connector = "AND"
	ConnectorOr  Connector = "OR"
)

// GroupConditionsByConnector splits conditions into groups: each group is either a single
// condition (AND) or a run of conditions connected by a connector (AND or OR). Groups are AND'd in order.
// E.g. [A, B.Or(), C, D] → [[A], [B,C], [D]] meaning (A) AND (B OR C) AND (D).
// Adapters can use this to build WHERE clauses with consistent OR precedence.
func GroupConditionsByConnector(conditions []Where) [][]Where {
	if len(conditions) == 0 {
		return nil
	}
	var groups [][]Where
	var current []Where
	for i, c := range conditions {
		if i == 0 {
			current = []Where{c}
			continue
		}

		if c.Connector == ConnectorOr {
			current = append(current, c)
			continue
		}

		groups = append(groups, current)
		current = []Where{c}
	}

	if len(current) > 0 {
		groups = append(groups, current)
	}
	return groups
}

// BuildGroupClause builds a single connector clause from a group of conditions.
// buildCondition is adapter-specific (e.g. column quoting, placeholder style). Returns combined
// clause and args; caller may wrap in parens when len(group) > 1 (e.g. for SQL).
func BuildGroupClause(group []Where, buildCondition func(Where) (string, []any)) (clause string, args []any) {
	var clauses []string
	for _, c := range group {
		part, partArgs := buildCondition(c)
		if part == "" {
			continue
		}
		clauses = append(clauses, part)
		args = append(args, partArgs...)
	}
	if len(clauses) == 0 {
		return "", nil
	}
	if len(clauses) == 1 {
		return clauses[0], args
	}
	return strings.Join(clauses, " OR "), args
}

// BuildWhereFromGroups builds a full WHERE expression by joining group clauses with AND.
// buildGroup turns each group into (clause, args); empty clauses are skipped.
func BuildWhereFromGroups(groups [][]Where, buildGroup func([]Where) (string, []any)) (clause string, args []any) {
	var parts []string
	for _, group := range groups {
		c, groupArgs := buildGroup(group)
		if c == "" {
			continue
		}
		parts = append(parts, c)
		args = append(args, groupArgs...)
	}
	return strings.Join(parts, " AND "), args
}

type OrderByDirection string

const (
	OrderByAsc  OrderByDirection = "ASC"  // order by ascending i.e oldest at top
	OrderByDesc OrderByDirection = "DESC" // order by descending i.e newest at top
)

type OrderBy struct {
	Column    string
	Direction OrderByDirection
}

// QueryOptions for additional query parameters
type QueryOptions struct {
	Limit   int
	Offset  int
	OrderBy []OrderBy
}

// Helper functions for building conditions

// Eq creates an equality condition
func Eq(column string, value any) Where {
	return Where{Column: column, Operator: OpEq, Value: value, Connector: ConnectorAnd}
}

// Ne creates a not-equals condition
func Ne(column string, value any) Where {
	return Where{Column: column, Operator: OpNe, Value: value, Connector: ConnectorAnd}
}

// Lt creates a less-than condition
func Lt(column string, value any) Where {
	return Where{Column: column, Operator: OpLt, Value: value, Connector: ConnectorAnd}
}

// Lte creates a less-than-or-equal condition
func Lte(column string, value any) Where {
	return Where{Column: column, Operator: OpLte, Value: value, Connector: ConnectorAnd}
}

// Gt creates a greater-than condition
func Gt(column string, value any) Where {
	return Where{Column: column, Operator: OpGt, Value: value, Connector: ConnectorAnd}
}

// Gte creates a greater-than-or-equal condition
func Gte(column string, value any) Where {
	return Where{Column: column, Operator: OpGte, Value: value, Connector: ConnectorAnd}
}

// In creates an IN condition
func In(column string, values []any) Where {
	return Where{Column: column, Operator: OpIn, Value: values, Connector: ConnectorAnd}
}

// NotIn creates a NOT IN condition
func NotIn(column string, values []any) Where {
	return Where{Column: column, Operator: OpNotIn, Value: values, Connector: ConnectorAnd}
}

// Contains creates a contains substring condition
func Contains(column, value string) Where {
	return Where{Column: column, Operator: OpContains, Value: value, Connector: ConnectorAnd}
}

// StartsWith creates a starts-with condition
func StartsWith(column, value string) Where {
	return Where{Column: column, Operator: OpStartsWith, Value: value, Connector: ConnectorAnd}
}

// EndsWith creates an ends-with condition
func EndsWith(column, value string) Where {
	return Where{Column: column, Operator: OpEndsWith, Value: value, Connector: ConnectorAnd}
}

// IsNull creates an IS NULL condition
func IsNull(column string) Where {
	return Where{Column: column, Operator: OpIsNull, Value: nil, Connector: ConnectorAnd}
}

// IsNotNull creates an IS NOT NULL condition
func IsNotNull(column string) Where {
	return Where{Column: column, Operator: OpIsNotNull, Value: nil, Connector: ConnectorAnd}
}

// Or modifier to change connector to OR
func (w Where) Or() Where {
	w.Connector = ConnectorOr
	return w
}
