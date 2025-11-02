package httpx

import (
	"strings"
)

// NormalizeBasePath normalizes the base path to start with a slash
func NormalizeBasePath(basePath string) string {
	if !strings.HasPrefix(basePath, "/") {
		basePath = "/" + basePath
	}
	return basePath
}
