package main

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Request struct {
	QuestionName       string   `json:"name"`
	QuestionDate       string   `json:"date"`
	QuestionDifficulty string   `json:"difficulty"`
	QuestionTags       []string `json:"tags"`
	MinutesTaken       int      `json:"minutes_taken"`
	NeededHelp         bool     `json:"needed_help"`
}

const tableName = "veet_code_questions_table"

var dynamoClient *dynamodb.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(),
		config.WithRegion("sa-east-1"),
		config.WithEndpointResolver(aws.EndpointResolverFunc(
			func(service, region string) (aws.Endpoint, error) {
				if service == dynamodb.ServiceID && region == "sa-east-1" {
					return aws.Endpoint{
						URL:           "http://localhost:8000",
						SigningRegion: "sa-east-1",
					}, nil
				}
				return aws.Endpoint{}, fmt.Errorf("unknown endpoint requested")
			},
		)),
	)
	if err != nil {
		panic(fmt.Sprintf("unable to load AWS SDK config: %v", err))
	}

	dynamoClient = dynamodb.NewFromConfig(cfg)
}

func main() {
	// Local test
	payload := Request{
		QuestionName:       "Two Sum",
		QuestionDate:       "2025-10-03",
		QuestionDifficulty: "Easy",
		QuestionTags:       []string{"Array", "HashMap"},
		MinutesTaken:       15,
		NeededHelp:         false,
	}

	body, _ := json.Marshal(payload)

	event := events.APIGatewayProxyRequest{
		Body: string(body),
		Headers: map[string]string{
			"Authorization": "Bearer dummy-sub-token",
		},
	}

	resp, err := Handler(context.Background(), event)
	if err != nil {
		fmt.Println("Error:", err)
		return
	}

	respJSON, _ := json.MarshalIndent(resp, "", "  ")
	fmt.Println(string(respJSON))
}

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (map[string]interface{}, error) {
	fmt.Println("Raw Event:", event)

	userID := "dummy-user-id" // mock user for local testing

	var request Request
	if err := json.Unmarshal([]byte(event.Body), &request); err != nil {
		return nil, fmt.Errorf("failed to unmarshal request body: %v", err)
	}

	questionID := fmt.Sprintf("%d", time.Now().UnixNano())
	tagsJSON, _ := json.Marshal(request.QuestionTags)

	if err := putItemToDynamoDB(userID, questionID, request, string(tagsJSON)); err != nil {
		return nil, err
	}

	successMessage := fmt.Sprintf("Question successfully added to DynamoDB. Name: %s, Date: %s", request.QuestionName, request.QuestionDate)
	return map[string]interface{}{
		"statusCode": 200,
		"body":       successMessage,
	}, nil
}

func putItemToDynamoDB(userID, questionID string, request Request, tagsJSON string) error {
	input := &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]types.AttributeValue{
			"user_id":              &types.AttributeValueMemberS{Value: userID},
			"question_id":          &types.AttributeValueMemberS{Value: questionID},
			"question_name":        &types.AttributeValueMemberS{Value: request.QuestionName},
			"question_solved_date": &types.AttributeValueMemberS{Value: request.QuestionDate},
			"difficulty":           &types.AttributeValueMemberS{Value: request.QuestionDifficulty},
			"tags":                 &types.AttributeValueMemberS{Value: tagsJSON},
			"minutes_taken":        &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", request.MinutesTaken)},
			"needed_help":          &types.AttributeValueMemberBOOL{Value: request.NeededHelp},
		},
	}

	_, err := dynamoClient.PutItem(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to put item in DynamoDB: %v", err)
	}
	return nil
}

