package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-lambda-go/events"
	"github.com/aws/aws-lambda-go/lambda"
	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
)

type Study struct {
	StudyTheme   string `dynamodbav:"study_theme"`
	StudyDate    string `dynamodbav:"study_date"`
	StudyMinutes string `dynamodbav:"minutes_of_study"`
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

func Handler(ctx context.Context, event events.APIGatewayProxyRequest) (events.APIGatewayProxyResponse, error) {
	log.Printf("Raw Event: %+v", event)

	studies, err := fetchAllStudies(ctx)
	if err != nil {
		log.Printf("Failed to fetch studies: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, nil
	}

	responseBody, err := json.Marshal(studies)
	if err != nil {
		log.Printf("Failed to marshal response: %v", err)
		return events.APIGatewayProxyResponse{
			StatusCode: 500,
			Body:       "Internal Server Error",
		}, nil
	}

	return events.APIGatewayProxyResponse{
		StatusCode: 200,
		Headers: map[string]string{
			"Content-Type":                 "application/json",
			"Access-Control-Allow-Origin":  "*",
			"Access-Control-Allow-Methods": "GET, OPTIONS",
		},
		Body: string(responseBody),
	}, nil
}

func fetchAllStudies(ctx context.Context) ([]Study, error) {
	var studies []Study
	input := &dynamodb.ScanInput{
		TableName: aws.String(tableName),
	}

	paginator := dynamodb.NewScanPaginator(dynamoClient, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to scan DynamoDB: %w", err)
		}

		var pageStudies []Study
		err = attributevalue.UnmarshalListOfMaps(page.Items, &pageStudies)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal DynamoDB items: %w", err)
		}

		studies = append(studies, pageStudies...)
	}

	return studies, nil
}

func main() {
	lambda.Start(Handler)
}

