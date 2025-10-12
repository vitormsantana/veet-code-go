package db

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	structstypes "github.com/vitormsantana/veet-code-go/cognito_dynamo/add_user_metrics/packages/typesandstructs"
)

func FetchQuestions(ctx context.Context, userID string) ([]structstypes.Question, error) {
	var questions []structstypes.Question

	if Client == nil {
		return nil, fmt.Errorf("dynamodb client not initialized")
	}

	input := &dynamodb.QueryInput{
		TableName:              aws.String(questionTableName),
		KeyConditionExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
	}

	paginator := dynamodb.NewQueryPaginator(Client, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to query DynamoDB: %w", err)
		}

		//for _, item := range page.Items {
		//	log.Printf("Raw item: %v", item)
		//}

		var pageQuestions []struct {
			UserID          string `dynamodbav:"user_id"`
			QuestionID      string `dynamodbav:"question_id"`
			Name            string `dynamodbav:"question_name"`
			Date            string `dynamodbav:"question_solved_date"`
			Difficulty      string `dynamodbav:"difficulty"`
			TagsJSON        string `dynamodbav:"tags"`
			MinutesTaken    int    `dynamodbav:"minutes_taken"`
			NeededHelp      bool   `dynamodbav:"needed_help"`
			Observation     string `dynamodbav:"obs"`
			CrackedExercise bool   `dynamodbav:"cracked_exercise"`
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
				QuestionID:      q.QuestionID,
				UserID:          q.UserID,
				QuestionName:    q.Name,
				QuestionDate:    q.Date,
				QuestionTags:    tags,
				MinutesTaken:    q.MinutesTaken,
				NeededHelp:      q.NeededHelp,
				Observation:     q.Observation,
				CrackedExercise: q.CrackedExercise,
			})
		}
	}

	return questions, nil
}
