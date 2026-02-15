package db

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/google/uuid"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_feedback_for_recomendation/packages/typesandstructs"
)

var Client *dynamodb.Client

const (
	TableName                      = "hammocker_user_feedback_table"
	recommendationsTableName       = "hammocker_recommendations_table"
	profileTableName               = "hammocker_user_profiles_table"
	feedbackSummaryTableName       = "hammocker_last_feedback_summaries_table"
	maxScanPageSize          int32 = 200
)

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

func FetchUserProfile(ctx context.Context, userID string) (*typesandstructs.UserProfile, error) {
	input := &dynamodb.GetItemInput{
		TableName: aws.String(profileTableName),
		Key: map[string]types.AttributeValue{
			"user_id": &types.AttributeValueMemberS{Value: userID},
		},
	}

	result, err := Client.GetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to get user profile: %w", err)
	}

	if result.Item == nil {
		return nil, fmt.Errorf("user profile not found for user_id: %s", userID)
	}

	var profile typesandstructs.UserProfile
	if err := attributevalue.UnmarshalMap(result.Item, &profile); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user profile: %w", err)
	}

	return &profile, nil
}

func FetchRecentFeedbacks(ctx context.Context, userID string, limit int) ([]typesandstructs.StoredFeedback, error) {
	if limit <= 0 {
		limit = 5
	}

	input := &dynamodb.ScanInput{
		TableName:        aws.String(TableName),
		FilterExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
		Limit: aws.Int32(maxScanPageSize),
	}

	paginator := dynamodb.NewScanPaginator(Client, input)
	var collected []typesandstructs.StoredFeedback

	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to scan feedbacks: %w", err)
		}

		var pageFeedbacks []typesandstructs.StoredFeedback
		if err := attributevalue.UnmarshalListOfMaps(page.Items, &pageFeedbacks); err != nil {
			return nil, fmt.Errorf("failed to unmarshal feedback page: %w", err)
		}

		collected = append(collected, pageFeedbacks...)
		if len(collected) >= limit {
			break
		}
	}

	if len(collected) == 0 {
		return nil, nil
	}

	// recent first
	sort.Slice(collected, func(i, j int) bool {
		return collected[i].FeedbackTimestamp > collected[j].FeedbackTimestamp
	})

	if len(collected) > limit {
		collected = collected[:limit]
	}

	return collected, nil
}

func FetchRecommendationBatch(ctx context.Context, recIDs []string) ([]typesandstructs.RecommendationRecord, error) {
	if len(recIDs) == 0 {
		return nil, nil
	}

	uniq := make(map[string]struct{})
	keys := make([]map[string]types.AttributeValue, 0, len(recIDs))
	for _, id := range recIDs {
		if strings.TrimSpace(id) == "" {
			continue
		}
		if _, exists := uniq[id]; exists {
			continue
		}
		uniq[id] = struct{}{}
		keys = append(keys, map[string]types.AttributeValue{
			"recommendation_id": &types.AttributeValueMemberS{Value: id},
		})
	}

	if len(keys) == 0 {
		return nil, nil
	}

	input := &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			recommendationsTableName: {
				Keys: keys,
			},
		},
	}

	resp, err := Client.BatchGetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to batch get recommendations: %w", err)
	}

	items := resp.Responses[recommendationsTableName]
	if len(items) == 0 {
		return nil, nil
	}

	var records []typesandstructs.RecommendationRecord
	if err := attributevalue.UnmarshalListOfMaps(items, &records); err != nil {
		return nil, fmt.Errorf("failed to unmarshal recommendations: %w", err)
	}

	return records, nil
}

func SaveProcessedFeedbackSummary(ctx context.Context, summary *typesandstructs.ProcessedFeedbackSummary) error {
	if summary == nil {
		return fmt.Errorf("processed feedback summary is nil")
	}

	if summary.SummaryID == "" {
		summary.SummaryID = uuid.NewString()
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if summary.GeneratedAtUTC == "" {
		summary.GeneratedAtUTC = now
	}
	if summary.AnalyzedAtUTC == "" {
		summary.AnalyzedAtUTC = now
	}

	item, err := attributevalue.MarshalMap(summary)
	if err != nil {
		return fmt.Errorf("failed to marshal processed summary: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(feedbackSummaryTableName),
		Item:      item,
	}

	if _, err := Client.PutItem(ctx, input); err != nil {
		return fmt.Errorf("failed to persist processed summary: %w", err)
	}

	return nil
}
