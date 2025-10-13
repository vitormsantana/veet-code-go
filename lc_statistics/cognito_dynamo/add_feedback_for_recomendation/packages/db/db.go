package db

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_feedback_for_recomendation/packages/typesandstructs"
)

var Client *dynamodb.Client

const TableName = "hammocker_user_feedback_table"

func Init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("sa-east-1"))
	if err != nil {
		panic(fmt.Sprintf("Unable to load AWS SDK config: %v", err))
	}
	Client = dynamodb.NewFromConfig(cfg)
}

func PutFeedback(userID string, request typesandstructs.Feedback) error {
	feedbackID := uuid.NewString()
	feedbackTimestamp := time.Now().UTC().Format(time.RFC3339)

	item := map[string]types.AttributeValue{
		"feedback_id":        &types.AttributeValueMemberS{Value: feedbackID},
		"user_id":            &types.AttributeValueMemberS{Value: userID},
		"recomendation_id":   &types.AttributeValueMemberS{Value: request.RecomendationID},
		"feedback_value":     &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", request.FeedbackValue)},
		"feedback_timestamp": &types.AttributeValueMemberS{Value: feedbackTimestamp},
	}

	if comment := request.FeedbackComment; comment != "" {
		item["feedback_comment"] = &types.AttributeValueMemberS{Value: comment}
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      item,
	}

	_, err := Client.PutItem(context.TODO(), input)
	return err
}
