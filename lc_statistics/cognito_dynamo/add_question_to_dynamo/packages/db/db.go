package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_question_to_dynamo/packages/typesandstructs"
)

var Client *dynamodb.Client

const TableName = "veet_code_questions_table"

func Init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("sa-east-1"))
	if err != nil {
		panic(fmt.Sprintf("Unable to load AWS SDK config: %v", err))
	}
	Client = dynamodb.NewFromConfig(cfg)
}

func PutItem(userID string, request typesandstructs.Request) error {
	questionID := fmt.Sprintf("%d", time.Now().UnixNano())

	tagsJSON, err := json.Marshal(request.QuestionTags)
	if err != nil {
		return fmt.Errorf("failed to marshal tags: %v", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item: map[string]types.AttributeValue{
			"user_id":              &types.AttributeValueMemberS{Value: userID},
			"question_id":          &types.AttributeValueMemberS{Value: questionID},
			"question_name":        &types.AttributeValueMemberS{Value: request.QuestionName},
			"question_solved_date": &types.AttributeValueMemberS{Value: request.QuestionDate},
			"difficulty":           &types.AttributeValueMemberS{Value: request.QuestionDifficulty},
			"tags":                 &types.AttributeValueMemberS{Value: string(tagsJSON)},
			"minutes_taken":        &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", request.MinutesTaken)},
			"needed_help":          &types.AttributeValueMemberBOOL{Value: request.NeededHelp},
		},
	}

	_, err = Client.PutItem(context.TODO(), input)
	return err
}
