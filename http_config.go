package aegis

import (
	"net/http"

	"github.com/thecodearcher/aegis/pkg/httpx"
)

type HTTPConfigOption func(*httpConfig)

type httpConfig struct {
	// Global middleware
	middleware []httpx.Middleware
	// The base path where all the routes will be mounted
	basePath string
	// overrides for specific plugins
	overrides map[string]*PluginHTTPOverride
	// Paths to be disabled by their ID
	disabledPathIDs []string
	// Response envelope configuration
	responseEnvelope *responseEnvelopeConfig
}

type EnvelopeFormatter func(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	rawBody []byte,
	err AegisError,
) error

type EnvelopeFields struct {
	Data    string
	Message string
}

type responseEnvelopeConfig struct {
	mode      EnvelopeMode
	fields    EnvelopeFields
	formatter EnvelopeFormatter
}

type PluginHTTPOverride struct {
	BasePath string
	// Middleware to be applied to the plugin's routes
	Middleware []httpx.Middleware
}

func WithHTTPBasePath(basePath string) HTTPConfigOption {
	return func(c *httpConfig) {
		c.basePath = basePath
	}
}

func WithHTTPMiddleware(globalMW ...httpx.Middleware) HTTPConfigOption {
	return func(c *httpConfig) {
		c.middleware = append(c.middleware, globalMW...)
	}
}

func WithHTTPOverrides(overrides map[string]*PluginHTTPOverride) HTTPConfigOption {
	return func(c *httpConfig) {
		c.overrides = overrides
	}
}

func WithHTTPDisabledPathIDs(disabledPathIDs []string) HTTPConfigOption {
	return func(c *httpConfig) {
		c.disabledPathIDs = disabledPathIDs
	}
}

func WithHTTPResponseEnvelopeMode(mode EnvelopeMode) HTTPConfigOption {
	return func(c *httpConfig) {
		if c.responseEnvelope == nil {
			c.responseEnvelope = &responseEnvelopeConfig{}
		}
		c.responseEnvelope.mode = mode
	}
}

func WithHTTPResponseEnvelopeFields(fields EnvelopeFields) HTTPConfigOption {
	return func(c *httpConfig) {
		if c.responseEnvelope == nil {
			c.responseEnvelope = &responseEnvelopeConfig{
				mode: EnvelopeAlways,
			}
		}
		c.responseEnvelope.fields = fields
	}
}

func WithHTTPResponseEnvelopeFormatter(formatter EnvelopeFormatter) HTTPConfigOption {
	return func(c *httpConfig) {
		if c.responseEnvelope == nil {
			c.responseEnvelope = &responseEnvelopeConfig{}
		}
		c.responseEnvelope.formatter = formatter
	}
}
