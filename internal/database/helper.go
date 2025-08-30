package database

import (
	"context"

	"github.com/thecodearcher/aegis"
)

func FindOne[T aegis.Model](ctx context.Context, db aegis.DatabaseAdapter, schema aegis.Schema[T], conditions []aegis.Where) (*T, error) {
	result, err := db.FindOne(ctx, schema.GetTableName(), conditions)
	if err != nil {
		return nil, err
	}

	model := schema.FromStorage(result)
	return &model, nil
}
