package main

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
)

type Request struct {
	StudyTheme       string   `json:"theme"`
	StudyDate       string   `json:"date"`
	StudyMinutes string   `json:"minutes"`
}

var dynamoClient  *dynamodb.Client
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

	fmt.Println("Study Theme: ", request.StudyTheme)
	fmt.Println("Study Date: ", request.StudyDate)
	fmt.Println("Minutes of Study: ", request.StudyMinutes)

	message := fmt.Sprintf("Study Theme: %s, Study Date: %s, Minutes of Study: %s", request.StudyTheme, request.StudyDate, request.StudyMinutes)
	
	err = putItemToDynamoDB(request)
	if err != nil {
		return nil, fmt.Errorf("failed to add item to DynamoDB: %v", err)
	}

	successMessage := "Study successfully added to DynamoDB."
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

func putItemToDynamoDB(request Request) error {
	minutes, err := strconv.Atoi(request.StudyMinutes)
	if err != nil {
    		return fmt.Errorf("invalid minutes_of_study: %v", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(tableName),
		Item: map[string]types.AttributeValue{
			"study_theme":       	&types.AttributeValueMemberS{Value: request.StudyTheme},
			"study_date": 		&types.AttributeValueMemberS{Value: request.StudyDate},
			"minutes_of_study":     &types.AttributeValueMemberN{Value: strconv.Itoa(minutes)},
		},
	}

	_, err = dynamoClient.PutItem(context.TODO(), input)
	if err != nil {
		return fmt.Errorf("failed to put item in DynamoDB: %v", err)
	}
	return nil
}

func main() {
	lambda.Start(Handler)
}
