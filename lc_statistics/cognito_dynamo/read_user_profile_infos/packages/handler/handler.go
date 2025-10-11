package handler

import (
	"context"
	"encoding/json"
	"log"

	"go.uber.org/zap"

	"github.com/aws/aws-lambda-go/events"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_user_profile_infos/packages/auth"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_user_profile_infos/packages/db"
)

var logger *zap.Logger

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatalf("Unable to initialize zap logger: %v", err)
	}
	logger.Info("Zap logger initialized in handler")
}

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	logger.Info("Handler invoked", zap.String("method", event.HTTPMethod), zap.String("path", event.Path))

	headers := map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "GET, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}

	authHeader := event.Headers["Authorization"]
	logger.Info("Extracting user ID from token", zap.String("auth_header", authHeader))

	userID, err := auth.GetUserIDFromToken(authHeader)
	if err != nil {
		logger.Warn("Unauthorized request", zap.Error(err))
		response := events.APIGatewayProxyResponse{
			StatusCode: 401,
			Headers:    headers,
			Body:       `{"message":"Unauthorized"}`,
		}
		logger.Info("Returning response", zap.Int("status", response.StatusCode), zap.String("body", response.Body))
		return response, nil
	}

	logger.Info("Fetching user profile", zap.String("user_id", userID))
	profile, err := db.FetchProfileInfos(ctx, userID)
	if err != nil {
		logger.Error("Failed to fetch profile infos", zap.String("user_id", userID), zap.Error(err))
		response := events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers:    headers,
			Body:       `{"message":"Internal Server Error"}`,
		}
		logger.Info("Returning response", zap.Int("status", response.StatusCode), zap.String("body", response.Body))
		return response, nil
	}

	logger.Info("Marshaling user profile to JSON", zap.String("user_id", userID))
	responseBody, err := json.Marshal(profile)
	if err != nil {
		logger.Error("Failed to marshal response", zap.String("user_id", userID), zap.Error(err))
		response := events.APIGatewayProxyResponse{
			StatusCode: 500,
			Headers:    headers,
			Body:       `{"message":"Internal Server Error"}`,
		}
		logger.Info("Returning response", zap.Int("status", response.StatusCode), zap.String("body", response.Body))
		return response, nil
	}

	response := events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers:    headers,
		Body:       string(responseBody),
	}
	logger.Info("Returning response", zap.Int("status", response.StatusCode), zap.String("body", response.Body))
	return response, nil
}
