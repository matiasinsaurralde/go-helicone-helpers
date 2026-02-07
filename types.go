package logger

// ILogRequest represents a basic log request
type ILogRequest struct {
	Model string                 `json:"model"`
	Extra map[string]interface{} `json:"-"`
}

// HeliconeEventTool represents a tool event
type HeliconeEventTool struct {
	Type     string                 `json:"_type"`
	ToolName string                 `json:"toolName"`
	Input    interface{}            `json:"input"`
	Extra    map[string]interface{} `json:"-"`
}

// VectorDBOperation represents the allowed operations for vector DB events
type VectorDBOperation string

const (
	VectorDBOperationSearch VectorDBOperation = "search"
	VectorDBOperationInsert VectorDBOperation = "insert"
	VectorDBOperationDelete VectorDBOperation = "delete"
	VectorDBOperationUpdate VectorDBOperation = "update"
)

// HeliconeEventVectorDB represents a vector database event
type HeliconeEventVectorDB struct {
	Type         string                 `json:"_type"`
	Operation    VectorDBOperation      `json:"operation"`
	Text         string                 `json:"text,omitempty"`
	Vector       []float64              `json:"vector,omitempty"`
	TopK         int                    `json:"topK,omitempty"`
	Filter       interface{}            `json:"filter,omitempty"`
	DatabaseName string                 `json:"databaseName,omitempty"`
	Extra        map[string]interface{} `json:"-"`
}

// HeliconeLogRequest represents either a basic log request or a custom event request
type HeliconeLogRequest interface{}

// IsToolEvent checks if the request is a tool event
func IsToolEvent(req HeliconeLogRequest) bool {
	_, ok := req.(HeliconeEventTool)
	return ok
}

// IsVectorDBEvent checks if the request is a vector DB event
func IsVectorDBEvent(req HeliconeLogRequest) bool {
	_, ok := req.(HeliconeEventVectorDB)
	return ok
}

// IsBasicLogRequest checks if the request is a basic log request
func IsBasicLogRequest(req HeliconeLogRequest) bool {
	_, ok := req.(ILogRequest)
	return ok
}

// ProviderRequest represents the provider request structure
type ProviderRequest struct {
	URL  string                 `json:"url"`
	JSON map[string]interface{} `json:"json"`
	Meta map[string]interface{} `json:"meta"`
}

// ProviderResponse represents the provider response structure
type ProviderResponse struct {
	Headers  map[string]string      `json:"headers"`
	Status   int                    `json:"status"`
	JSON     map[string]interface{} `json:"json,omitempty"`
	TextBody string                 `json:"textBody,omitempty"`
}

// Timing represents the timing information for the request
type Timing struct {
	StartTime struct {
		Seconds      int `json:"seconds"`
		Milliseconds int `json:"milliseconds"`
	} `json:"startTime"`
	EndTime struct {
		Seconds      int `json:"seconds"`
		Milliseconds int `json:"milliseconds"`
	} `json:"endTime"`
	TimeToFirstToken *int `json:"timeToFirstToken,omitempty"`
}

// Provider identifies the Helicone logging backend. It is used to select the
// correct API path (/oai/v1/log, /anthropic/v1/log, or /custom/v1/log).
type Provider string

const (
	ProviderOpenAI     Provider = "openai"
	ProviderAnthropic  Provider = "anthropic"
	ProviderCustom     Provider = "custom"
)

// LoggerOptions represents the configuration options for the logger.
// LoggingEndpoint is the base URL (e.g. "https://api.worker.helicone.ai"); the
// path is chosen from Provider. Leave LoggingEndpoint empty to use the default.
// Provider is the default provider for all logs from this logger; leave nil for custom.
type LoggerOptions struct {
	APIKey          string
	Headers         map[string]string
	LoggingEndpoint string
	Provider        *Provider
}

// LogOptions represents options for logging a request.
// Provider overrides the logger's default for this call; nil uses the logger default or custom.
type LogOptions struct {
	StartTime         int64
	EndTime           int64
	AdditionalHeaders map[string]string
	TimeToFirstToken  *int
	Status            int
	Provider          *Provider
}
