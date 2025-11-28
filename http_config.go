package aegis

import (
	"net/http"

	"github.com/thecodearcher/aegis/pkg/httpx"
)

type HTTPConfigOption func(*HTTPConfig)

type HTTPConfig struct {
	// Global middleware
	middleware []httpx.Middleware
	// The base path where all the routes will be mounted
	basePath string
	// overrides for specific plugins
	overrides map[string]*PluginHTTPOverride
	// Paths to be disabled by their ID or pattern
	disabledPaths []string
	// Response envelope configuration
	responseEnvelope *responseEnvelopeConfig
	// SessionTransformer customizes the session response payload before it's sent to the client.
	// Returns a map[string]any for the response body, or an AegisError to handle an error condition.
	sessionTransformer SessionTransformer
	// HTTPHooks are functions that are called before and after the request is processed
	hooks *httpx.Hooks
}

type EnvelopeSerializer func(
	w http.ResponseWriter,
	r *http.Request,
	status int,
	rawBody []byte,
	err *AegisError,
) error

// SessionTransformer is a function to serialize the session data to a map[string]any for the response body
type SessionTransformer func(user map[string]any, pendingActions []PendingAction, sessionResult *SessionResult) (map[string]any, *AegisError)

type EnvelopeFields struct {
	Data    string
	Message string
}

type responseEnvelopeConfig struct {
	mode       EnvelopeMode
	fields     EnvelopeFields
	serializer EnvelopeSerializer
}

type PluginHTTPOverride struct {
	BasePath string
	// Middleware to be applied to the plugin's routes
	Middleware []httpx.Middleware
}

func NewDefaultHTTPConfig(opts ...HTTPConfigOption) *HTTPConfig {
	config := &HTTPConfig{
		middleware:    []httpx.Middleware{},
		basePath:      "/auth",
		overrides:     map[string]*PluginHTTPOverride{},
		disabledPaths: []string{},
		responseEnvelope: &responseEnvelopeConfig{
			mode: EnvelopeOff,
		},
	}
	for _, opt := range opts {
		opt(config)
	}
	return config
}

func WithHTTPBasePath(basePath string) HTTPConfigOption {
	return func(c *HTTPConfig) {
		c.basePath = basePath
	}
}

func WithHTTPMiddleware(globalMW ...httpx.Middleware) HTTPConfigOption {
	return func(c *HTTPConfig) {
		c.middleware = append(c.middleware, globalMW...)
	}
}

func WithHTTPOverrides(overrides map[string]*PluginHTTPOverride) HTTPConfigOption {
	return func(c *HTTPConfig) {
		c.overrides = overrides
	}
}

// WithHTTPDisabledPaths adds paths to be disabled by their ID or pattern
func WithHTTPDisabledPaths(disabledPaths []string) HTTPConfigOption {
	return func(c *HTTPConfig) {
		c.disabledPaths = disabledPaths
	}
}

func WithHTTPResponseEnvelopeMode(mode EnvelopeMode) HTTPConfigOption {
	return func(c *HTTPConfig) {
		c.responseEnvelope.mode = mode
	}
}

func WithHTTPResponseEnvelopeFields(fields EnvelopeFields) HTTPConfigOption {
	return func(c *HTTPConfig) {
		c.responseEnvelope.fields = fields
	}
}

func WithHTTPResponseEnvelopeSerializer(serializer EnvelopeSerializer) HTTPConfigOption {
	return func(c *HTTPConfig) {
		c.responseEnvelope.serializer = serializer
	}
}

func WithHTTPSessionTransformer(transformer SessionTransformer) HTTPConfigOption {
	return func(c *HTTPConfig) {
		c.sessionTransformer = transformer
	}
}

func WithHTTPHooks(hooks *httpx.Hooks) HTTPConfigOption {
	return func(c *HTTPConfig) {
		c.hooks = hooks
	}
}
