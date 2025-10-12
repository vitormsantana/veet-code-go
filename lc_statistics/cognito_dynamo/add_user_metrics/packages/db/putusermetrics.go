package db

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	structstypes "github.com/vitormsantana/veet-code-go/cognito_dynamo/add_user_metrics/packages/typesandstructs"
	"go.uber.org/zap"
)

func PutUserMetrics(ctx context.Context, userID string, req structstypes.Request, questions []structstypes.Question) (*structstypes.UserMetrics, error) {
	if Client == nil {
		return nil, fmt.Errorf("dynamodb client not initialized")
	}

	metricsDate := req.MetricsDate
	if metricsDate == "" {
		metricsDate = time.Now().UTC().Format(time.RFC3339)
	}

	shortWindow := req.ShortWindowDays
	if shortWindow <= 0 {
		shortWindow = 7
	}

	longWindow := req.LongWindowDays
	if longWindow <= 0 {
		longWindow = 30
	}

	metrics := &structstypes.UserMetrics{
		UserID:                 userID,
		Date:                   metricsDate,
		ShortWindowDays:        shortWindow,
		LongWindowDays:         longWindow,
		AvgMinutesPerTag:       calculate_avg_minutes_per_tag(questions),
		HelpRatePerTag:         calculate_help_rate_per_tag(questions),
		SolvedPerTag:           calculate_solved_per_tag(questions),
		FailedPerTag:           calculate_failed_per_tag(questions),
		AvgSolvedLastShortDays: calculate_avg_solved_last_days(questions, shortWindow),
		AvgSolvedLastLongDays:  calculate_avg_solved_last_days(questions, longWindow),
		AvgFailedLastShortDays: calculate_avg_failed_last_days(questions, shortWindow),
		AvgFailedLastLongDays:  calculate_avg_failed_last_days(questions, longWindow),
		ConsistencyRate:        calculate_consistency_rate(questions),
		LastActivityDaysAgo:    calculate_last_activity_days_ago(questions),
		TotalQuestionsAnalyzed: len(questions),
		CalculatedAtUTC:        time.Now().UTC().Format(time.RFC3339),
	}

	if logger != nil {
		logger.Info("calculated user metrics",
			zap.String("user_id", userID),
			zap.Int("short_window_days", shortWindow),
			zap.Int("long_window_days", longWindow),
			zap.Int("total_questions", len(questions)),
			zap.Float64("avg_solved_short", metrics.AvgSolvedLastShortDays),
			zap.Float64("avg_solved_long", metrics.AvgSolvedLastLongDays),
			zap.Float64("avg_failed_short", metrics.AvgFailedLastShortDays),
			zap.Float64("avg_failed_long", metrics.AvgFailedLastLongDays),
			zap.Float64("consistency_rate", metrics.ConsistencyRate),
		)
	}

	item, err := attributevalue.MarshalMap(metrics)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal metrics: %w", err)
	}

	input := &dynamodb.PutItemInput{
		TableName: aws.String(metricsTableName),
		Item:      item,
	}

	if _, err := Client.PutItem(ctx, input); err != nil {
		return nil, fmt.Errorf("failed to store metrics: %w", err)
	}

	if logger != nil {
		logger.Info("stored user metrics in dynamodb",
			zap.String("user_id", userID),
			zap.String("metrics_date", metrics.Date),
			zap.String("calculated_at", metrics.CalculatedAtUTC),
		)
	}

	return metrics, nil
}
