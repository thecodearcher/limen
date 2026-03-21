package limen

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"regexp"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

var testSecret = []byte("01234567890123456789012345678901")

// ---------------------------------------------------------------------------
// In-memory DatabaseAdapter for tests
// ---------------------------------------------------------------------------

type memTable struct {
	rows   []map[string]any
	nextID int64
}

type testMemoryAdapter struct {
	mu     sync.Mutex
	tables map[SchemaTableName]*memTable
}

func (a *testMemoryAdapter) table(name SchemaTableName) *memTable {
	t, ok := a.tables[name]
	if !ok {
		t = &memTable{nextID: 1}
		a.tables[name] = t
	}
	return t
}

func (a *testMemoryAdapter) Create(_ context.Context, tableName SchemaTableName, data map[string]any) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(data) == 0 {
		return fmt.Errorf("no data to insert")
	}

	tbl := a.table(tableName)

	row := make(map[string]any, len(data))
	for k, v := range data {
		row[k] = derefPointer(v)
	}

	if _, hasID := row["id"]; !hasID {
		row["id"] = tbl.nextID
		tbl.nextID++
	}

	tbl.rows = append(tbl.rows, row)
	return nil
}

func (a *testMemoryAdapter) FindOne(_ context.Context, tableName SchemaTableName, conditions []Where, orderBy []OrderBy) (map[string]any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	tbl := a.table(tableName)
	matched := filterRows(tbl.rows, conditions)
	sortRows(matched, orderBy)

	if len(matched) == 0 {
		return nil, ErrRecordNotFound
	}
	return maps.Clone(matched[0]), nil
}

func (a *testMemoryAdapter) FindMany(_ context.Context, tableName SchemaTableName, conditions []Where, options *QueryOptions) ([]map[string]any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	tbl := a.table(tableName)
	matched := filterRows(tbl.rows, conditions)

	if options != nil {
		sortRows(matched, options.OrderBy)
		if options.Offset > 0 {
			if options.Offset < len(matched) {
				matched = matched[options.Offset:]
			} else {
				matched = nil
			}
		}
		if options.Limit > 0 && options.Limit < len(matched) {
			matched = matched[:options.Limit]
		}
	}

	results := make([]map[string]any, len(matched))
	for i, r := range matched {
		results[i] = maps.Clone(r)
	}
	return results, nil
}

func (a *testMemoryAdapter) Update(_ context.Context, tableName SchemaTableName, conditions []Where, updates map[string]any) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(updates) == 0 {
		return nil
	}

	tbl := a.table(tableName)
	for _, row := range tbl.rows {
		if matchesConditions(row, conditions) {
			for k, v := range updates {
				row[k] = derefPointer(v)
			}
		}
	}
	return nil
}

func (a *testMemoryAdapter) Delete(_ context.Context, tableName SchemaTableName, conditions []Where) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	tbl := a.table(tableName)
	tbl.rows = slices.DeleteFunc(tbl.rows, func(row map[string]any) bool {
		return matchesConditions(row, conditions)
	})
	return nil
}

func (a *testMemoryAdapter) Exists(_ context.Context, tableName SchemaTableName, conditions []Where) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	tbl := a.table(tableName)
	for _, row := range tbl.rows {
		if matchesConditions(row, conditions) {
			return true, nil
		}
	}
	return false, nil
}

func (a *testMemoryAdapter) Count(_ context.Context, tableName SchemaTableName, conditions []Where) (int64, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	tbl := a.table(tableName)
	var count int64
	for _, row := range tbl.rows {
		if matchesConditions(row, conditions) {
			count++
		}
	}
	return count, nil
}

// derefPointer flattens the pointer types that ToStorage produces (*string,
// *time.Time) into plain values so FromStorage type assertions work the same
// way they do with real SQL drivers.
func derefPointer(v any) any {
	switch p := v.(type) {
	case *string:
		if p == nil {
			return nil
		}
		return *p
	case *time.Time:
		if p == nil {
			return nil
		}
		return *p
	default:
		return v
	}
}

