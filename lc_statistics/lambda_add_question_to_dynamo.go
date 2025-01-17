package main

import (
	"context"
	"fmt"
	"encoding/json"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-lambda-go/events"
)

type Request struct {
	QuestionName       string   `json:"name"`
	QuestionDate       string   `json:"date"`
	QuestionDifficulty string   `json:"difficulty"`
	QuestionTags       []string `json:"tags"`
}

var dynamoClient  *dynamodb.Client
const tableName = "veet_code_questions_table"

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("sa-east-1"))
	if err != nil {
		panic(fmt.Sprintf("Unable to load AWS SDK config: %v", err))
	}

	dynamoClient = dynamodb.NewFromConfig(cfg)
}

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (map[string]interface{}, error) {

	fmt.Println("Raw Event:", event)

	var request Request
	err := json.Unmarshal([]byte(event.Body), &request)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal request body: %v", err)
	}

	fmt.Println("Question Name: ", request.QuestionName)
	fmt.Println("Question Date: ", request.QuestionDate)
	fmt.Println("Question Difficulty: ", request.QuestionDifficulty)
	fmt.Println("Question Tags: ", request.QuestionTags)

	message := fmt.Sprintf("Question Name: %s, Question Date: %s, Question Difficulty: %s, Question Tags: %s", request.QuestionName, request.QuestionDate, request.QuestionDifficulty, request.QuestionTags)
	
	tagsJSON, err := json.Marshal(request.QuestionTags)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal tags: %v", err)
	}

	err = putItemToDynamoDB(request, string(tagsJSON))
	if err != nil {
		return nil, fmt.Errorf("failed to add item to DynamoDB: %v", err)
	}

	successMessage := "Question successfully added to DynamoDB."
	fullMessage := fmt.Sprintf("%s %s", successMessage, message)

	headers := map[string]string{
		"Access-Control-Allow-Origin":      "*",           
		"Access-Control-Allow-Methods":     "POST, OPTIONS",
		"Access-Control-Allow-Headers":     "Content-Type, Authorization",
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
		"body": string(body),
	}, nil
}

func putItemToDynamoDB(request Request, tagsJSON string) error {
	input := &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]types.AttributeValue{
			"question_name":       &types.AttributeValueMemberS{Value: request.QuestionName},
			"question_solved_date": &types.AttributeValueMemberS{Value: request.QuestionDate},
			"difficulty":          &types.AttributeValueMemberS{Value: request.QuestionDifficulty},
			"tags":                &types.AttributeValueMemberS{Value: tagsJSON},
		},
	}

	_, err := dynamoClient.PutItem(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to put item in DynamoDB: %v", err)
	}
	return nil
}

func main() {
	lambda.Start(Handler)
}
