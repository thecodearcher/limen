package gorm

import (
	"context"
	"errors"
	"fmt"

	"gorm.io/gorm"

	"github.com/thecodearcher/limen"
)

// Adapter implements limen.DatabaseAdapter using GORM
type Adapter struct {
	db *gorm.DB // Regular DB connection
	tx *gorm.DB // Transaction DB (nil when not in transaction)
}

// New creates a new GORM adapter
func New(db *gorm.DB) *Adapter {
	return &Adapter{db: db}
}

// getDB returns the transaction DB if in a transaction, otherwise returns the regular DB
func (a *Adapter) getDB() *gorm.DB {
	if a.tx != nil {
		return a.tx
	}
	return a.db
}

// BeginTx starts a new transaction
func (a *Adapter) BeginTx(ctx context.Context) (limen.DatabaseTx, error) {
	tx := a.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	return &Adapter{
		db: a.db,
		tx: tx,
	}, nil
}

// Commit commits the transaction if this adapter is in transaction mode
func (a *Adapter) Commit() error {
	if a.tx == nil {
		return fmt.Errorf("not in a transaction")
	}
	err := a.tx.Commit().Error
	a.tx = nil
	return err
}

// Rollback rolls back the transaction if this adapter is in transaction mode
func (a *Adapter) Rollback() error {
	if a.tx == nil {
		return fmt.Errorf("not in a transaction")
	}
	err := a.tx.Rollback().Error
	a.tx = nil // Clear transaction state
	return err
}

func (a *Adapter) Create(ctx context.Context, tableName limen.SchemaTableName, data map[string]any) error {
	db := a.getDB()
	return db.WithContext(ctx).Table(string(tableName)).Create(data).Error
}

func (a *Adapter) FindOne(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where, orderBy []limen.OrderBy) (map[string]any, error) {
	var result map[string]any
	db := a.getDB()
	query := db.WithContext(ctx).Table(string(tableName))

	query = a.applyConditions(query, conditions)

	for _, orderBy := range orderBy {
		query = query.Order(orderBy.Column + " " + string(orderBy.Direction))
	}

	err := query.Take(&result).Error
	return result, a.formatError(err)
}

func (a *Adapter) FindMany(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where, options *limen.QueryOptions) ([]map[string]any, error) {
	var results []map[string]any
	db := a.getDB()
	query := db.WithContext(ctx).Table(string(tableName))

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
	return results, a.formatError(err)
}

func (a *Adapter) Update(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where, updates map[string]any) error {
	db := a.getDB()
	query := db.WithContext(ctx).Table(string(tableName))
	query = a.applyConditions(query, conditions)
	return query.Updates(updates).Error
}

func (a *Adapter) Delete(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where) error {
	db := a.getDB()
	query := db.WithContext(ctx).Table(string(tableName))
	query = a.applyConditions(query, conditions)
	return query.Delete(nil).Error
}

func (a *Adapter) Exists(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where) (bool, error) {
	var count int64
	db := a.getDB()
	query := db.WithContext(ctx).Table(string(tableName))
	query = a.applyConditions(query, conditions)
	err := query.Count(&count).Error
	return count > 0, err
}

func (a *Adapter) Count(ctx context.Context, tableName limen.SchemaTableName, conditions []limen.Where) (int64, error) {
	var count int64
	db := a.getDB()
	query := db.WithContext(ctx).Table(string(tableName))
	query = a.applyConditions(query, conditions)
	err := query.Count(&count).Error
	return count, err
}

func (a *Adapter) applyConditions(query *gorm.DB, conditions []limen.Where) *gorm.DB {
	if len(conditions) == 0 {
		return query
	}

	if len(conditions) == 1 {
		clause, args := a.buildWhereClause(conditions[0])
		if clause == "" {
			return query
		}
		return query.Where(clause, args...)
	}

	groups := limen.GroupConditionsByConnector(conditions)
	for _, group := range groups {
		query = a.applyGroup(query, group)
	}
	return query
}

// applyGroup applies one group (single condition or OR of several) as one Where.
func (a *Adapter) applyGroup(query *gorm.DB, group []limen.Where) *gorm.DB {
	if len(group) == 0 {
		return query
	}
	clause, args := limen.BuildGroupClause(group, a.buildWhereClause)
	if clause == "" {
		return query
	}
	return query.Where(clause, args...)
}

func (a *Adapter) buildWhereClause(condition limen.Where) (string, []any) {
	switch condition.Operator {
	case limen.OpEq:
		return condition.Column + " = ?", []any{condition.Value}
	case limen.OpNe:
		return condition.Column + " != ?", []any{condition.Value}
	case limen.OpLt:
		return condition.Column + " < ?", []any{condition.Value}
	case limen.OpLte:
		return condition.Column + " <= ?", []any{condition.Value}
	case limen.OpGt:
		return condition.Column + " > ?", []any{condition.Value}
	case limen.OpGte:
		return condition.Column + " >= ?", []any{condition.Value}
	case limen.OpIn:
		return condition.Column + " IN ?", []any{condition.Value}
	case limen.OpNotIn:
		return condition.Column + " NOT IN ?", []any{condition.Value}
	case limen.OpContains:
		return condition.Column + " LIKE ?", []any{"%" + condition.Value.(string) + "%"}
	case limen.OpStartsWith:
		return condition.Column + " LIKE ?", []any{condition.Value.(string) + "%"}
	case limen.OpEndsWith:
		return condition.Column + " LIKE ?", []any{"%" + condition.Value.(string)}
	case limen.OpIsNull:
		return condition.Column + " IS NULL", nil
	case limen.OpIsNotNull:
		return condition.Column + " IS NOT NULL", nil
	default:
		return "", nil
	}
}

func (a *Adapter) formatError(err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return limen.ErrRecordNotFound
	}
	return err
}
