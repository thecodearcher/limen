package aegis

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"net/url"
	"os"
	"path"
	reflect "reflect"
	"regexp"
	"slices"
	"strings"
	"time"
)

type CharSetType int

const (
	CharSetAlphanumeric CharSetType = iota
	CharSetNumeric
)

var (
	alphanumericChars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ1234567890"
	numericChars      = "0123456789"
)

// generateCryptoSecureRandomString generates a cryptographically secure random string
func generateCryptoSecureRandomString() string {
	bytes := make([]byte, 32)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func GenerateRandomString(length int, charSetType ...CharSetType) string {
	chars := alphanumericChars
	if len(charSetType) > 0 && charSetType[0] == CharSetNumeric {
		chars = numericChars
	}
	charCount := len(chars)
	expectedBytes := make([]byte, length)

	rand.Read(expectedBytes)
	for i := range length {
		expectedBytes[i] = chars[int(expectedBytes[i])%charCount]
	}
	return string(expectedBytes)
}

func ipExtractorFromRemoteAddr(request *http.Request) string {
	if ip := request.Header.Get("X-Forwarded-For"); ip != "" {
		return ip
	}
	if ip := request.Header.Get("X-Real-IP"); ip != "" {
		return ip
	}
	ip, _, _ := net.SplitHostPort(request.RemoteAddr)
	return ip
}

// compilePattern compiles a glob pattern to a regex
// Returns the compiled regex and an error if compilation fails
func compilePattern(pattern string) (*regexp.Regexp, error) {
	regexPattern := globToRegex(pattern)
	return regexp.Compile(regexPattern)
}

// globToRegex converts a glob pattern to a regex pattern
func globToRegex(pattern string) string {
	var result strings.Builder
	result.WriteString("^")

	runes := []rune(pattern)
	i := 0

	for i < len(runes) {
		char := runes[i]

		switch char {
		case '*':
			// Check if it's **
			if i+1 < len(runes) && runes[i+1] == '*' {
				// ** matches zero or more characters including /
				result.WriteString(".*")
				i += 2 // Skip both stars
				continue
			} else {
				// Single * matches any sequence except /
				result.WriteString("[^/]*")
			}

		case '?':
			// ? matches any single character except /
			result.WriteString("[^/]")

		case ':':
			// Route parameter (:param) - match one path segment. Skip param name until next / or end.
			result.WriteString("[^/]+")
			i++
			for i < len(runes) && runes[i] != '/' {
				i++
			}
			continue

		case '[':
			// Character class - copy until closing ]
			result.WriteRune('[')
			i++
			for i < len(runes) && runes[i] != ']' {
				result.WriteRune(runes[i])
				i++
			}
			if i < len(runes) {
				result.WriteRune(']')
			}

		case '\\':
			// Escape character - escape the next character
			if i+1 < len(runes) {
				result.WriteRune('\\')
				result.WriteRune(runes[i+1])
				i += 2
				continue
			}
			result.WriteRune('\\')

		case '.', '+', '(', ')', '|', '{', '}', '^', '$':
			// Escape regex special characters
			result.WriteRune('\\')
			result.WriteRune(char)

		default:
			result.WriteRune(char)
		}

		i++
	}

	result.WriteString("$")
	return result.String()
}

// sortRulesBySpecificity sorts the rules by specificity.
//
// The rules are sorted by the following criteria:
//  1. Patterns without wildcards are more specific
//  2. Longer paths are more specific
//  3. Patterns with fewer wildcards are more specific
func sortRulesBySpecificity(rules []*RateLimitRule) {
	slices.SortFunc(rules, func(a *RateLimitRule, b *RateLimitRule) int {
		pathA, pathB := a.path, b.path

		// Exact matches first
		if !containsWildcard(pathA) && containsWildcard(pathB) {
			return -1
		}
		if containsWildcard(pathA) && !containsWildcard(pathB) {
			return 1
		}

		// Longer paths are more specific
		if len(pathA) != len(pathB) {
			return len(pathB) - len(pathA)
		}

		// Count wildcards (fewer is more specific)
		wildcardsA := countWildcards(pathA)
		wildcardsB := countWildcards(pathB)
		return wildcardsA - wildcardsB
	})
}

func containsWildcard(path string) bool {
	return strings.Contains(path, "*") || strings.Contains(path, "?")
}

func countWildcards(path string) int {
	count := 0
	for _, char := range path {
		if char == '*' || char == '?' {
			count++
		}
	}
	return count
}

func pathMatcher(req *http.Request, pathRegex *regexp.Regexp) bool {
	normalizedPath := NormalizePath(req.URL.Path)
	return pathRegex.MatchString(normalizedPath)
}

func originMatcher(request *http.Request, origins []*regexp.Regexp) bool {
	requestOrigin := request.Header.Get("Origin")
	referer := request.Header.Get("Referer")
	if requestOrigin == "" && referer == "" {
		return false
	}
	if requestOrigin == "" {
		refererURL, err := url.Parse(referer)
		if err != nil {
			return false
		}
		requestOrigin = refererURL.Scheme + "://" + refererURL.Host
	}

	for _, pattern := range origins {
		if pattern.MatchString(requestOrigin) {
			return true
		}
	}

	return false
}

func writeToFile(data []byte, outputPath string) error {
	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, bytes.NewReader(data)); err != nil {
		return fmt.Errorf("failed to write schemas file: %w", err)
	}

	return nil
}

