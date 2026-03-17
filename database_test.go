package limen

import (
	"testing"
	"time"
)

func TestWhereConditions(t *testing.T) {
	tests := []struct {
		name      string
		condition Where
		expected  Where
	}{
		{
			name:      "Eq condition",
			condition: Eq("email", "test@example.com"),
			expected: Where{
				Column:    "email",
				Operator:  OpEq,
				Value:     "test@example.com",
				Connector: ConnectorAnd,
			},
		},
		{
			name:      "Ne condition",
			condition: Ne("status", "inactive"),
			expected: Where{
				Column:    "status",
				Operator:  OpNe,
				Value:     "inactive",
				Connector: ConnectorAnd,
			},
		},
		{
			name:      "Lt condition",
			condition: Lt("age", 18),
			expected: Where{
				Column:    "age",
				Operator:  OpLt,
				Value:     18,
				Connector: ConnectorAnd,
			},
		},
		{
			name:      "Contains condition",
			condition: Contains("name", "john"),
			expected: Where{
				Column:    "name",
				Operator:  OpContains,
				Value:     "john",
				Connector: ConnectorAnd,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.condition.Column != tt.expected.Column {
				t.Errorf("Expected field %s, got %s", tt.expected.Column, tt.condition.Column)
			}
			if tt.condition.Operator != tt.expected.Operator {
				t.Errorf("Expected operator %s, got %s", tt.expected.Operator, tt.condition.Operator)
			}

			if tt.condition.Value != tt.expected.Value {
				t.Errorf("Expected value %v, got %v", tt.expected.Value, tt.condition.Value)
			}
			if tt.condition.Connector != tt.expected.Connector {
				t.Errorf("Expected connector %s, got %s", tt.expected.Connector, tt.condition.Connector)
			}
		})
	}
}

func TestOrModifier(t *testing.T) {
	condition := Eq("email", "test@example.com").Or()

	if condition.Connector != ConnectorOr {
		t.Errorf("Expected connector to be OR, got %s", condition.Connector)
	}

	if condition.Column != "email" {
		t.Errorf("Expected field to remain 'email', got %s", condition.Column)
	}

	if condition.Operator != OpEq {
		t.Errorf("Expected operator to remain 'eq', got %s", condition.Operator)
	}
}

func TestQueryOptions(t *testing.T) {
	options := &QueryOptions{
		Limit:  10,
		Offset: 20,
		OrderBy: []OrderBy{
			{
				Column:    "created_at",
				Direction: OrderByDesc,
			},
			{},
			{
				Column:    "name",
				Direction: OrderByAsc,
			},
		},
	}

	if options.Limit != 10 {
		t.Errorf("Expected limit to be 10, got %d", options.Limit)
	}

	if options.Offset != 20 {
		t.Errorf("Expected offset to be 20, got %d", options.Offset)
	}

	if len(options.OrderBy) != 2 {
		t.Errorf("Expected 2 order by clauses, got %d", len(options.OrderBy))
	}

	if options.OrderBy[0].Column != "created_at" || options.OrderBy[0].Direction != OrderByDesc {
		t.Errorf("Expected first order by to be 'created_at DESC', got %s", options.OrderBy[0])
	}
}

func TestComplexConditions(t *testing.T) {
	now := time.Now()
	conditions := []Where{
		Eq("status", "active"),
		Gt("created_at", now.AddDate(0, -1, 0)),
		Contains("email", "gmail").Or(),
		In("role", []any{"admin", "user"}),
	}

	if len(conditions) != 4 {
		t.Errorf("Expected 4 conditions, got %d", len(conditions))
	}

	// Check that OR condition is properly set
	orCondition := conditions[2]
	if orCondition.Connector != ConnectorOr {
		t.Errorf("Expected third condition to have OR connector, got %s", orCondition.Connector)
	}

	// Check that other conditions have AND connector
	for i, condition := range conditions {
		if i == 2 {
			continue // Skip the OR condition
		}
		if condition.Connector != ConnectorAnd {
			t.Errorf("Expected condition %d to have AND connector, got %s", i, condition.Connector)
		}
	}
}
