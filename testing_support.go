package limen

import (
	"cmp"
	"context"
	"fmt"
	"maps"
	"math"
	"net/http"
	"net/http/httptest"
	"reflect"
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

func (a *testMemoryAdapter) Create(_ context.Context, tableName SchemaTableName, data map[string]any) (DatabaseResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(data) == 0 {
		return DatabaseResult{}, fmt.Errorf("no data to insert")
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
	return DatabaseResult{RowsAffected: 1}, nil
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

func (a *testMemoryAdapter) Update(_ context.Context, tableName SchemaTableName, conditions []Where, updates map[string]any) (DatabaseResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(updates) == 0 {
		return DatabaseResult{}, nil
	}

	tbl := a.table(tableName)
	var rowsAffected int64
	for _, row := range tbl.rows {
		if !testMatchesConditions(row, conditions) {
			continue
		}
		for column, value := range updates {
			arithmeticUpdate, ok := value.(ArithmeticUpdate)
			if !ok {
				row[column] = testDerefPointer(value)
				continue
			}

			updated, err := testApplyArithmeticUpdate(row[column], arithmeticUpdate)
			if err != nil {
				return DatabaseResult{}, fmt.Errorf("update column %q: %w", column, err)
			}
			row[column] = updated
		}
		rowsAffected++
	}
	return DatabaseResult{RowsAffected: rowsAffected}, nil
}

func testApplyArithmeticUpdate(current any, update ArithmeticUpdate) (any, error) {
	if err := update.Validate(); err != nil {
		return nil, err
	}
	delta := update.Value()
	switch value := testDerefPointer(current).(type) {
	case int:
		return value + int(delta), nil
	case int32:
		sum := int64(value) + delta
		if sum > math.MaxInt32 || sum < math.MinInt32 {
			return nil, fmt.Errorf("arithmetic update overflows int32")
		}
		return int32(sum), nil
	case int64:
		return value + delta, nil
	}
	return nil, fmt.Errorf("cannot apply arithmetic update to %T", current)
}

func (a *testMemoryAdapter) Delete(_ context.Context, tableName SchemaTableName, conditions []Where) (DatabaseResult, error) {
	a.mu.Lock()
	defer a.mu.Unlock()

	tbl := a.table(tableName)
	before := len(tbl.rows)
	tbl.rows = slices.DeleteFunc(tbl.rows, func(row map[string]any) bool {
		return testMatchesConditions(row, conditions)
	})
	return DatabaseResult{RowsAffected: int64(before - len(tbl.rows))}, nil
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

// testDerefPointer flattens pointer values produced by ToStorage so the
// in-memory adapter behaves like a database driver.
func testDerefPointer(v any) any {
	switch p := v.(type) {
	case *string:
		return testDereference(p)
	case *time.Time:
		return testDereference(p)
	case *int:
		return testDereference(p)
	case *int32:
		return testDereference(p)
	case *int64:
		return testDereference(p)
	default:
		return v
	}
}

func testDereference[T any](value *T) any {
	if value == nil {
		return nil
	}
	return *value
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
		return testIsNil(val)
	case OpIsNotNull:
		return !testIsNil(val)
	case OpEq, "":
		return testValuesEqual(val, c.Value)
	case OpNe:
		return !testValuesEqual(val, c.Value)
	case OpLt:
		order, ok := testCompareValues(val, c.Value)
		return ok && order < 0
	case OpLte:
		order, ok := testCompareValues(val, c.Value)
		return ok && order <= 0
	case OpGt:
		order, ok := testCompareValues(val, c.Value)
		return ok && order > 0
	case OpGte:
		order, ok := testCompareValues(val, c.Value)
		return ok && order >= 0
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
	if a == b {
		return true
	}
	// Named string types (e.g. InvitationStatus) must match plain strings.
	if as, ok := testStringValue(a); ok {
		bs, ok := testStringValue(b)
		return ok && as == bs
	}
	return false
}

func testStringValue(value any) (string, bool) {
	switch v := value.(type) {
	case string:
		return v, true
	default:
		rv := reflect.ValueOf(value)
		if rv.Kind() == reflect.String {
			return rv.String(), true
		}
		return "", false
	}
}

func testIsNil(value any) bool {
	if value == nil {
		return true
	}
	rv := reflect.ValueOf(value)
	switch rv.Kind() {
	case reflect.Pointer, reflect.Map, reflect.Slice, reflect.Interface, reflect.Func, reflect.Chan:
		return rv.IsNil()
	default:
		return false
	}
}

func testAsTime(value any) (time.Time, bool) {
	switch v := value.(type) {
	case time.Time:
		return v, true
	case *time.Time:
		if v == nil {
			return time.Time{}, false
		}
		return *v, true
	default:
		return time.Time{}, false
	}
}

func testCompareValues(a, b any) (int, bool) {
	if av, ok := testIntegerValue(a); ok {
		bv, ok := testIntegerValue(b)
		if !ok {
			return 0, false
		}
		return cmp.Compare(av, bv), true
	}

	if av, ok := testAsTime(a); ok {
		bv, ok := testAsTime(b)
		if !ok {
			return 0, false
		}
		return av.Compare(bv), true
	}

	switch av := a.(type) {
	case string:
		bv, ok := b.(string)
		if !ok {
			return 0, false
		}
		return cmp.Compare(av, bv), true
	default:
		return 0, false
	}
}

func testIntegerValue(value any) (int64, bool) {
	switch value := value.(type) {
	case int:
		return int64(value), true
	case int32:
		return int64(value), true
	case int64:
		return value, true
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
			order, ok := testCompareValues(a[ob.Column], b[ob.Column])
			if !ok || order == 0 {
				continue
			}
			if ob.Direction == OrderByDesc {
				return -order
			}
			return order
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
	return NewTestLimenWithSchema(t, nil, plugins...)
}

// NewTestLimenWithSchema is NewTestLimen with a custom schema configuration,
// for tests that need schema-level features such as public IDs.
func NewTestLimenWithSchema(t *testing.T, schema *SchemaConfig, plugins ...Plugin) (*Limen, *LimenCore) {
	t.Helper()

	l, err := New(&Config{
		BaseURL:  "http://localhost:8080",
		Database: newTestMemoryAdapter(t),
		Schema:   schema,
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

// SeedTestSession creates a session and returns its SessionResult.
// The user must already exist.
func SeedTestSession(t *testing.T, l *Limen, userID any, email string) *SessionResult {
	t.Helper()
	req := httptest.NewRequestWithContext(t.Context(), http.MethodPost, "/signin", http.NoBody)
	auth := &AuthenticationResult{User: &User{ID: userID, Email: email}}
	result, err := l.core.SessionManager.CreateSession(context.Background(), req, auth)
	if err != nil {
		t.Fatalf("SeedTestSession: %v", err)
	}
	return result
}

// SeedTestSessionRecord creates a session and returns the stored *Session.
func SeedTestSessionRecord(t *testing.T, l *Limen, userID any, email string) *Session {
	t.Helper()

	result := SeedTestSession(t, l, userID, email)
	if result.Token == "" {
		t.Fatal("SeedTestSessionRecord: empty session token")
	}

	sessions, err := l.ListSessions(t.Context(), userID)
	if err != nil {
		t.Fatalf("SeedTestSessionRecord: %v", err)
	}
	for i := range sessions {
		if sessions[i].Token == result.Token {
			return &sessions[i]
		}
	}
	t.Fatalf("SeedTestSessionRecord: session for token %q not found", result.Token)
	return nil
}
