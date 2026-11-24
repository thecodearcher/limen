package limen

import (
	"context"
	"fmt"
	"net/http"
)

// DatabaseHookFunc returning an error from a before hook aborts the write.
type DatabaseHookFunc func(hc *DatabaseHookContext) error

// DatabaseHook runs before or after writes to one schema, or to every schema when Schema is AnySchema.
type DatabaseHook struct {
	Schema    SchemaName
	Operation Operation
	Before    bool
	Run       DatabaseHookFunc
}

const AnySchema SchemaName = "*"

func BeforeCreate(schema SchemaName, fn DatabaseHookFunc) DatabaseHook {
	return DatabaseHook{Schema: schema, Operation: OperationCreate, Before: true, Run: fn}
}

func AfterCreate(schema SchemaName, fn DatabaseHookFunc) DatabaseHook {
	return DatabaseHook{Schema: schema, Operation: OperationCreate, Run: fn}
}

func BeforeUpdate(schema SchemaName, fn DatabaseHookFunc) DatabaseHook {
	return DatabaseHook{Schema: schema, Operation: OperationUpdate, Before: true, Run: fn}
}

func AfterUpdate(schema SchemaName, fn DatabaseHookFunc) DatabaseHook {
	return DatabaseHook{Schema: schema, Operation: OperationUpdate, Run: fn}
}

func BeforeDelete(schema SchemaName, fn DatabaseHookFunc) DatabaseHook {
	return DatabaseHook{Schema: schema, Operation: OperationDelete, Before: true, Run: fn}
}

func AfterDelete(schema SchemaName, fn DatabaseHookFunc) DatabaseHook {
	return DatabaseHook{Schema: schema, Operation: OperationDelete, Run: fn}
}

type DatabaseHookContext struct {
	Context context.Context
	// Request is nil outside a handler.
	Request    *http.Request
	Schema     Schema
	Operation  Operation
	SoftDelete bool
	// Payload is the column map the adapter receives. Nil for delete.
	Payload    map[string]any
	Conditions []Where
	// Result is nil in before hooks.
	Result *DatabaseResult
}

// Get reads a payload value by logical field.
func (hc *DatabaseHookContext) Get(field SchemaField) any {
	column := hc.Schema.GetField(field)
	if column == "" || hc.Payload == nil {
		return nil
	}
	return hc.Payload[column]
}

// Set writes a payload value by logical field.
func (hc *DatabaseHookContext) Set(field SchemaField, value any) {
	column := hc.Schema.GetField(field)
	if column == "" || hc.Payload == nil {
		return
	}
	hc.Payload[column] = value
}

func groupDatabaseHooks(hooks []DatabaseHook, schemas map[SchemaName]SchemaDefinition) (map[SchemaName][]DatabaseHook, error) {
	grouped := make(map[SchemaName][]DatabaseHook)
	for _, hook := range hooks {
		if hook.Run == nil {
			return nil, fmt.Errorf("database hook for schema %q has no Run function", hook.Schema)
		}
		switch hook.Operation {
		case OperationCreate, OperationUpdate, OperationDelete:
		default:
			return nil, fmt.Errorf("database hook for schema %q has unsupported operation %d", hook.Schema, hook.Operation)
		}
		if _, ok := schemas[hook.Schema]; !ok && hook.Schema != AnySchema {
			return nil, fmt.Errorf("database hook references unknown schema %q", hook.Schema)
		}
		grouped[hook.Schema] = append(grouped[hook.Schema], hook)
	}
	return grouped, nil
}

type contextKeyDatabaseHookNested struct{}

func (core *LimenCore) databaseHooksFor(hc *DatabaseHookContext, before bool) []DatabaseHook {
	if hc.Context.Value(contextKeyDatabaseHookNested{}) != nil {
		return nil
	}
	var matched []DatabaseHook
	for _, hooks := range [][]DatabaseHook{core.databaseHooks[AnySchema], core.databaseHooks[hc.Schema.GetSchemaName()]} {
		for _, hook := range hooks {
			if hook.Before == before && hook.Operation == hc.Operation {
				matched = append(matched, hook)
			}
		}
	}
	return matched
}

func (core *LimenCore) runBeforeHooks(hc *DatabaseHookContext) error {
	return runDatabaseHooks(hc, core.databaseHooksFor(hc, true))
}

func (core *LimenCore) runAfterHooks(hc *DatabaseHookContext, result DatabaseResult) error {
	hooks := core.databaseHooksFor(hc, false)
	if len(hooks) == 0 {
		return nil
	}
	hc.Result = &result
	if queue, ok := hc.Context.Value(contextKeyAfterHooks{}).(*[]func(context.Context) error); ok {
		*queue = append(*queue, func(ctx context.Context) error {
			hc.Context = ctx
			return runDatabaseHooks(hc, hooks)
		})
		return nil
	}
	return runDatabaseHooks(hc, hooks)
}

func runDatabaseHooks(hc *DatabaseHookContext, hooks []DatabaseHook) error {
	if len(hooks) == 0 {
		return nil
	}
	callerCtx := hc.Context
	hc.Context = context.WithValue(callerCtx, contextKeyDatabaseHookNested{}, true)
	hc.Request = getAdditionalFieldsContext(callerCtx).request
	defer func() { hc.Context = callerCtx }()

	for _, hook := range hooks {
		if err := hook.Run(hc); err != nil {
			return err
		}
	}
	return nil
}
