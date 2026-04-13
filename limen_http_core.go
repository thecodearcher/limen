package limen

import (
	"fmt"
	"net/url"
	"regexp"
)

type LimenHTTPCore struct {
	Responder              *Responder
	core                   *LimenCore
	authInstance           *Limen
	config                 *httpConfig
	trustedOriginsPatterns []*regexp.Regexp
}

func (httpCore *LimenHTTPCore) SessionCookieName() string {
	if httpCore.config.cookieConfig == nil {
		return ""
	}
	return httpCore.config.cookieConfig.sessionCookieName
}

func (httpCore *LimenHTTPCore) IsTrustedOrigin(urlStr string) bool {
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return false
	}

	normalizedURL := fmt.Sprintf("%s://%s", parsedURL.Scheme, parsedURL.Host)

	if normalizedURL == httpCore.core.GetBaseURL() {
		return true
	}

	for _, pattern := range httpCore.trustedOriginsPatterns {
		if pattern.MatchString(normalizedURL) {
			return true
		}
	}
	return false
}
