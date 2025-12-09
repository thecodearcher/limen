package aegis

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"regexp"
	"slices"
	"strings"

	"github.com/thecodearcher/aegis/pkg/httpx"
)

// generateCryptoSecureRandomString generates a cryptographically secure random string
func generateCryptoSecureRandomString() (string, error) {
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
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
	normalizedPath := httpx.NormalizePath(req.URL.Path)
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

func sliceToMap[T any](slice []T, fn func(T) string) map[string]T {
	if len(slice) == 0 {
		return make(map[string]T)
	}
	m := make(map[string]T, len(slice))
	for _, item := range slice {
		m[fn(item)] = item
	}
	return m
}
