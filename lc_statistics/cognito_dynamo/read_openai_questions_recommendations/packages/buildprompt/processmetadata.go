package buildprompt

import (
	"context"
	"fmt"
	"log"
	"sort"
	"strings"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_openai_questions_recommendations/packages/structstypes"
)

const (
	feedbacksTableName       = "hammocker_user_feedback_table"
	recommendationsTableName = "hammocker_recommendations_table"
)

var dynamoClient *dynamodb.Client

func init() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("sa-east-1"))
	if err != nil {
		log.Fatalf("Unable to load AWS SDK config: %v", err)
	}
	dynamoClient = dynamodb.NewFromConfig(cfg)
}

func FetchRecentFeedbacks(ctx context.Context, userID string, batchSize int) ([]structstypes.Feedback, error) {
	var recentFeedbacks []structstypes.Feedback

	input := &dynamodb.ScanInput{
		TableName:        aws.String(feedbacksTableName),
		FilterExpression: aws.String("user_id = :uid"),
		ExpressionAttributeValues: map[string]types.AttributeValue{
			":uid": &types.AttributeValueMemberS{Value: userID},
		},
	}

	paginator := dynamodb.NewScanPaginator(dynamoClient, input)
	for paginator.HasMorePages() {
		page, err := paginator.NextPage(ctx)
		if err != nil {
			return nil, fmt.Errorf("failed to scan DynamoDB: %w", err)
		}

		for _, item := range page.Items {
			log.Printf("Raw item: %v", item)
		}

		var pageFeedbacks []struct {
			FeedbackID        string `json:"feedback_id" dynamodbav:"feedback_id"`
			FeedbackTimestamp string `json:"feedback_timestamp" dynamodbav:"feedback_timestamp"`
			FeedbackValue     int    `json:"feedback_value" dynamodbav:"feedback_value"`
			FeedbackComment   string `json:"feedback_comment,omitempty" dynamodbav:"feedback_comment,omitempty"`
			RecomendationID   string `json:"recomendation_id" dynamodbav:"recomendation_id"`
			UserID            string `json:"user_id" dynamodbav:"user_id"`
		}
		err = attributevalue.UnmarshalListOfMaps(page.Items, &pageFeedbacks)
		if err != nil {
			return nil, fmt.Errorf("failed to unmarshal DynamoDB items: %w", err)
		}

		for _, f := range pageFeedbacks {
			recentFeedbacks = append(recentFeedbacks, structstypes.Feedback{
				FeedbackID:        f.FeedbackID,
				FeedbackTimestamp: f.FeedbackTimestamp,
				FeedbackValue:     f.FeedbackValue,
				FeedbackComment:   f.FeedbackComment,
				RecomendationID:   f.RecomendationID,
				UserID:            f.UserID,
			})
		}

		if len(recentFeedbacks) >= batchSize {
			break
		}
	}

	sort.Slice(recentFeedbacks, func(i, j int) bool {
		return recentFeedbacks[i].FeedbackTimestamp > recentFeedbacks[j].FeedbackTimestamp
	})

	if len(recentFeedbacks) > batchSize {
		recentFeedbacks = recentFeedbacks[:batchSize]
	}

	return recentFeedbacks, nil
}

func FetchRecommendationBatch(ctx context.Context, recIDs []string) ([]structstypes.RecommendationRecord, error) {
	if len(recIDs) == 0 {
		return nil, nil
	}

	keys := make([]map[string]types.AttributeValue, 0, len(recIDs))
	for _, id := range recIDs {
		keys = append(keys, map[string]types.AttributeValue{
			"recommendation_id": &types.AttributeValueMemberS{Value: id},
		})
	}

	input := &dynamodb.BatchGetItemInput{
		RequestItems: map[string]types.KeysAndAttributes{
			recommendationsTableName: {
				Keys: keys,
			},
		},
	}

	resp, err := dynamoClient.BatchGetItem(ctx, input)
	if err != nil {
		return nil, fmt.Errorf("failed to batch get recommendations: %w", err)
	}

	var records []structstypes.RecommendationRecord
	items := resp.Responses[recommendationsTableName]
	if len(items) == 0 {
		log.Printf("No recommendations found for IDs: %v", recIDs)
		return records, nil
	}

	err = attributevalue.UnmarshalListOfMaps(items, &records)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal recommendations: %w", err)
	}

	return records, nil
}

