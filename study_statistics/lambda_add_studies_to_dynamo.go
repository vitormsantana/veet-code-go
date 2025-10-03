package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Request struct {
	Studies []Study `json:"studies"`
}

type Study struct {
	StudyTheme   string `json:"theme"`
	StudyDate    string `json:"date"`
	StudyMinutes string `json:"minutes"`
}

var dynamoClient *dynamodb.Client
const tableName = "studies_table"

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

	fmt.Println("Received Studies:", request.Studies)

	err = putMultipleItemsToDynamoDB(request.Studies)
	if err != nil {
		return nil, fmt.Errorf("failed to add items to DynamoDB: %v", err)
	}

	successMessage := fmt.Sprintf("%d studies successfully added to DynamoDB.", len(request.Studies))

	headers := map[string]string{
		"Access-Control-Allow-Origin":      "*",           
		"Access-Control-Allow-Methods":     "POST, OPTIONS",
		"Access-Control-Allow-Headers":     "Content-Type, Authorization",
	}

	body, err := json.Marshal(map[string]string{
		"message": successMessage,
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

func putMultipleItemsToDynamoDB(studies []Study) error {
	var writeRequests []types.WriteRequest

	for _, study := range studies {
		minutes, err := strconv.Atoi(study.StudyMinutes)
		if err != nil {
			return fmt.Errorf("invalid minutes_of_study: %v", err)
		}

		writeRequests = append(writeRequests, types.WriteRequest{
			PutRequest: &types.PutRequest{
				Item: map[string]types.AttributeValue{
					"study_theme":    &types.AttributeValueMemberS{Value: study.StudyTheme},
					"study_date":     &types.AttributeValueMemberS{Value: study.StudyDate},
					"minutes_of_study": &types.AttributeValueMemberN{Value: strconv.Itoa(minutes)},
				},
			},
		})
	}

	// Batch write with a maximum of 25 items per request (DynamoDB limit)
	const maxBatchSize = 25
	for i := 0; i < len(writeRequests); i += maxBatchSize {
		end := i + maxBatchSize
		if end > len(writeRequests) {
			end = len(writeRequests)
		}

		input := &dynamodb.BatchWriteItemInput{
			RequestItems: map[string][]types.WriteRequest{
				tableName: writeRequests[i:end],
			},
		}

		_, err := dynamoClient.BatchWriteItem(context.TODO(), input)
		if err != nil {
			return fmt.Errorf("failed to batch write items to DynamoDB: %v", err)
		}
	}

	return nil
}

func main() {
	lambda.Start(Handler)
}
