package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultLoggingBaseURL = "https://api.worker.helicone.ai"

// ManualLogger represents the Helicone manual logger
type ManualLogger struct {
	apiKey           string
	headers          map[string]string
	loggingBaseURL   string
	defaultProvider  *Provider
}

// getLoggingEndpoint returns the full URL for the given provider.
// If provider is nil or ProviderCustom, returns the custom log path.
func (l *ManualLogger) getLoggingEndpoint(provider *Provider) string {
	p := provider
	if p == nil {
		p = l.defaultProvider
	}
	if p == nil || *p == ProviderCustom {
		return l.loggingBaseURL + "/custom/v1/log"
	}
	switch *p {
	case ProviderOpenAI:
		return l.loggingBaseURL + "/oai/v1/log"
	case ProviderAnthropic:
		return l.loggingBaseURL + "/anthropic/v1/log"
	default:
		return l.loggingBaseURL + "/custom/v1/log"
	}
}

// New creates a new instance of ManualLogger.
// LoggingEndpoint is the base URL (e.g. "https://api.worker.helicone.ai"); leave empty for default.
func New(opts LoggerOptions) *ManualLogger {
	baseURL := opts.LoggingEndpoint
	if baseURL == "" {
		baseURL = defaultLoggingBaseURL
	}
	return &ManualLogger{
		apiKey:          opts.APIKey,
		headers:         opts.Headers,
		loggingBaseURL:  strings.TrimSuffix(baseURL, "/"),
		defaultProvider: opts.Provider,
	}
}

// LogRequest logs a custom request to Helicone
func (l *ManualLogger) LogRequest(request HeliconeLogRequest, operation func(*ResultRecorder) (any, error), additionalHeaders map[string]string) (any, error) {
	startTime := time.Now().UnixMilli()
	recorder := NewResultRecorder(l, request)

	result, err := operation(recorder)
	if err != nil {
		return nil, err
	}

	endTime := time.Now().UnixMilli()

	err = l.SendLog(request, recorder.GetResults(), LogOptions{
		StartTime:         startTime,
		EndTime:           endTime,
		AdditionalHeaders: additionalHeaders,
		Status:            200,
	})
	if err != nil {
		return nil, fmt.Errorf("error sending log: %w", err)
	}

	return result, nil
}

// SendLog sends a log to Helicone
func (l *ManualLogger) SendLog(request HeliconeLogRequest, response interface{}, options LogOptions) error {
	providerRequest := ProviderRequest{
		URL:  "custom-model-nopath",
		Meta: make(map[string]interface{}),
	}

	// Handle request based on its type
	switch v := request.(type) {
	case ILogRequest:
		providerRequest.JSON = map[string]interface{}{
			"model": v.Model,
		}
		if v.Extra != nil {
			for k, val := range v.Extra {
				providerRequest.JSON[k] = val
			}
		}
	case HeliconeEventTool:
		providerRequest.JSON = map[string]interface{}{
			"_type":    "tool",
			"toolName": v.ToolName,
			"input":    v.Input,
		}
		if v.Extra != nil {
			for k, val := range v.Extra {
				providerRequest.JSON[k] = val
			}
		}
	case HeliconeEventVectorDB:
		providerRequest.JSON = map[string]interface{}{
			"_type":        "vector_db",
			"operation":    v.Operation,
			"text":         v.Text,
			"vector":       v.Vector,
			"topK":         v.TopK,
			"filter":       v.Filter,
			"databaseName": v.DatabaseName,
		}
		if v.Extra != nil {
			for k, val := range v.Extra {
				providerRequest.JSON[k] = val
			}
		}
	default:
		return fmt.Errorf("unsupported request type: %T", request)
	}

	providerResponse := ProviderResponse{
		Headers: l.headers,
		Status:  options.Status,
	}

	// Handle response based on its type
	switch v := response.(type) {
	case map[string]interface{}:
		providerResponse.JSON = v
		if IsToolEvent(request) {
			if toolEvent, ok := request.(HeliconeEventTool); ok {
				providerResponse.JSON["_type"] = toolEvent.Type
				providerResponse.JSON["toolName"] = toolEvent.ToolName
			}
		}
	case string:
		providerResponse.TextBody = v
	default:
		return fmt.Errorf("unsupported response type: %T", response)
	}

	timing := Timing{
		TimeToFirstToken: options.TimeToFirstToken,
	}
	timing.StartTime.Seconds = int(options.StartTime / 1000)
	timing.StartTime.Milliseconds = int(options.StartTime % 1000)
	timing.EndTime.Seconds = int(options.EndTime / 1000)
	timing.EndTime.Milliseconds = int(options.EndTime % 1000)

	payload := map[string]interface{}{
		"providerRequest":  providerRequest,
		"providerResponse": providerResponse,
		"timing":           timing,
	}

	jsonData, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("error marshaling payload: %w", err)
	}

	endpoint := l.getLoggingEndpoint(options.Provider)
	req, err := http.NewRequest("POST", endpoint, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("error creating request: %w", err)
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", l.apiKey))
	req.Header.Set("Content-Type", "application/json")

	// Add additional headers
	for k, v := range options.AdditionalHeaders {
		req.Header.Set(k, v)
	}

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("error making request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

type ResultRecorder struct {
	results map[string]interface{}
}

func NewResultRecorder(logger *ManualLogger, request HeliconeLogRequest) *ResultRecorder {
	return &ResultRecorder{
		results: make(map[string]interface{}),
	}
}

func (r *ResultRecorder) AppendResults(data map[string]interface{}) {
	for k, v := range data {
		r.results[k] = v
	}
}

func (r *ResultRecorder) GetResults() map[string]interface{} {
	return r.results
}
