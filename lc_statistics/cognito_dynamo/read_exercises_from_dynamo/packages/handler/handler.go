package handler

import (
	"context"
	"encoding/json"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_exercises_from_dynamo/packages/auth"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_exercises_from_dynamo/packages/db"
)

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	headers := map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}

	authHeader := event.Headers["Authorization"]
	userID, err := auth.GetUserIDFromToken(authHeader)
	if err != nil {
		log.Printf("Unauthorized request: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 401,
			Headers:    headers,
			Body:       `{"message":"Unauthorized"}`,
		}, nil
	}

	questions, err := db.FetchQuestions(ctx, userID)
	if err != nil {
		log.Printf("Failed to fetch questions: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers:    headers,
			Body:       `{"message":"Internal Server Error"}`,
		}, nil
	}

	responseBody, err := json.Marshal(questions)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers:    headers,
			Body:       `{"message":"Internal Server Error"}`,
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    headers,
		Body:       string(responseBody),
	}, nil
}
