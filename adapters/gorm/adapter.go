package gorm

import (
	"context"

	"gorm.io/gorm"

	"github.com/thecodearcher/aegis"
)

// Adapter implements aegis.DatabaseAdapter using GORM
type Adapter struct {
	db *gorm.DB
}

// New creates a new GORM adapter
func New(db *gorm.DB) *Adapter {
	return &Adapter{db: db}
}

func (a *Adapter) Create(ctx context.Context, tableName aegis.TableName, data map[string]any) error {
	return a.db.WithContext(ctx).Table(string(tableName)).Create(data).Error
}

func (a *Adapter) FindOne(ctx context.Context, tableName aegis.TableName, conditions []aegis.Where, orderBy []aegis.OrderBy) (map[string]any, error) {
	var result map[string]any
	query := a.db.WithContext(ctx).Table(string(tableName))

	query = a.applyConditions(query, conditions)

	for _, orderBy := range orderBy {
		query = query.Order(orderBy.Column + " " + string(orderBy.Direction))
	}

	err := query.Take(&result).Error
	return result, err
}

func (a *Adapter) FindMany(ctx context.Context, tableName aegis.TableName, conditions []aegis.Where, options *aegis.QueryOptions) ([]map[string]any, error) {
	var results []map[string]any
	query := a.db.WithContext(ctx).Table(string(tableName))

	query = a.applyConditions(query, conditions)

	if options != nil {
		if options.Limit > 0 {
			query = query.Limit(options.Limit)
		}
		if options.Offset > 0 {
			query = query.Offset(options.Offset)
		}
		for _, orderBy := range options.OrderBy {
			query = query.Order(orderBy.Column + " " + string(orderBy.Direction))
		}
	}

	err := query.Find(&results).Error
	return results, err
}

func (a *Adapter) Update(ctx context.Context, tableName aegis.TableName, conditions []aegis.Where, updates map[string]any) error {
	query := a.db.WithContext(ctx).Table(string(tableName))
	query = a.applyConditions(query, conditions)
	return query.Updates(updates).Error
}

func (a *Adapter) Delete(ctx context.Context, tableName aegis.TableName, conditions []aegis.Where) error {
	query := a.db.WithContext(ctx).Table(string(tableName))
	query = a.applyConditions(query, conditions)
	return query.Delete(nil).Error
}

func (a *Adapter) Exists(ctx context.Context, tableName aegis.TableName, conditions []aegis.Where) (bool, error) {
	var count int64
	query := a.db.WithContext(ctx).Table(string(tableName))
	query = a.applyConditions(query, conditions)
	err := query.Count(&count).Error
	return count > 0, err
}

func (a *Adapter) Count(ctx context.Context, tableName aegis.TableName, conditions []aegis.Where) (int64, error) {
	var count int64
	query := a.db.WithContext(ctx).Table(string(tableName))
	query = a.applyConditions(query, conditions)
	err := query.Count(&count).Error
	return count, err
}

func (a *Adapter) applyConditions(query *gorm.DB, conditions []aegis.Where) *gorm.DB {
	for i, condition := range conditions {
		whereClause, args := a.buildWhereClause(condition)

		if i == 0 || condition.Connector == aegis.ConnectorAnd {
			query = query.Where(whereClause, args...)
		} else {
			query = query.Or(whereClause, args...)
		}
	}
	return query
}

func (a *Adapter) buildWhereClause(condition aegis.Where) (string, []any) {
	switch condition.Operator {
	case aegis.OpEq:
		return condition.Column + " = ?", []any{condition.Value}
	case aegis.OpNe:
		return condition.Column + " != ?", []any{condition.Value}
	case aegis.OpLt:
		return condition.Column + " < ?", []any{condition.Value}
	case aegis.OpLte:
		return condition.Column + " <= ?", []any{condition.Value}
	case aegis.OpGt:
		return condition.Column + " > ?", []any{condition.Value}
	case aegis.OpGte:
		return condition.Column + " >= ?", []any{condition.Value}
	case aegis.OpIn:
		return condition.Column + " IN ?", []any{condition.Value}
	case aegis.OpNotIn:
		return condition.Column + " NOT IN ?", []any{condition.Value}
	case aegis.OpContains:
		return condition.Column + " LIKE ?", []any{"%" + condition.Value.(string) + "%"}
	case aegis.OpStartsWith:
		return condition.Column + " LIKE ?", []any{condition.Value.(string) + "%"}
	case aegis.OpEndsWith:
		return condition.Column + " LIKE ?", []any{"%" + condition.Value.(string)}
	case aegis.OpIsNull:
		return condition.Column + " IS NULL", nil
	case aegis.OpIsNotNull:
		return condition.Column + " IS NOT NULL", nil
	default:
		return "", nil
	}
}
