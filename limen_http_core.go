package limen

import (
	"regexp"
)

type LimenHTTPCore struct {
	Responder              *Responder
	core                   *LimenCore
	authInstance           *Limen
	config                 *httpConfig
	trustedOriginsPatterns []*regexp.Regexp
}

// Cookies returns the shared CookieManager for cookie operations.
func (httpCore *LimenHTTPCore) Cookies() *CookieManager {
	return httpCore.core.Cookies()
}

func (httpCore *LimenHTTPCore) SessionCookieName() string {
	if httpCore.config.cookieConfig == nil {
		return ""
	}
	return httpCore.config.cookieConfig.sessionCookieName
}

func (httpCore *LimenHTTPCore) IsTrustedOrigin(url string) bool {
	for _, pattern := range httpCore.trustedOriginsPatterns {
		if pattern.MatchString(url) {
			return true
		}
	}
	return false
}
