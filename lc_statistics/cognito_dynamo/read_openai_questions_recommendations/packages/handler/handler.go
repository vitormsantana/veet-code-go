package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/aws/aws-lambda-go/events"
	openai "github.com/sashabaranov/go-openai"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_openai_questions_recommendations/packages/auth"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_openai_questions_recommendations/packages/buildprompt"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_openai_questions_recommendations/packages/db"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_openai_questions_recommendations/packages/generatestatistics"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_openai_questions_recommendations/packages/structstypes"
)

var (
	openAIClient    *openai.Client
	openAIModel     string
	openAIMaxTokens int
	openAITimeout   time.Duration
)

const (
	defaultGoal              = "Be better in coding interviews."
	defaultOpenAIModel       = "gpt-4o-mini"
	defaultOpenAITokens      = 512
	defaultTimeoutSeconds    = 8
	openAIBufferMilliseconds = 250
)

func init() {
	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		log.Printf("OPENAI_API_KEY environment variable is not set")
		return
	}

	openAIModel = getEnvOrDefault("OPENAI_MODEL", defaultOpenAIModel)
	openAIMaxTokens = getIntEnv("OPENAI_MAX_TOKENS", defaultOpenAITokens)
	openAITimeout = getTimeoutEnv("OPENAI_TIMEOUT_SECONDS", defaultTimeoutSeconds)

	config := openai.DefaultConfig(apiKey)
	if openAITimeout > 0 {
		config.HTTPClient = &http.Client{Timeout: openAITimeout + 2*time.Second}
	}

	openAIClient = openai.NewClientWithConfig(config)
}

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization, X-Amz-Date, X-Api-Key, X-Amz-Security-Token",
	}

	if event.HTTPMethod == http.MethodOptions {
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusNoContent,
			Headers:    headers,
		}
		log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
		return response, nil
	}

	if openAIClient == nil {
		log.Printf("OpenAI client is not initialized. Check OPENAI_API_KEY environment variable")
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"message":"OpenAI client not initialized"}`,
		}
		log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
		return response, nil
	}

	authHeader := getAuthorizationHeader(event.Headers)
	userID, err := auth.GetUserIDFromToken(authHeader)
	if err != nil {
		log.Printf("Unauthorized request: %v", err)
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Headers:    headers,
			Body:       `{"message":"unauthorized"}`,
		}
		log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
		return response, nil
	}

	goal := extractGoal(event)

	profile, err := db.FetchUserProfile(ctx, userID)
	if err != nil {
		log.Printf("Failed to fetch profile: %v", err)
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"message":"Internal Server Error"}`,
		}
		log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
		return response, nil
	}

	questions, err := db.FetchQuestions(ctx, userID)
	if err != nil {
		log.Printf("Failed to fetch questions: %v", err)
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"message":"Internal Server Error"}`,
		}
		log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
		return response, nil
	}

	stats := generatestatistics.GenerateStatistics(questions)
	questionNames := uniqueQuestionNames(questions)

	metrics, err := db.FetchLatestMetrics(ctx, userID)
	if err != nil {
		log.Printf("Failed to fetch user metrics: %v", err)
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"message":"Internal Server Error"}`,
		}
		log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
		return response, nil
	}

	prompt, err := buildprompt.BuildPrompt(goal, questionNames, stats, profile, metrics)
	if err != nil {
		log.Printf("Failed to build prompt: %v", err)
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"message":"Internal Server Error"}`,
		}
		log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
		return response, nil
	}

	log.Printf("Prompt: %s", prompt)

	suggestions, err := requestSuggestions(ctx, prompt)
	if err != nil {
		log.Printf("Error generating suggestions: %v", err)
		status := http.StatusInternalServerError
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, errInsufficientTime) {
			status = http.StatusGatewayTimeout
		}
		response := events.APIGatewayProxyResponse{
			StatusCode: status,
			Headers:    headers,
			Body:       fmt.Sprintf(`{"message":"%s"}`, escapeJSONString(err.Error())),
		}
		log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
		return response, nil
	}

	log.Printf("Suggestions: %s", suggestions)

	responsePayload, err := json.Marshal(map[string]string{"suggestions": suggestions})
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"message":"Internal Server Error"}`,
		}
		log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
		return response, nil
	}

	response := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    headers,
		Body:       string(responsePayload),
	}
	log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
	return response, nil
}

