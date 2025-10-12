package db

import (
	"context"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_exercises_from_dynamo/packages/structstypes"
)

var dynamoClient *dynamodb.Client

const tableName = "hammocker_user_metrics_table"

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("sa-east-1"))
	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}
	dynamoClient = dynamodb.NewFromConfig(cfg)
}

func FetchMetrics(ctx context.Context, userID string) ([]structstypes.UserMetrics, error) {
	var metrics []structstypes.UserMetrics

	input := &dynamodb.QueryInput{
		TableName:              aws.String(tableName),
		KeyConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
	}

	paginator := dynamodb.NewQueryPaginator(dynamoClient, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query DynamoDB: %w", err)
		}

		if len(page.Items) == 0 {
			continue
		}

		var pageMetrics []structstypes.UserMetrics
		err = attributevalue.UnmarshalListOfMaps(page.Items, &pageMetrics)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal DynamoDB items: %w", err)
		}

		metrics = append(metrics, pageMetrics...)
	}

	return metrics, nil
}
