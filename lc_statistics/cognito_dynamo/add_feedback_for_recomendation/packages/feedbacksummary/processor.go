package feedbacksummary

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	openai "github.com/sashabaranov/go-openai"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_feedback_for_recomendation/packages/db"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_feedback_for_recomendation/packages/typesandstructs"
)

const (
	defaultOpenAIModel       = "gpt-4o-mini"
	defaultOpenAITokens      = 512
	defaultTimeoutSeconds    = 8
	openAIBufferMilliseconds = 250
	defaultFeedbackBatchSize = 30
)

var (
	openAIClient    *openai.Client
	openAIModel     string
	openAIMaxTokens int
	openAITimeout   time.Duration

	ErrNoFeedback = errors.New("no recent feedback available to process")
)

func ensureOpenAIConfigured() error {
	if openAIClient != nil {
		return nil
	}

	apiKey := strings.TrimSpace(os.Getenv("OPENAI_API_KEY"))
	if apiKey == "" {
		return fmt.Errorf("OPENAI_API_KEY environment variable is not set")
	}

	openAIModel = getEnvOrDefault("OPENAI_MODEL", defaultOpenAIModel)
	openAIMaxTokens = getIntEnv("OPENAI_MAX_TOKENS", defaultOpenAITokens)
	openAITimeout = getTimeoutEnv("OPENAI_TIMEOUT_SECONDS", defaultTimeoutSeconds)

	config := openai.DefaultConfig(apiKey)
	if openAITimeout > 0 {
		config.HTTPClient = &http.Client{Timeout: openAITimeout + 2*time.Second}
	}

	openAIClient = openai.NewClientWithConfig(config)
	return nil
}

func ProcessUserFeedback(ctx context.Context, userID string, feedbackLimit int) (*typesandstructs.ProcessedFeedbackSummary, error) {
	if err := ensureOpenAIConfigured(); err != nil {
		return nil, err
	}

	if feedbackLimit <= 0 {
		feedbackLimit = defaultFeedbackBatchSize
	}

	profile, err := db.FetchUserProfile(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch user profile: %w", err)
	}

	pairs, err := fetchRecentFeedbackPairs(ctx, userID, feedbackLimit)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch recent feedbacks: %w", err)
	}

	if len(pairs) == 0 {
		return nil, ErrNoFeedback
	}

	prompt, err := BuildPrompt(profile, pairs)
	if err != nil {
		return nil, fmt.Errorf("failed to build prompt: %w", err)
	}

	log.Printf("Prompt: %s", prompt)

	rawSummary, err := requestSummary(ctx, prompt)
	if err != nil {
		return nil, fmt.Errorf("failed to request summary: %w", err)
	}

	log.Printf("Raw summary response: %s", rawSummary)

	details, err := parseStructuredSummary(rawSummary)
	if err != nil {
		return nil, fmt.Errorf("failed to parse AI summary: %w", err)
	}

	feedbackIDs, recommendationIDs := collectIdentifiers(pairs)

	record := typesandstructs.ProcessedFeedbackSummary{
		UserID:            userID,
		FeedbackCount:     len(pairs),
		FeedbackIDs:       feedbackIDs,
		RecommendationIDs: recommendationIDs,
		PromptUsed:        prompt,
		ModelUsed:         openAIModel,
		SummaryText:       details.Narrative,
		StructuredSummary: details,
		RawModelResponse:  strings.TrimSpace(rawSummary),
		AnalyzedAtUTC:     time.Now().UTC().Format(time.RFC3339),
	}

	if err := db.SaveProcessedFeedbackSummary(ctx, &record); err != nil {
		return nil, fmt.Errorf("failed to store processed summary: %w", err)
	}

	log.Printf("Stored processed feedback summary with id %s for user %s", record.SummaryID, userID)

	return &record, nil
}

type feedbackRecommendationPair struct {
	Feedback       typesandstructs.StoredFeedback
	Recommendation *typesandstructs.RecommendationRecord
}