func extractGoal(event events.APIGatewayProxyRequest) string {
	goal := defaultGoal

	if value, ok := event.QueryStringParameters["goal"]; ok {
		trimmed := strings.TrimSpace(value)
		if trimmed != "" {
			goal = trimmed
		}
	}

	if goal != defaultGoal {
		return goal
	}

	if event.Body == "" {
		return goal
	}

	var payload struct {
		Goal string `json:"goal"`
	}
	if err := json.Unmarshal([]byte(event.Body), &payload); err != nil {
		return goal
	}
	trimmed := strings.TrimSpace(payload.Goal)
	if trimmed != "" {
		goal = trimmed
	}

	return goal
}

func uniqueQuestionNames(questions []structstypes.Question) []string {
	seen := make(map[string]struct{})
	result := make([]string, 0, len(questions))
	for _, q := range questions {
		name := strings.TrimSpace(q.Name)
		if name == "" {
			continue
		}
		if _, exists := seen[name]; exists {
			continue
		}
		seen[name] = struct{}{}
		result = append(result, name)
	}
	return result
}

func getAuthorizationHeader(headers map[string]string) string {
	if headers == nil {
		return ""
	}

	if val := headers["Authorization"]; val != "" {
		return val
	}

	if val := headers["authorization"]; val != "" {
		return val
	}

	for key, val := range headers {
		if strings.EqualFold(key, "Authorization") && val != "" {
			return val
		}
	}

	return ""
}

func escapeJSONString(input string) string {
	b, err := json.Marshal(input)
	if err != nil {
		return input
	}
	escaped := string(b)
	if len(escaped) >= 2 {
		return escaped[1 : len(escaped)-1]
	}
	return escaped
}

var errInsufficientTime = errors.New("insufficient time remaining for OpenAI request")

func requestSuggestions(ctx context.Context, prompt string) (string, error) {
	if openAIClient == nil {
		return "", fmt.Errorf("OpenAI client not initialized")
	}

	callCtx, cancel := withOpenAITimeout(ctx)
	if cancel != nil {
		defer cancel()
	}
	if callCtx == nil {
		return "", errInsufficientTime
	}

	if openAIMaxTokens <= 0 {
		openAIMaxTokens = defaultOpenAITokens
	}

	chatRequest := openai.ChatCompletionRequest{
		Model:     openAIModel,
		MaxTokens: openAIMaxTokens,
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    openai.ChatMessageRoleUser,
				Content: prompt,
			},
		},
	}

	chatResponse, err := openAIClient.CreateChatCompletion(callCtx, chatRequest)
	if err != nil {
		return "", err
	}

	if len(chatResponse.Choices) == 0 {
		return "", fmt.Errorf("no suggestions returned")
	}

	return chatResponse.Choices[0].Message.Content, nil
}

func withOpenAITimeout(ctx context.Context) (context.Context, context.CancelFunc) {
	buffer := time.Duration(openAIBufferMilliseconds) * time.Millisecond

	if openAITimeout <= 0 {
		if deadline, ok := ctx.Deadline(); ok {
			remaining := time.Until(deadline) - buffer
			if remaining <= 0 {
				return nil, nil
			}
			return context.WithTimeout(ctx, remaining)
		}
		return ctx, nil
	}

	if deadline, ok := ctx.Deadline(); ok {
		remaining := time.Until(deadline) - buffer
		if remaining <= 0 {
			return nil, nil
		}
		timeout := openAITimeout
		if timeout > remaining {
			timeout = remaining
			if timeout <= 0 {
				return nil, nil
			}
		}
		return context.WithTimeout(ctx, timeout)
	}

	return context.WithTimeout(ctx, openAITimeout)
}

func getEnvOrDefault(name, fallback string) string {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	return value
}

func getIntEnv(name string, fallback int) int {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		log.Printf("Invalid %s value %q, using fallback %d", name, value, fallback)
		return fallback
	}
	return parsed
}

func getTimeoutEnv(name string, fallbackSeconds int) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return time.Duration(fallbackSeconds) * time.Second
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		log.Printf("Invalid %s value %q, using fallback %d", name, value, fallbackSeconds)
		return time.Duration(fallbackSeconds) * time.Second
	}
	return time.Duration(parsed) * time.Second
}
