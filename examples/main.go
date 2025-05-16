package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"

	"github.com/google/uuid"
	logger "github.com/helicone/go-helicone-helpers"
	"github.com/openai/openai-go"
	"github.com/openai/openai-go/option"
)

func main() {
	// Replace with your actual API key
	apiKey := os.Getenv("HELICONE_API_KEY")
	openaiApiKey := os.Getenv("OPENAI_API_KEY")

	// Example: Basic Logger
	fmt.Println("Testing Basic Logger...")
	testBasicLogger(apiKey, openaiApiKey)
}

func testBasicLogger(apiKey string, openaiApiKey string) {
	sessionId := uuid.New().String()
	manualLogger := logger.New(logger.LoggerOptions{
		APIKey: apiKey,
		Headers: map[string]string{
			"Helicone-User-Id": "test-user-123",
		},
	})

	request := logger.ILogRequest{
		Model: "gpt-4o",
		Extra: map[string]interface{}{
			"messages": []map[string]string{
				{"role": "user", "content": "Hello from basic logger!"},
			},
		},
	}
	openaiClient := openai.NewClient(option.WithAPIKey(openaiApiKey))

	result, err := manualLogger.LogRequest(request, func(recorder *logger.ResultRecorder) (interface{}, error) {
		chatCompletion, err := openaiClient.Chat.Completions.New(context.TODO(), openai.ChatCompletionNewParams{
			Messages: []openai.ChatCompletionMessageParamUnion{
				openai.UserMessage("Say this is a test"),
			},
			Model: openai.ChatModelGPT4o,
		})
		if err != nil {
			panic(err.Error())
		}
		// Simulate some processing time
		jsonData, _ := json.Marshal(chatCompletion)
		var resultMap map[string]interface{}
		json.Unmarshal(jsonData, &resultMap)
		recorder.AppendResults(resultMap)
		return "Response from basic logger test", nil
	}, map[string]string{
		"Helicone-Session-Name": "XPedia Travel Planner",
		"Helicone-Session-Id":   sessionId,
		"Helicone-Session-Path": `/planning/tips/llm`,
	})
	if err != nil {
		log.Fatal("Basic logger error:", err)
	}

	fmt.Printf("Basic logger result: %v\n", result)

	request2 := logger.HeliconeEventTool{
		ToolName: "FlightBookingAPI",
		Input: map[string]interface{}{
			"flightId":      "123",
			"cabinClass":    "economy",
			"departureDate": "2023-01-01",
			"returnDate":    "2023-01-05",
			"origin":        "LAX",
			"destination":   "SFO",
		},
	}

	request3 := logger.HeliconeEventVectorDB{
		Operation:    logger.VectorDBOperationSearch,
		Text:         "Generate travel tips for a trip to San Francisco",
		Vector:       []float64{1.0, 2.0, 3.0},
		TopK:         5,
		Filter:       map[string]interface{}{"destination": "SFO"},
		DatabaseName: "travel_tips",
		Extra: map[string]interface{}{
			"query": map[string]interface{}{
				"destination_parsed": "San Francisco",
			},
		},
	}

	result2, err := manualLogger.LogRequest(request2, func(recorder *logger.ResultRecorder) (interface{}, error) {
		recorder.AppendResults(map[string]interface{}{
			"status": "success",
			"booking": map[string]interface{}{
				"bookingId":     "123",
				"flightId":      "123",
				"cabinClass":    "economy",
				"departureDate": "2023-01-01",
				"returnDate":    "2023-01-05",
				"origin":        "LAX",
				"destination":   "SFO",
			},
		})
		return "Response from tool logger test", nil
	}, map[string]string{
		"Helicone-Session-Name": "XPedia Travel Planner",
		"Helicone-Session-Id":   sessionId,
		"Helicone-Session-Path": `/planning/tips/tool`,
	})
	if err != nil {
		log.Fatal("Tool logger error:", err)
	}

	fmt.Printf("Tool logger result: %v\n", result2)

	result3, err := manualLogger.LogRequest(request3, func(recorder *logger.ResultRecorder) (interface{}, error) {
		recorder.AppendResults(map[string]interface{}{
			"status":              "failed",
			"message":             "no travel tips found",
			"similarityThreshold": 0.5,
			"actualSimilarity":    0.5,
			"metadata": map[string]interface{}{
				"destination":        "San Francisco",
				"destination_parsed": "San Francisco",
				"timestamp":          "2023-01-01T00:00:00Z",
			},
		})
		return "Response from vector db logger test", nil
	}, map[string]string{
		"Helicone-Session-Name": "XPedia Travel Planner",
		"Helicone-Session-Id":   sessionId,
		"Helicone-Session-Path": `/planning/tips/vector-db`,
	})

	if err != nil {
		log.Fatal("Vector db logger error:", err)
	}

	fmt.Printf("Vector db logger result: %v\n", result3)
}
