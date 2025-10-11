package db

import (
	"context"
	"fmt"
	"log"

	"go.uber.org/zap"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_user_profile_infos/packages/structstypes"
)

var (
	dynamoClient *dynamodb.Client
	logger       *zap.Logger
)

const tableName = "hammocker_user_profiles_table"

func init() {
	var err error
	logger, err = zap.NewProduction()
	if err != nil {
		log.Fatalf("Unable to initialize zap logger: %v", err)
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("sa-east-1"))
	if err != nil {
		logger.Fatal("Unable to load AWS SDK config", zap.Error(err))
	}
	dynamoClient = dynamodb.NewFromConfig(cfg)
	logger.Info("DynamoDB client initialized")
}

func FetchProfileInfos(ctx context.Context, userID string) (*structstypes.UserProfile, error) {
	logger.Info("Fetching user profile", zap.String("user_id", userID))

	if dynamoClient == nil {
		err := fmt.Errorf("dynamodb client not initialized")
		logger.Error("DynamoDB client not initialized", zap.Error(err))
		return nil, err
	}

	input := &dynamodb.GetItemInput{
		TableName: aws.String(tableName),
		Key: map[string]types.AttributeValue{
			"user_id": &types.AttributeValueMemberS{Value: userID},
		},
	}

	result, err := dynamoClient.GetItem(ctx, input)
	if err != nil {
		logger.Error("Failed to get user profile", zap.String("user_id", userID), zap.Error(err))
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	if result.Item == nil {
		logger.Warn("No user profile found", zap.String("user_id", userID))
		return nil, nil
	}

	var profile structstypes.UserProfile
	if err := attributevalue.UnmarshalMap(result.Item, &profile); err != nil {
		logger.Error("Failed to unmarshal user profile", zap.String("user_id", userID), zap.Error(err))
		return nil, fmt.Errorf("failed to unmarshal user profile: %w", err)
	}

	logger.Info("User profile fetched successfully", zap.String("user_id", userID))
	return &profile, nil
}