func fetchRecentFeedbackPairs(ctx context.Context, userID string, limit int) ([]feedbackRecommendationPair, error) {
	feedbacks, err := db.FetchRecentFeedbacks(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	if len(feedbacks) == 0 {
		return nil, nil
	}

	recIDs := make([]string, 0, len(feedbacks))
	for _, f := range feedbacks {
		recIDs = append(recIDs, f.RecomendationID)
	}

	recs, err := db.FetchRecommendationBatch(ctx, recIDs)
	if err != nil {
		return nil, err
	}

	recMap := make(map[string]*typesandstructs.RecommendationRecord, len(recs))
	for i := range recs {
		rec := recs[i]
		recMap[rec.RecommendationID] = &rec
	}

	pairs := make([]feedbackRecommendationPair, 0, len(feedbacks))
	for _, f := range feedbacks {
		pairs = append(pairs, feedbackRecommendationPair{
			Feedback:       f,
			Recommendation: recMap[f.RecomendationID],
		})
	}
	return pairs, nil
}

func collectIdentifiers(pairs []feedbackRecommendationPair) (feedbackIDs []string, recommendationIDs []string) {
	feedbackIDs = make([]string, 0, len(pairs))
	recSet := make(map[string]struct{})

	for _, pair := range pairs {
		if id := strings.TrimSpace(pair.Feedback.FeedbackID); id != "" {
			feedbackIDs = append(feedbackIDs, id)
		}
		if pair.Recommendation != nil {
			if recID := strings.TrimSpace(pair.Recommendation.RecommendationID); recID != "" {
				recSet[recID] = struct{}{}
			}
		}
	}

	for id := range recSet {
		recommendationIDs = append(recommendationIDs, id)
	}
	sort.Strings(recommendationIDs)
	return feedbackIDs, recommendationIDs
}

func requestSummary(ctx context.Context, prompt string) (string, error) {
	callCtx, cancel := withOpenAITimeout(ctx)
	if cancel != nil {
		defer cancel()
	}
	if callCtx == nil {
		return "", fmt.Errorf("insufficient time remaining for OpenAI request")
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
		return "", fmt.Errorf("no summary returned")
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

func getTimeoutEnv(name string, fallback int) time.Duration {
	value := strings.TrimSpace(os.Getenv(name))
	if value == "" {
		return time.Duration(fallback) * time.Second
	}
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed <= 0 {
		log.Printf("Invalid %s value %q, using fallback %d seconds", name, value, fallback)
		return time.Duration(fallback) * time.Second
	}
	return time.Duration(parsed) * time.Second
}

func parseStructuredSummary(raw string) (typesandstructs.ProcessedFeedbackDetails, error) {
	type payload struct {
		Narrative       string   `json:"narrative"`
		PriorityTopics  []string `json:"priority_topics"`
		AvoidTopics     []string `json:"avoid_topics"`
		ToneGuidance    []string `json:"tone_guidance"`
		CoachingActions []string `json:"coaching_actions"`
		Confidence      string   `json:"confidence"`
	}

	extracted := extractJSONObject(raw)
	if extracted == "" {
		return typesandstructs.ProcessedFeedbackDetails{}, fmt.Errorf("no JSON object found in response")
	}

	var p payload
	if err := json.Unmarshal([]byte(extracted), &p); err != nil {
		return typesandstructs.ProcessedFeedbackDetails{}, fmt.Errorf("failed to unmarshal summary JSON: %w", err)
	}

	details := typesandstructs.ProcessedFeedbackDetails{
		Narrative:       strings.TrimSpace(p.Narrative),
		PriorityTopics:  normalizeArray(p.PriorityTopics),
		AvoidTopics:     normalizeArray(p.AvoidTopics),
		ToneGuidance:    normalizeArray(p.ToneGuidance),
		CoachingActions: normalizeArray(p.CoachingActions),
		Confidence:      strings.ToLower(strings.TrimSpace(p.Confidence)),
	}

	if details.Narrative == "" {
		return typesandstructs.ProcessedFeedbackDetails{}, fmt.Errorf("summary narrative is empty")
	}

	return details, nil
}

func extractJSONObject(input string) string {
	trimmed := strings.TrimSpace(input)
	if strings.HasPrefix(trimmed, "```") {
		trimmed = strings.TrimPrefix(trimmed, "```json")
		trimmed = strings.TrimPrefix(trimmed, "```JSON")
		trimmed = strings.TrimPrefix(trimmed, "```")
		if idx := strings.LastIndex(trimmed, "```"); idx != -1 {
			trimmed = trimmed[:idx]
		}
	}
	trimmed = strings.TrimSpace(trimmed)

	start := strings.Index(trimmed, "{")
	end := strings.LastIndex(trimmed, "}")
	if start == -1 || end == -1 || start >= end {
		return ""
	}
	return trimmed[start : end+1]
}

func normalizeArray(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	set := make(map[string]string)
	for _, v := range values {
		trimmed := strings.TrimSpace(v)
		if trimmed == "" {
			continue
		}
		key := strings.ToLower(trimmed)
		if _, exists := set[key]; exists {
			continue
		}
		set[key] = trimmed
	}
	if len(set) == 0 {
		return nil
	}
	result := make([]string, 0, len(set))
	for _, value := range set {
		result = append(result, value)
	}
	sort.Slice(result, func(i, j int) bool {
		return strings.ToLower(result[i]) < strings.ToLower(result[j])
	})
	return result
}