// ---------------------------------------------------------------------------
// Condition matching
// ---------------------------------------------------------------------------

func filterRows(rows []map[string]any, conditions []Where) []map[string]any {
	if len(conditions) == 0 {
		return slices.Clone(rows)
	}
	var out []map[string]any
	for _, row := range rows {
		if matchesConditions(row, conditions) {
			out = append(out, row)
		}
	}
	return out
}

func matchesConditions(row map[string]any, conditions []Where) bool {
	if len(conditions) == 0 {
		return true
	}
	groups := GroupConditionsByConnector(conditions)
	for _, group := range groups {
		if !matchesGroup(row, group) {
			return false
		}
	}
	return true
}

// matchesGroup returns true when at least one condition in the group matches
// (OR semantics within a group, AND semantics between groups).
func matchesGroup(row map[string]any, group []Where) bool {
	for _, c := range group {
		if matchesSingle(row, c) {
			return true
		}
	}
	return false
}

func matchesSingle(row map[string]any, c Where) bool {
	val := row[c.Column]

	switch c.Operator {
	case OpIsNull:
		return val == nil
	case OpIsNotNull:
		return val != nil
	case OpEq, "":
		return valuesEqual(val, c.Value)
	case OpNe:
		return !valuesEqual(val, c.Value)
	case OpLt:
		cmp, ok := compareValues(val, c.Value)
		return ok && cmp < 0
	case OpLte:
		cmp, ok := compareValues(val, c.Value)
		return ok && cmp <= 0
	case OpGt:
		cmp, ok := compareValues(val, c.Value)
		return ok && cmp > 0
	case OpGte:
		cmp, ok := compareValues(val, c.Value)
		return ok && cmp >= 0
	case OpContains:
		return strings.Contains(fmt.Sprint(val), fmt.Sprint(c.Value))
	case OpStartsWith:
		return strings.HasPrefix(fmt.Sprint(val), fmt.Sprint(c.Value))
	case OpEndsWith:
		return strings.HasSuffix(fmt.Sprint(val), fmt.Sprint(c.Value))
	case OpIn:
		vals, ok := c.Value.([]any)
		if !ok {
			return false
		}
		for _, v := range vals {
			if valuesEqual(val, v) {
				return true
			}
		}
		return false
	case OpNotIn:
		vals, ok := c.Value.([]any)
		if !ok {
			return true
		}
		for _, v := range vals {
			if valuesEqual(val, v) {
				return false
			}
		}
		return true
	default:
		return valuesEqual(val, c.Value)
	}
}

func valuesEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a == b
}

func compareValues(a, b any) (int, bool) {
	switch av := a.(type) {
	case time.Time:
		bv, ok := b.(time.Time)
		if !ok {
			return 0, false
		}
		return av.Compare(bv), true
	case int64:
		bv, ok := b.(int64)
		if !ok {
			return 0, false
		}
		if av < bv {
			return -1, true
		}
		if av > bv {
			return 1, true
		}
		return 0, true
	case string:
		bv, ok := b.(string)
		if !ok {
			return 0, false
		}
		if av < bv {
			return -1, true
		}
		if av > bv {
			return 1, true
		}
		return 0, true
	default:
		return 0, false
	}
}

// ---------------------------------------------------------------------------
// Row sorting
// ---------------------------------------------------------------------------

func sortRows(rows []map[string]any, orderBy []OrderBy) {
	if len(orderBy) == 0 {
		return
	}
	slices.SortStableFunc(rows, func(a, b map[string]any) int {
		for _, ob := range orderBy {
			if ob.Column == "" {
				continue
			}
			cmp, ok := compareValues(a[ob.Column], b[ob.Column])
			if !ok || cmp == 0 {
				continue
			}
			if ob.Direction == OrderByDesc {
				return -cmp
			}
			return cmp
		}
		return 0
	})
}

