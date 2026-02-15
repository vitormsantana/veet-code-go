package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_feedback_for_recomendation/packages/auth"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_feedback_for_recomendation/packages/db"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_feedback_for_recomendation/packages/typesandstructs"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_feedback_for_recomendation/packages/feedbacksummary"
)

func init() {
	db.Init()
}

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "POST, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}

	if event.HTTPMethod == http.MethodOptions {
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusNoContent,
			Headers:    headers,
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

	var request typesandstructs.Feedback
	if err := json.Unmarshal([]byte(event.Body), &request); err != nil {
		log.Printf("Failed to unmarshal request body: %v", err)
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    headers,
			Body:       `{"message":"invalid request body"}`,
		}
		log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
		return response, nil
	}

	if err := validateFeedback(request); err != nil {
		log.Printf("Invalid feedback payload: %v", err)
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    headers,
			Body:       fmt.Sprintf(`{"message":%q}`, err.Error()),
		}
		log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
		return response, nil
	}

	request.FeedbackComment = strings.TrimSpace(request.FeedbackComment)

	if err := db.PutFeedback(userID, request); err != nil {
		log.Printf("Failed to add item to DynamoDB: %v", err)
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"message":"failed to add feedback"}`,
		}
		log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
		return response, nil
	}

	processLatestFeedbackSummary(ctx, userID)

	confirmation := fmt.Sprintf("Feedback recorded for recommendation %s.", request.RecomendationID)

	responseBody, err := json.Marshal(map[string]string{"message": confirmation})
	if err != nil {
		log.Printf("Failed to marshal response payload: %v", err)
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"message":"internal server error"}`,
		}
		log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
		return response, nil
	}

	response := events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    headers,
		Body:       string(responseBody),
	}
	log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
	return response, nil
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

func validateFeedback(f typesandstructs.Feedback) error {
	if strings.TrimSpace(f.RecomendationID) == "" {
		return fmt.Errorf("recomendation_id is required")
	}

	if f.FeedbackValue != -1 && f.FeedbackValue != 1 {
		return fmt.Errorf("feedback_value must be -1 or 1")
	}

	return nil
}

func processLatestFeedbackSummary(ctx context.Context, userID string) {
	summary, err := feedbacksummary.ProcessUserFeedback(ctx, userID, 5)
	if err != nil {
		if errors.Is(err, feedbacksummary.ErrNoFeedback) {
			log.Printf("Processed feedback summary skipped for user %s: %v", userID, err)
			return
		}
		log.Printf("Failed to generate processed feedback summary for user %s: %v", userID, err)
		return
	}

	log.Printf("Processed feedback summary %s generated for user %s using %d feedbacks", summary.SummaryID, userID, summary.FeedbackCount)
}
