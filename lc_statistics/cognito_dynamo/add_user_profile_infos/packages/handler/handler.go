package handler

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_user_profile_infos/packages/auth"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_user_profile_infos/packages/calculateprofilescore"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_user_profile_infos/packages/db"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_user_profile_infos/packages/typesandstructs"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	baseLogger *zap.Logger
	logger     *zap.SugaredLogger
)

func init() {
	config := zap.NewProductionConfig()
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	config.Level = zap.NewAtomicLevelAt(zap.InfoLevel)

	l, err := config.Build()
	if err != nil {
		panic(fmt.Sprintf("failed to create zap logger: %v", err))
	}
	baseLogger = l
	zap.ReplaceGlobals(baseLogger)
	logger = baseLogger.Sugar()

	logger.Infow("handler init: initializing DynamoDB client", "table", db.TableName)
	db.Init(logger)
	logger.Infow("DynamoDB client ready")
}

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	if baseLogger != nil {
		defer func() {
			if err := baseLogger.Sync(); err != nil && !errors.Is(err, syscall.ENOTTY) && !errors.Is(err, syscall.EINVAL) {
				fmt.Fprintf(os.Stderr, "zap sync error: %v\n", err)
			}
		}()
	}

	headers := map[string]string{
		"Content-Type":                 "application/json",
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "POST, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}

	if event.HTTPMethod == http.MethodOptions {
		logger.Infow("OPTIONS preflight handled", "path", event.Path, "requestId", event.RequestContext.RequestID)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusNoContent,
			Headers:    headers,
		}, nil
	}

	logger.Infow("Request received", "method", event.HTTPMethod, "path", event.Path, "requestId", event.RequestContext.RequestID)
	authHeader := getAuthorizationHeader(event.Headers)
	userID, err := auth.GetUserIDFromToken(authHeader)
	if err != nil {
		logger.Warnw("Unauthorized request", "error", err, "requestId", event.RequestContext.RequestID)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusUnauthorized,
			Headers:    headers,
			Body:       `{"message":"unauthorized"}`,
		}, nil
	}

	var request typesandstructs.UserProfile
	if err := json.Unmarshal([]byte(event.Body), &request); err != nil {
		logger.Warnw("Failed to unmarshal request body", "error", err, "requestId", event.RequestContext.RequestID)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusBadRequest,
			Headers:    headers,
			Body:       `{"message":"invalid request body"}`,
		}, nil
	}
	logger.Infow("Parsed profile payload", "userID", userID, "target_companies", request.TargetCompanies, "desired_role", request.DesiredRole, "years_experience", request.YearsOfExperience)

	request.UserID = userID

	request.AIPersonalizationScore = calculateprofilescore.CalculateProfileScore(&request)
	logger.Infow("Calculated personalization score", "userID", userID, "score", request.AIPersonalizationScore)

	existing, err := db.GetUserProfile(userID)
	if err != nil {
		logger.Warnw("Failed to retrieve existing profile", "userID", userID, "error", err)
	}
	if existing != nil && existing.AIPersonalizationScore > request.AIPersonalizationScore {
		request.AIPersonalizationScore = existing.AIPersonalizationScore
		logger.Infow("Preserved higher existing personalization score", "userID", userID, "score", existing.AIPersonalizationScore)
	}

	if err := db.PutUserProfile(request); err != nil {
		logger.Errorw("Failed to save user profile", "userID", userID, "error", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"message":"failed to save user profile"}`,
		}, nil
	}
	logger.Infow("User profile persisted", "userID", userID)

	successMessage := fmt.Sprintf(
		"User profile successfully saved for %s (%s, %s). Personalization score: %.2f",
		request.UserID, request.DesiredRole, request.CountryTarget, request.AIPersonalizationScore,
	)

	responseBody, err := json.Marshal(map[string]string{"message": successMessage})
	if err != nil {
		logger.Errorw("Failed to marshal response payload", "userID", userID, "error", err)
		return events.APIGatewayProxyResponse{
			StatusCode: http.StatusInternalServerError,
			Headers:    headers,
			Body:       `{"message":"internal server error"}`,
		}, nil
	}
	logger.Infow("Returning success response", "userID", userID, "timestamp", time.Now().UTC())

	return events.APIGatewayProxyResponse{
		StatusCode: http.StatusOK,
		Headers:    headers,
		Body:       string(responseBody),
	}, nil
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