func addTimestampFields(fields []ColumnDefinition) []ColumnDefinition {
	return append(fields, ColumnDefinition{
		Name:         string(SchemaCreatedAtField),
		LogicalField: SchemaCreatedAtField,
		Type:         ColumnTypeTime,
		IsNullable:   false,
		IsPrimaryKey: false,
		DefaultValue: string(DatabaseDefaultValueNow),
		Tags: map[string]string{
			"json": "created_at",
		},
	}, ColumnDefinition{
		Name:         string(SchemaUpdatedAtField),
		LogicalField: SchemaUpdatedAtField,
		Type:         ColumnTypeTime,
		IsNullable:   false,
		IsPrimaryKey: false,
		Tags: map[string]string{
			"json": "updated_at",
		},
	})
}

func addSoftDeleteField(fields []ColumnDefinition, config *SchemaConfig, schemaName SchemaName) []ColumnDefinition {
	softDeleteField := config.getCoreSchemaCustomizationField(schemaName, SchemaSoftDeleteField)
	if softDeleteField != "" {
		return append(fields, ColumnDefinition{
			Name:         softDeleteField,
			LogicalField: SchemaSoftDeleteField,
			Type:         ColumnTypeTime,
			IsNullable:   true,
			IsPrimaryKey: false,
			Tags: map[string]string{
				"json": softDeleteField,
			},
		})
	}
	return fields
}

// IsValidCoreSchema checks if a string is a valid core schema name
func IsValidCoreSchema(name string) bool {
	switch SchemaName(name) {
	case CoreSchemaUsers, CoreSchemaSessions, CoreSchemaVerifications, CoreSchemaRateLimits, CoreSchemaAccounts:
		return true
	default:
		return false
	}
}

func getNullableValue[T any](value any) *T {
	if value == nil {
		return nil
	}
	v := value.(T)
	return &v
}

func getString(v any) string {
	if v == nil {
		return ""
	}
	s, _ := v.(string)
	return s
}

func getTime(v any) time.Time {
	if v == nil {
		return time.Time{}
	}
	t, _ := v.(time.Time)
	return t
}

func GetAdditionalFieldsFromRequest(response http.ResponseWriter, request *http.Request, schema Schema) (map[string]any, error) {
	if schema.GetAdditionalFields() != nil {
		additionalFieldsContext := newAdditionalFieldsContext(request, response)
		return schema.GetAdditionalFields()(additionalFieldsContext)
	}
	return make(map[string]any), nil
}

func joinCustomStringSlice[T ~string](fields []T, separator string) string {
	var joined strings.Builder
	for i := range fields {
		joined.WriteString(string(fields[i]))
		if i < len(fields)-1 {
			joined.WriteString(separator)
		}
	}
	return joined.String()
}

