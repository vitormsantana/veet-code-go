package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/aws/aws-lambda-go/events"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_statistics_from_exercises_table/packages/auth"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_statistics_from_exercises_table/packages/db"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_statistics_from_exercises_table/packages/generatestatistics"
)

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, OPTIONS",
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

	_, err = json.Marshal(questions)
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

	stats := generatestatistics.GenerateStatistics(questions)
	statsJSON, err := json.MarshalIndent(stats, "", "  ")
	if err != nil {
		fmt.Printf("Failed to marshal stats: %v\n", err)
	} else {
		fmt.Printf("Generated stats(JSON): \n%s\n", statsJSON)
	}

	responseBody, err := json.Marshal(stats)
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
