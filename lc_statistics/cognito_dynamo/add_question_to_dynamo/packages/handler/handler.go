package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"

	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_question_to_dynamo/packages/auth"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_question_to_dynamo/packages/db"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_question_to_dynamo/packages/typesandstructs"

	"github.com/aws/aws-lambda-go/events"
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

	var request typesandstructs.Request
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

	if err := db.PutItem(userID, request); err != nil {
		log.Printf("Failed to add item to DynamoDB: %v", err)
		response := events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"message":"failed to add item"}`,
		}
		log.Printf("Returning response: status=%d body=%s", response.StatusCode, response.Body)
		return response, nil
	}

	successMessage := "Question successfully added to DynamoDB."
	fullMessage := fmt.Sprintf("%s Question Name: %s, Date: %s", successMessage, request.QuestionName, request.QuestionDate)

	responseBody, err := json.Marshal(map[string]string{"message": fullMessage})
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
