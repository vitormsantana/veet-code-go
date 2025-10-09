package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_statistics_from_exercises_table/packages/structstypes"
)

var dynamoClient *dynamodb.Client

const tableName = "veet_code_questions_table"

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("sa-east-1"))
	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}
	dynamoClient = dynamodb.NewFromConfig(cfg)
}

func FetchQuestions(ctx context.Context, userID string) ([]structstypes.Question, error) {
	var questions []structstypes.Question

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

		for _, item := range page.Items {
			log.Printf("Raw item: %v", item)
		}

		var pageQuestions []struct {
			Name       string `dynamodbav:"question_name"`
			Date       string `dynamodbav:"question_solved_date"`
			Difficulty string `dynamodbav:"difficulty"`
			Tags       string `dynamodbav:"tags"`
		}
		err = attributevalue.UnmarshalListOfMaps(page.Items, &pageQuestions)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal DynamoDB items: %w", err)
		}

		for _, q := range pageQuestions {
			var tags []string
			if err := json.Unmarshal([]byte(q.Tags), &tags); err != nil {
				log.Printf("Failed to parse tags for question %s: %v", q.Name, err)
				tags = []string{}
			}

			questions = append(questions, structstypes.Question{
				Name:       q.Name,
				Date:       q.Date,
				Difficulty: q.Difficulty,
				Tags:       tags,
			})
		}
	}

	return questions, nil
}