// ---------------------------------------------------------------------------
// Test helpers
// ---------------------------------------------------------------------------

func newTestMemoryAdapter(t *testing.T) DatabaseAdapter {
	t.Helper()
	return &testMemoryAdapter{
		tables: make(map[SchemaTableName]*memTable),
	}
}

// newTestLimen creates a fully-initialized *Limen backed by an in-memory adapter.
func newTestLimen(t *testing.T, plugins ...Plugin) *Limen {
	t.Helper()

	db := newTestMemoryAdapter(t)

	l, err := New(&Config{
		BaseURL:  "http://localhost:8080",
		Database: db,
		Secret:   testSecret,
		Plugins:  plugins,
	})
	require.NoError(t, err)
	return l
}

// newTestLimenWithSessionConfig is like newTestLimen but lets callers
// customize the session configuration (e.g. short-session duration, bearer).
func newTestLimenWithSessionConfig(t *testing.T, opts ...SessionConfigOption) *Limen {
	t.Helper()

	db := newTestMemoryAdapter(t)

	l, err := New(&Config{
		BaseURL:  "http://localhost:8080",
		Database: db,
		Secret:   testSecret,
		Session:  NewDefaultSessionConfig(opts...),
	})
	require.NoError(t, err)
	return l
}

// newTestHTTPCore creates an LimenHTTPCore for handler-level tests from a *Limen.
func newTestHTTPCore(t *testing.T, l *Limen) *LimenHTTPCore {
	t.Helper()
	return &LimenHTTPCore{
		Responder:              newResponder(l.config.HTTP, l.core.cookies, l.config.Session.BearerEnabled),
		authInstance:           l,
		core:                   l.core,
		config:                 l.config.HTTP,
		trustedOriginsPatterns: []*regexp.Regexp{},
	}
}

// seedUser inserts a user directly into the in-memory DB and returns the
// generated ID. Useful for tests that validate sessions or look up users.
func seedUser(t *testing.T, l *Limen, email string) any {
	t.Helper()
	ctx := context.Background()
	extra := map[string]any{
		"first_name": "Test",
	}
	err := l.core.DBAction.CreateUser(ctx, &User{Email: email}, extra)
	require.NoError(t, err)

	user, err := l.core.DBAction.FindUserByEmail(ctx, email)
	require.NoError(t, err)
	return user.ID
}

// seedSession creates a real session in the DB via the real SessionManager and
// returns the SessionResult (token + cookie). The user must already exist.
func seedSession(t *testing.T, l *Limen, userID any, email string) *SessionResult {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "/signin", nil)
	auth := &AuthenticationResult{User: &User{ID: userID, Email: email}}
	result, err := l.core.SessionManager.CreateSession(context.Background(), req, auth, false)
	require.NoError(t, err)
	return result
}

// ---------------------------------------------------------------------------
// Test plugin
// ---------------------------------------------------------------------------

func newTestPlugin(t *testing.T) Plugin {
	t.Helper()
	return &testPlugin{}
}

// newTestPlugin creates a new test plugin.
type testPlugin struct {
}

func (p *testPlugin) Name() PluginName {
	return "test"
}

func (p *testPlugin) Initialize(core *LimenCore) error {
	return nil
}

func (p *testPlugin) PluginHTTPConfig() PluginHTTPConfig {
	return PluginHTTPConfig{
		BasePath: "/test",
	}
}

func (p *testPlugin) GetSchemas(schema *SchemaConfig) []SchemaIntrospector {
	return []SchemaIntrospector{}
}

func (p *testPlugin) RegisterRoutes(httpCore *LimenHTTPCore, routeBuilder *RouteBuilder) {
	routeBuilder.AddRoute(MethodGET, "/test", RouteID("test"), func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}, nil, nil)
}

func (p *testPlugin) TestMethodOnPlugin() string {
	return "test-method-on-plugin"
}
