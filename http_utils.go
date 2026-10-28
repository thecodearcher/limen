package limen

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

type bodyContextKey struct{}

// normalizePath normalizes the base path to start with a slash.
func normalizePath(basePath string) string {
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}

	basePath = strings.TrimSuffix(basePath, "/")
	return basePath
}

// GetBody extracts the parsed JSON body from the request context
func GetJSONBody(req *http.Request) map[string]any {
	if req == nil {
		return nil
	}

	if body, ok := req.Context().Value(bodyContextKey{}).(map[string]any); ok {
		return body
	}

	return nil
}

// queryToMap groups one level of bracketed keys: filter[plan]=pro becomes {"filter": {"plan": "pro"}}.
func queryToMap(r *http.Request) map[string]any {
	q := r.URL.Query()
	out := make(map[string]any, len(q))
	for key, values := range q {
		if _, _, ok := splitBracketKey(key); !ok {
			out[key] = queryValue(values)
		}
	}

	for key, values := range q {
		name, sub, ok := splitBracketKey(key)
		if !ok {
			continue
		}
		if _, flat := q[name]; flat {
			out[key] = queryValue(values)
			continue
		}
		group, _ := out[name].(map[string]any)
		if group == nil {
			group = make(map[string]any)
			out[name] = group
		}
		group[sub] = queryValue(values)
	}
	return out
}

func queryValue(values []string) any {
	if len(values) == 1 {
		return values[0]
	}
	return values
}

func splitBracketKey(key string) (name, sub string, ok bool) {
	name, rest, found := strings.Cut(key, "[")
	if !found || name == "" {
		return "", "", false
	}
	sub, found = strings.CutSuffix(rest, "]")
	if !found || sub == "" || strings.ContainsAny(sub, "[]") {
		return "", "", false
	}
	return name, sub, true
}

// shouldParseBody checks if the request body should be parsed as JSON
func shouldParseBody(req *http.Request) bool {
	if req.Method != "POST" && req.Method != "PUT" && req.Method != "PATCH" {
		return false
	}
	contentType := req.Header.Get("Content-Type")
	return strings.HasPrefix(contentType, "application/json") && req.Body != nil
}

// parseJSONBody reads and parses the JSON body from the request
// Returns the parsed body map and the original body bytes for restoration
func parseJSONBody(req *http.Request) (map[string]any, []byte, error) {
	defer req.Body.Close()
	bodyBytes, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, nil, err
	}

	var body map[string]any
	if err := json.Unmarshal(bodyBytes, &body); err != nil {
		return nil, bodyBytes, err
	}

	return body, bodyBytes, nil
}

// parseAndStoreBody parses the JSON body if needed and stores it in context.
func parseAndStoreBody(req *http.Request) *http.Request {
	if GetJSONBody(req) != nil {
		return req
	}

	if !shouldParseBody(req) {
		return req
	}

	body, bodyBytes, err := parseJSONBody(req)
	if err != nil {
		return req
	}

	req = req.WithContext(context.WithValue(req.Context(), bodyContextKey{}, body))

	// Restore body for handlers that need to read it
	req.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))

	return req
}

func getCurrentRouteFromContext(ctx context.Context) *route {
	if route, ok := ctx.Value(currentRouteContextKey{}).(*route); ok {
		return route
	}
	return nil
}

type paginationOptions struct {
	sortSchema     Schema
	sortableFields []SchemaField
}

type PaginationOption func(*paginationOptions)

// WithSortableFields allows sort_by to name one of fields, resolved against schema.
func WithSortableFields(schema Schema, fields ...SchemaField) PaginationOption {
	return func(o *paginationOptions) {
		o.sortSchema = schema
		o.sortableFields = fields
	}
}

// ParsePagination reads page, per_page, sort_by and sort_direction from the request
// query, applying defaults and clamping per_page to MaxPerPage. sort_by is ignored
// unless it names a field allowed by WithSortableFields.
func ParsePagination(r *http.Request, opts ...PaginationOption) *QueryOptions {
	q := r.URL.Query()

	perPage := min(queryInt(q.Get("per_page"), DefaultPerPage), MaxPerPage)
	page := queryInt(q.Get("page"), DefaultPage)

	queryOptions := &QueryOptions{
		Limit:  perPage,
		Offset: (page - 1) * perPage,
	}

	var options paginationOptions
	for _, opt := range opts {
		opt(&options)
	}

	sortBy := SchemaField(strings.TrimSpace(q.Get("sort_by")))
	if !slices.Contains(options.sortableFields, sortBy) {
		return queryOptions
	}

	column := options.sortSchema.GetField(sortBy)
	if column == "" {
		return queryOptions
	}

	direction := OrderByAsc
	if strings.EqualFold(strings.TrimSpace(q.Get("sort_direction")), string(OrderByDesc)) {
		direction = OrderByDesc
	}
	queryOptions.OrderBy = []OrderBy{{Column: column, Direction: direction}}

	return queryOptions
}

func queryInt(raw string, fallback int) int {
	if n, err := strconv.Atoi(strings.TrimSpace(raw)); err == nil && n > 0 {
		return n
	}
	return fallback
}
