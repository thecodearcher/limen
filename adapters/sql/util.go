package sql

import (
	"sort"
	"time"
)

// timeLayouts are tried in order when normalizing driver output to time.Time (most precise first).
var timeLayouts = []string{
	time.RFC3339Nano,
	time.RFC3339,
	"2006-01-02 15:04:05.999999",
	"2006-01-02 15:04:05.999",
	"2006-01-02 15:04:05",
	"2006-01-02 15:04:05.999999 -0700",
	"2006-01-02 15:04:05 -0700",
	"2006-01-02",
}

// normalizeRow converts driver-specific types in place: []byte → string or time.Time (if it parses as a datetime).
// Some drivers (e.g. MySQL text protocol) return []byte for string/date columns; limen expects string or time.Time.
func normalizeRow(m map[string]any) {
	for k, v := range m {
		switch x := v.(type) {
		case []byte:
			if t := parseTimeBytes(x); !t.IsZero() {
				m[k] = t
			} else {
				m[k] = string(x)
			}
		case string:
			if t := parseTimeString(x); !t.IsZero() {
				m[k] = t
			}
		}
	}
}

func parseTimeBytes(b []byte) time.Time {
	if len(b) == 0 {
		return time.Time{}
	}
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, string(b)); err == nil {
			return t
		}
	}
	return time.Time{}
}

func parseTimeString(s string) time.Time {
	if s == "" {
		return time.Time{}
	}
	for _, layout := range timeLayouts {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}

// sortedKeys returns map keys sorted so INSERT/UPDATE column order is deterministic.
func sortedKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
