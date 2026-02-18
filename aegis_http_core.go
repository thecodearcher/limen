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

func (httpCore *AegisHTTPCore) SessionCookieName() string {
	if httpCore.config.cookieConfig == nil {
		return ""
	}
	return httpCore.config.cookieConfig.name
}

func (httpCore *AegisHTTPCore) IsTrustedOrigin(url string) bool {
	for _, pattern := range httpCore.trustedOriginsPatterns {
		if pattern.MatchString(url) {
			return true
		}
	}
	return false
}
