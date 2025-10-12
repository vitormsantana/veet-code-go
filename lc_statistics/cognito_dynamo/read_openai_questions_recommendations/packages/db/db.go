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
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_openai_questions_recommendations/packages/structstypes"
)

var dynamoClient *dynamodb.Client

const (
	questionsTableName = "veet_code_questions_table"
	metricsTableName   = "hammocker_user_metrics_table"
	profileTableName   = "hammocker_user_profiles_table"
)

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
		TableName:              aws.String(questionsTableName),
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

		var pageQuestions []struct {
			UserID       string `dynamodbav:"user_id"`
			QuestionID   string `dynamodbav:"question_id"`
			Name         string `dynamodbav:"question_name"`
			Date         string `dynamodbav:"question_solved_date"`
			Difficulty   string `dynamodbav:"difficulty"`
			TagsJSON     string `dynamodbav:"tags"`
			MinutesTaken int    `dynamodbav:"minutes_taken"`
			NeededHelp   bool   `dynamodbav:"needed_help"`
			Observation  string `dynamodbav:"obs"`
		}
		err = attributevalue.UnmarshalListOfMaps(page.Items, &pageQuestions)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal DynamoDB items: %w", err)
		}

		for _, q := range pageQuestions {
			var tags []string
			if err := json.Unmarshal([]byte(q.TagsJSON), &tags); err != nil {
				log.Printf("Failed to parse tags for question %s: %v", q.Name, err)
				tags = []string{}
			}

			questions = append(questions, structstypes.Question{
				QuestionID:   q.QuestionID,
				UserID:       q.UserID,
				Name:         q.Name,
				Date:         q.Date,
				Difficulty:   q.Difficulty,
				Tags:         tags,
				MinutesTaken: q.MinutesTaken,
				NeededHelp:   q.NeededHelp,
				Observation:  q.Observation,
			})
		}
	}

	return questions, nil
}
func FetchUserProfile(ctx context.Context, userID string) (*structstypes.UserProfile, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(profileTableName),
		Key: map[string]types.AttributeValue{
			"user_id": &types.AttributeValueMemberS{Value: userID},
		},
	}

	result, err := dynamoClient.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile from DynamoDB: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("user profile not found for user_id: %s", userID)
	}

	var profile structstypes.UserProfile
	if err := attributevalue.UnmarshalMap(result.Item, &profile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user profile: %w", err)
	}

	return &profile, nil
}

func FetchLatestMetrics(ctx context.Context, userID string) (*structstypes.UserMetrics, error) {
	input := &dynamodb.QueryInput{
		TableName:              aws.String(metricsTableName),
		KeyConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
		ScanIndexForward: aws.Bool(false),
		Limit:            aws.Int32(1),
	}

	result, err := dynamoClient.Query(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to query user metrics: %w", err)
	}

	if len(result.Items) == 0 {
		return nil, nil
	}

	var metrics structstypes.UserMetrics
	if err := attributevalue.UnmarshalMap(result.Items[0], &metrics); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user metrics: %w", err)
	}

	return &metrics, nil
}
