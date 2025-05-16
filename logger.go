package logger

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

const defaultLoggingEndpoint = "https://api.worker.helicone.ai/custom/v1/log"

// ManualLogger represents the Helicone manual logger
type ManualLogger struct {
	apiKey          string
	headers         map[string]string
	loggingEndpoint string
}

// New creates a new instance of ManualLogger
func New(opts LoggerOptions) *ManualLogger {
	endpoint := opts.LoggingEndpoint
	if endpoint == "" {
		endpoint = defaultLoggingEndpoint
	}

	return &ManualLogger{
		apiKey:          opts.APIKey,
		headers:         opts.Headers,
		loggingEndpoint: endpoint,
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

	req, err := http.NewRequest("POST", l.loggingEndpoint, bytes.NewBuffer(jsonData))
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
