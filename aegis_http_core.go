package aegis

import (
	"regexp"
)

type AegisHTTPCore struct {
	Responder              *Responder
	core                   *AegisCore
	authInstance           *Aegis
	config                 *httpConfig
	trustedOriginsPatterns []*regexp.Regexp
}

// Cookies returns the shared CookieManager for cookie operations.
func (httpCore *AegisHTTPCore) Cookies() *CookieManager {
	return httpCore.core.Cookies()
}

func (httpCore *AegisHTTPCore) SessionCookieName() string {
	if httpCore.config.cookieConfig == nil {
		return ""
	}
	return httpCore.config.cookieConfig.sessionCookieName
}

func (httpCore *AegisHTTPCore) IsTrustedOrigin(url string) bool {
	for _, pattern := range httpCore.trustedOriginsPatterns {
		if pattern.MatchString(url) {
			return true
		}
	}
	return false
}