func compileTrustedOrigins(origins ...string) []*regexp.Regexp {
	patterns := make([]*regexp.Regexp, 0, len(origins))
	for _, pattern := range origins {
		normalizedPattern := pattern
		if !strings.Contains(pattern, "://") {
			normalizedPattern = "*://" + pattern
		}
		regexPattern := globToRegex(normalizedPattern)
		re, err := regexp.Compile(regexPattern)
		if err != nil {
			log.Panicf("failed to compile pattern for trusted origin %s: %v", pattern, err)
		}
		patterns = append(patterns, re)
	}
	return patterns
}

func processCustomRateLimitRules(basePath string, customRules map[string]*RateLimitRule) map[string]*RateLimitRule {
	rules := make(map[string]*RateLimitRule)

	for pattern, rule := range customRules {
		completePath := path.Join(basePath, pattern)

		if err := compileAndSetRulePattern(rule, completePath); err != nil {
			log.Panicf("failed to compile pattern for path %s: %v", completePath, err)
		}

		rules[completePath] = rule
	}

	return rules
}

// compileAndSetRulePattern compiles the pattern and sets it on the rule
func compileAndSetRulePattern(rule *RateLimitRule, completePath string) error {
	compiledPattern, err := compilePattern(completePath)
	if err != nil {
		return fmt.Errorf("failed to compile pattern: %w", err)
	}

	rule.path = completePath
	rule.pathRegex = compiledPattern
	return nil
}

func resolveRuleOverride(rule *RateLimitRule, customRules map[string]*RateLimitRule) *RateLimitRule {
	if customRule, exists := customRules[rule.path]; exists {
		delete(customRules, rule.path)
		return customRule
	}
	return rule
}

func normalizePluginPath(basePath string, pluginBasePath string, override *PluginHTTPOverride) string {
	if override != nil && override.BasePath != "" {
		pluginBasePath = override.BasePath
	}

	return path.Join(basePath, NormalizePath(pluginBasePath))
}

func isCoreSchema(schema Schema) bool {
	switch schema.(type) {
	case *UserSchema, *VerificationSchema, *SessionSchema, *RateLimitSchema, *AccountSchema:
		return true
	}

	return embedsCoreSchema(schema)
}

func embedsCoreSchema(schema Schema) bool {
	sType := reflect.TypeOf(schema)
	if sType == nil || sType.Kind() != reflect.Pointer {
		return false
	}
	sType = sType.Elem()
	if sType.Kind() != reflect.Struct {
		return false
	}

	coreTypes := map[reflect.Type]bool{
		reflect.TypeFor[UserSchema]():         true,
		reflect.TypeFor[VerificationSchema](): true,
		reflect.TypeFor[SessionSchema]():      true,
		reflect.TypeFor[RateLimitSchema]():    true,
		reflect.TypeFor[AccountSchema]():      true,
	}

	for i := 0; i < sType.NumField(); i++ {
		field := sType.Field(i)
		if field.Anonymous {
			fieldType := field.Type
			if fieldType.Kind() == reflect.Pointer {
				fieldType = fieldType.Elem()
			}
			if coreTypes[fieldType] {
				return true
			}
		}
	}
	return false
}

func GetFromMap[T any](m map[string]any, key string) T {
	var result T
	if m == nil {
		return result
	}

	if value, ok := m[key].(T); ok {
		result = value
	}
	return result
}

// ExtractCookieValue extracts the value of a cookie from the Set-Cookie headers.
// Returns empty string if the cookie is not found.
func ExtractCookieValue(headers http.Header, cookieName string) string {
	prefix := cookieName + "="
	for _, cookie := range headers.Values("Set-Cookie") {
		cookieValue := strings.Split(cookie, ";")[0]
		if strings.HasPrefix(cookieValue, prefix) {
			return cookieValue[len(prefix):]
		}
	}
	return ""
}

// simple wrapper around url.JoinPath that returns an empty string if the join fails
func joinURL(baseURL string, path ...string) string {
	url, err := url.JoinPath(baseURL, path...)
	if err != nil {
		return ""
	}
	return url
}
