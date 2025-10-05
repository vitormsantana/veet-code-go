package handler

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_question_to_dynamo/packages/auth"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_question_to_dynamo/packages/db"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_question_to_dynamo/packages/typesandstructs"

	"github.com/aws/aws-lambda-go/events"
)

func init() {
	db.Init()
}

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (map[string]interface{}, error) {

	fmt.Println("Raw Event:", event)

	authHeader := event.Headers["Authorization"]
	userID, err := auth.GetUserIDFromToken(authHeader)
	if err != nil {
		return nil, fmt.Errorf("unauthorized: %v", err)
	}

	var request typesandstructs.Request
	err = json.Unmarshal([]byte(event.Body), &request)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal request body: %v", err)
	}

	err = db.PutItem(userID, request)
	if err != nil {
		return nil, fmt.Errorf("failed to add item to DynamoDB: %v", err)
	}

	successMessage := "Question successfully added to DynamoDB."
	fullMessage := fmt.Sprintf("%s Question Name: %s, Date: %s", successMessage, request.QuestionName, request.QuestionDate)

	headers := map[string]string{
		"Access-Control-Allow-Origin":  "*",
		"Access-Control-Allow-Methods": "POST, OPTIONS",
		"Access-Control-Allow-Headers": "Content-Type, Authorization",
	}

	body, err := json.Marshal(map[string]string{
		"message": fullMessage,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to marshal response body: %v", err)
	}

	return map[string]interface{}{
		"statusCode": 200,
		"headers":    headers,
		"body":       string(body),
	}, nil
}
