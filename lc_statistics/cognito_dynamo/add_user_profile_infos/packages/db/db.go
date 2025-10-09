package db

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_user_profile_infos/packages/typesandstructs"
	"go.uber.org/zap"
)

var Client *dynamodb.Client
var logger *zap.SugaredLogger

const TableName = "hammocker_user_profiles_table"

func Init(l *zap.SugaredLogger) {
	logger = l
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("sa-east-1"))
	if err != nil {
		panic(fmt.Sprintf("Unable to load AWS SDK config: %v", err))
	}
	Client = dynamodb.NewFromConfig(cfg)
	if logger != nil {
		logger.Infow("DynamoDB client initialized", "table", TableName)
	}
}

func PutUserProfile(profile typesandstructs.UserProfile) error {
	if Client == nil {
		return fmt.Errorf("dynamodb client not initialized")
	}
	if profile.ProfileLastUpdated == "" {
		profile.ProfileLastUpdated = time.Now().UTC().Format(time.RFC3339)
	}

	var topicsJSON string
	if json.Valid([]byte(profile.TopicsFamiliarity)) {
		topicsJSON = profile.TopicsFamiliarity
	} else {
		topicBytes, err := json.Marshal(profile.TopicsFamiliarity)
		if err != nil {
			return fmt.Errorf("failed to marshal topics familiarity: %v", err)
		}
		topicsJSON = string(topicBytes)
	}

	item := map[string]types.AttributeValue{
		"user_id":                  &types.AttributeValueMemberS{Value: profile.UserID},
		"target_companies":         &types.AttributeValueMemberS{Value: profile.TargetCompanies},
		"desired_role":             &types.AttributeValueMemberS{Value: profile.DesiredRole},
		"desired_level":            &types.AttributeValueMemberS{Value: profile.DesiredLevel},
		"years_of_experience":      &types.AttributeValueMemberN{Value: fmt.Sprintf("%d", profile.YearsOfExperience)},
		"main_stack":               &types.AttributeValueMemberS{Value: profile.MainStack},
		"leetcode_experience":      &types.AttributeValueMemberS{Value: profile.LeetCodeExperience},
		"interview_experience":     &types.AttributeValueMemberS{Value: profile.InterviewExperience},
		"country_target":           &types.AttributeValueMemberS{Value: profile.CountryTarget},
		"scheduled_interview":      &types.AttributeValueMemberS{Value: profile.ScheduledInterview},
		"topics_familiarity":       &types.AttributeValueMemberS{Value: topicsJSON},
		"profile_last_updated":     &types.AttributeValueMemberS{Value: profile.ProfileLastUpdated},
		"ai_personalization_score": &types.AttributeValueMemberN{Value: fmt.Sprintf("%.2f", profile.AIPersonalizationScore)},
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(TableName),
		Item:      item,
	}

	_, err := Client.PutItem(context.TODO(), input)
	if err != nil {
		if logger != nil {
			logger.Errorw("Failed to put user profile", "userID", profile.UserID, "error", err)
		}
		return fmt.Errorf("failed to put user profile: %v", err)
	}
	if logger != nil {
		logger.Infow("User profile stored", "userID", profile.UserID)
	}
	return nil
}

func GetUserProfile(userID string) (*typesandstructs.UserProfile, error) {
	if Client == nil {
		return nil, fmt.Errorf("dynamodb client not initialized")
	}
	if logger != nil {
		logger.Infow("Fetching user profile", "userID", userID)
	}
	input := &dynamodb.GetItemInput{
		TableName: aws.String(TableName),
		Key: map[string]types.AttributeValue{
			"user_id": &types.AttributeValueMemberS{Value: userID},
		},
	}

	result, err := Client.GetItem(context.TODO(), input)
	if err != nil {
		if logger != nil {
			logger.Errorw("Failed to get user profile", "userID", userID, "error", err)
		}
		return nil, fmt.Errorf("failed to get user profile: %v", err)
	}

	if result.Item == nil {
		if logger != nil {
			logger.Infow("No existing user profile", "userID", userID)
		}
		return nil, nil
	}

	var profile typesandstructs.UserProfile
	err = attributevalue.UnmarshalMap(result.Item, &profile)
	if err != nil {
		if logger != nil {
			logger.Errorw("Failed to unmarshal user profile", "userID", userID, "error", err)
		}
		return nil, fmt.Errorf("failed to unmarshal user profile: %v", err)
	}

	return &profile, nil
}