func GetRecentFeedbacksWithRecommendations(ctx context.Context, userID string, batchSize int) ([]struct {
	Feedback       structstypes.Feedback
	Recommendation *structstypes.RecommendationRecord
}, error) {
	feedbacks, err := FetchRecentFeedbacks(ctx, userID, batchSize)
	if err != nil {
		return nil, err
	}

	recIDs := make([]string, 0, len(feedbacks))
	for _, f := range feedbacks {
		recIDs = append(recIDs, f.RecomendationID)
	}

	recs, err := FetchRecommendationBatch(ctx, recIDs)
	if err != nil {
		return nil, err
	}

	recMap := make(map[string]*structstypes.RecommendationRecord)
	for i := range recs {
		recMap[recs[i].RecommendationID] = &recs[i]
	}

	var combined []struct {
		Feedback       structstypes.Feedback
		Recommendation *structstypes.RecommendationRecord
	}

	for _, f := range feedbacks {
		combined = append(combined, struct {
			Feedback       structstypes.Feedback
			Recommendation *structstypes.RecommendationRecord
		}{
			Feedback:       f,
			Recommendation: recMap[f.RecomendationID],
		})
	}

	return combined, nil
}

func SummarizeFeedbackContext(combined []struct {
	Feedback       structstypes.Feedback
	Recommendation *structstypes.RecommendationRecord
}) string {
	if len(combined) == 0 {
		return "No user feedback context available yet."
	}

	var positives, negatives int
	var comments []string
	var focusAreas []string
	var tones []string

	for _, pair := range combined {
		if pair.Feedback.FeedbackValue > 0 {
			positives++
		} else {
			negatives++
		}

		if c := strings.TrimSpace(pair.Feedback.FeedbackComment); c != "" {
			if len(c) > 120 {
				c = c[:117] + "..."
			}
			comments = append(comments, c)
		}

		if pair.Recommendation != nil {
			if len(pair.Recommendation.Metadata.FocusTags) > 0 {
				focusAreas = append(focusAreas, pair.Recommendation.Metadata.FocusTags...)
			}
			if pair.Recommendation.Metadata.Tone != "" {
				tones = append(tones, pair.Recommendation.Metadata.Tone)
			}
		}
	}

	focusAreas = uniqueStrings(focusAreas)
	tones = uniqueStrings(tones)

	summary := fmt.Sprintf(
		"Recent Feedback Summary (last %d sessions): %d positive, %d negative.\n"+
			"Frequent focus areas: %s. Preferred tone trends: %s.\n",
		len(combined),
		positives, negatives,
		strings.Join(focusAreas, ", "),
		strings.Join(tones, ", "),
	)

	if len(comments) > 0 {
		summary += "Sample user remarks: \"" + strings.Join(comments[:min(3, len(comments))], "\" | \"") + "\".\n"
	}

	return summary
}

func FormatProcessedFeedbackSummary(summary *structstypes.ProcessedFeedbackSummary) string {
	if summary == nil {
		return "No recent feedback context available."
	}

	var sb strings.Builder
	sb.WriteString("Processed Feedback Summary\n")
	if summary.GeneratedAtUTC != "" {
		sb.WriteString(fmt.Sprintf("Generated at: %s\n", summary.GeneratedAtUTC))
	}
	if summary.AnalyzedAtUTC != "" {
		sb.WriteString(fmt.Sprintf("Analyzed at: %s\n", summary.AnalyzedAtUTC))
	}
	sb.WriteString(fmt.Sprintf("Feedback records considered: %d\n", summary.FeedbackCount))

	if summary.StructuredSummary.Narrative != "" {
		sb.WriteString("\nNarrative:\n")
		sb.WriteString(summary.StructuredSummary.Narrative)
		sb.WriteString("\n")
	}

	if len(summary.StructuredSummary.PriorityTopics) > 0 {
		sb.WriteString("Topics to emphasize: ")
		sb.WriteString(strings.Join(summary.StructuredSummary.PriorityTopics, ", "))
		sb.WriteString("\n")
	}

	if len(summary.StructuredSummary.AvoidTopics) > 0 {
		sb.WriteString("Topics to avoid for now: ")
		sb.WriteString(strings.Join(summary.StructuredSummary.AvoidTopics, ", "))
		sb.WriteString("\n")
	}

	if len(summary.StructuredSummary.ToneGuidance) > 0 {
		sb.WriteString("Preferred tone: ")
		sb.WriteString(strings.Join(summary.StructuredSummary.ToneGuidance, ", "))
		sb.WriteString("\n")
	}

	if len(summary.StructuredSummary.CoachingActions) > 0 {
		sb.WriteString("Coaching reminders: ")
		sb.WriteString(strings.Join(summary.StructuredSummary.CoachingActions, ", "))
		sb.WriteString("\n")
	}

	if summary.StructuredSummary.Confidence != "" {
		sb.WriteString("Confidence in summary: ")
		sb.WriteString(strings.Title(summary.StructuredSummary.Confidence))
		sb.WriteString("\n")
	}

	return strings.TrimSpace(sb.String())
}

func uniqueStrings(in []string) []string {
	m := make(map[string]bool)
	var out []string
	for _, s := range in {
		if !m[s] {
			m[s] = true
			out = append(out, s)
		}
	}
	return out
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
