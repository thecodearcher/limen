package limen

import (
	"context"
	"fmt"
	"maps"
	"net/http"
	"net/http/httptest"
	"slices"
	"strings"
	"sync"
	"testing"
	"time"
)

// testSecret is a fixed 32-byte key for deterministic test encryption.
var testSecret = []byte("01234567890123456789012345678901")

// ---------------------------------------------------------------------------
// In-memory DatabaseAdapter
// ---------------------------------------------------------------------------

type testMemTable struct {
	rows   []map[string]any
	nextID int64
}

type testMemoryAdapter struct {
	mu     sync.Mutex
	tables map[SchemaTableName]*testMemTable
}

func newTestMemoryAdapter(t *testing.T) *testMemoryAdapter {
	t.Helper()
	return &testMemoryAdapter{
		tables: make(map[SchemaTableName]*testMemTable),
	}
}

func (a *testMemoryAdapter) table(name SchemaTableName) *testMemTable {
	t, ok := a.tables[name]
	if !ok {
		t = &testMemTable{nextID: 1}
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
		row[k] = testDerefPointer(v)
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
	matched := testFilterRows(tbl.rows, conditions)
	testSortRows(matched, orderBy)

	if len(matched) == 0 {
		return nil, ErrRecordNotFound
	}
	return maps.Clone(matched[0]), nil
}

func (a *testMemoryAdapter) FindMany(_ context.Context, tableName SchemaTableName, conditions []Where, options *QueryOptions) ([]map[string]any, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	tbl := a.table(tableName)
	matched := testFilterRows(tbl.rows, conditions)

	if options != nil {
		testSortRows(matched, options.OrderBy)
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
		if testMatchesConditions(row, conditions) {
			for k, v := range updates {
				row[k] = testDerefPointer(v)
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
		return testMatchesConditions(row, conditions)
	})
	return nil
}

func (a *testMemoryAdapter) Exists(_ context.Context, tableName SchemaTableName, conditions []Where) (bool, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	tbl := a.table(tableName)
	for _, row := range tbl.rows {
		if testMatchesConditions(row, conditions) {
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
		if testMatchesConditions(row, conditions) {
			count++
		}
	}
	return count, nil
}

// testDerefPointer flattens pointer types that ToStorage produces (*string,
// *time.Time) into plain values so FromStorage type assertions match real SQL
// driver behavior.
func testDerefPointer(v any) any {
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

func testFilterRows(rows []map[string]any, conditions []Where) []map[string]any {
	if len(conditions) == 0 {
		return slices.Clone(rows)
	}
	var out []map[string]any
	for _, row := range rows {
		if testMatchesConditions(row, conditions) {
			out = append(out, row)
		}
	}
	return out
}

func testMatchesConditions(row map[string]any, conditions []Where) bool {
	if len(conditions) == 0 {
		return true
	}
	groups := GroupConditionsByConnector(conditions)
	for _, group := range groups {
		if !testMatchesGroup(row, group) {
			return false
		}
	}
	return true
}

func testMatchesGroup(row map[string]any, group []Where) bool {
	for _, c := range group {
		if testMatchesSingle(row, c) {
			return true
		}
	}
	return false
}

func testMatchesSingle(row map[string]any, c Where) bool {
	val := row[c.Column]

	switch c.Operator {
	case OpIsNull:
		return val == nil
	case OpIsNotNull:
		return val != nil
	case OpEq, "":
		return testValuesEqual(val, c.Value)
	case OpNe:
		return !testValuesEqual(val, c.Value)
	case OpLt:
		cmp, ok := testCompareValues(val, c.Value)
		return ok && cmp < 0
	case OpLte:
		cmp, ok := testCompareValues(val, c.Value)
		return ok && cmp <= 0
	case OpGt:
		cmp, ok := testCompareValues(val, c.Value)
		return ok && cmp > 0
	case OpGte:
		cmp, ok := testCompareValues(val, c.Value)
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
			if testValuesEqual(val, v) {
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
			if testValuesEqual(val, v) {
				return false
			}
		}
		return true
	default:
		return testValuesEqual(val, c.Value)
	}
}

func testValuesEqual(a, b any) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}
	return a == b
}

func testCompareValues(a, b any) (int, bool) {
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

func testSortRows(rows []map[string]any, orderBy []OrderBy) {
	if len(orderBy) == 0 {
		return
	}
	slices.SortStableFunc(rows, func(a, b map[string]any) int {
		for _, ob := range orderBy {
			if ob.Column == "" {
				continue
			}
			cmp, ok := testCompareValues(a[ob.Column], b[ob.Column])
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
// High-level test helpers
// ---------------------------------------------------------------------------

// NewTestLimen creates a fully-initialized *Limen backed by an in-memory
// adapter.
func NewTestLimen(t *testing.T, plugins ...Plugin) (*Limen, *LimenCore) {
	t.Helper()

	l, err := New(&Config{
		BaseURL:  "http://localhost:8080",
		Database: newTestMemoryAdapter(t),
		Secret:   testSecret,
		Plugins:  plugins,
	})
	if err != nil {
		t.Fatalf("NewTestLimen: %v", err)
	}
	return l, l.core
}

// SeedTestUser inserts a user directly into the in-memory DB and returns the
// full *User. The Limen instance must have been created with NewTestLimen.
func SeedTestUser(t *testing.T, l *Limen, email string) *User {
	t.Helper()
	ctx := context.Background()
	extra := map[string]any{"first_name": "Test"}
	if err := l.core.DBAction.CreateUser(ctx, &User{Email: email}, extra); err != nil {
		t.Fatalf("SeedTestUser: %v", err)
	}
	user, err := l.core.DBAction.FindUserByEmail(ctx, email)
	if err != nil {
		t.Fatalf("SeedTestUser find: %v", err)
	}
	return user
}

// SeedTestSession creates a session via the real SessionManager and returns the
// SessionResult. The user must already exist.
func SeedTestSession(t *testing.T, l *Limen, userID any, email string) *SessionResult {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", http.NoBody)
	auth := &AuthenticationResult{User: &User{ID: userID, Email: email}}
	result, err := l.core.SessionManager.CreateSession(context.Background(), req, auth, false)
	if err != nil {
		t.Fatalf("SeedTestSession: %v", err)
	}
	return result
}
