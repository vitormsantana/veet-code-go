package db

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	structstypes "github.com/vitormsantana/veet-code-go/cognito_dynamo/add_user_metrics/packages/typesandstructs"
	"go.uber.org/zap"
)

var (
	Client *dynamodb.Client
	logger *zap.Logger
)

var questionDateLayouts = []string{
	"2006-01-02",
	time.RFC3339,
	"02/01/2006",
}

const (
	metricsTableName  = "hammocker_user_metrics_table"
	questionTableName = "veet_code_questions_table"
)

func Init() error {
	if Client != nil {
		return nil
	}

	if logger == nil {
		l, err := zap.NewProduction()
		if err != nil {
			return fmt.Errorf("failed to initialize zap logger: %w", err)
		}
		logger = l
		logger.Info("zap logger initialized for add_user_metrics db package")
	}

	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("sa-east-1"))
	if err != nil {
		return fmt.Errorf("unable to load AWS SDK config: %w", err)
	}
	Client = dynamodb.NewFromConfig(cfg)
	return nil
}

func calculate_avg_minutes_per_tag(questions []structstypes.Question) map[string]float64 {
	if len(questions) == 0 {
		return map[string]float64{}
	}

	tagTotals := make(map[string]float64)
	tagCounts := make(map[string]int)

	for _, q := range questions {
		if q.MinutesTaken <= 0 || len(q.QuestionTags) == 0 {
			continue
		}
		for _, tag := range q.QuestionTags {
			tagTotals[tag] += float64(q.MinutesTaken)
			tagCounts[tag]++
		}
	}

	avgMinutes := make(map[string]float64)
	for tag, total := range tagTotals {
		if tagCounts[tag] > 0 {
			avgMinutes[tag] = total / float64(tagCounts[tag])
		}
	}
	return avgMinutes
}

func calculate_help_rate_per_tag(questions []structstypes.Question) map[string]float64 {
	if len(questions) == 0 {
		return map[string]float64{}
	}

	totalHelpsPerTag := make(map[string]int)
	totalTimesEachTagAppeared := make(map[string]int)

	for _, q := range questions {
		for _, tag := range q.QuestionTags {
			if q.NeededHelp {
				totalHelpsPerTag[tag]++
			}
			totalTimesEachTagAppeared[tag]++
		}
	}

	helpRatePerTag := make(map[string]float64)
	for tag, total := range totalTimesEachTagAppeared {
		if total > 0 {
			helpRatePerTag[tag] = float64(totalHelpsPerTag[tag]) / float64(total)
		}
	}
	return helpRatePerTag
}

func calculate_solved_per_tag(questions []structstypes.Question) map[string]int {
	solvedPerTag := make(map[string]int)
	for _, q := range questions {
		for _, tag := range q.QuestionTags {
			solvedPerTag[tag]++
		}
	}
	return solvedPerTag
}

func calculate_failed_per_tag(questions []structstypes.Question) map[string]int {
	failedPerTag := make(map[string]int)
	for _, q := range questions {
		if !q.CrackedExercise {
			for _, tag := range q.QuestionTags {
				failedPerTag[tag]++
			}
		}
	}
	return failedPerTag
}

func parseQuestionDate(dateStr string) (time.Time, error) {
	var lastErr error
	for _, layout := range questionDateLayouts {
		if layout == "" {
			continue
		}
		parsed, err := time.ParseInLocation(layout, dateStr, time.UTC)
		if err == nil {
			return parsed, nil
		}
		lastErr = err
	}
	return time.Time{}, fmt.Errorf("unable to parse date %q: %w", dateStr, lastErr)
}

func computeWindowStats(questions []structstypes.Question, days int) structstypes.WindowStats {
	stats := structstypes.WindowStats{Days: days}
	if len(questions) == 0 || days <= 0 {
		return stats
	}

	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, -days)

	for _, q := range questions {
		parsed, err := parseQuestionDate(q.QuestionDate)
		if err != nil {
			stats.ParseErrorQuestionIDs = append(stats.ParseErrorQuestionIDs, q.QuestionID)
			stats.ParseErrorQuestionNotes = append(stats.ParseErrorQuestionNotes, fmt.Sprintf("%s:%s", q.QuestionID, q.QuestionDate))
			if logger != nil {
				logger.Warn("failed to parse question date for window stats",
					zap.String("question_id", q.QuestionID),
					zap.String("question_date", q.QuestionDate),
					zap.Int("window_days", days),
					zap.Error(err),
				)
			}
			continue
		}

		if parsed.After(cutoff) {
			stats.TotalConsidered++
			stats.ConsideredIDs = append(stats.ConsideredIDs, q.QuestionID)
			stats.ConsideredDetails = append(stats.ConsideredDetails,
				fmt.Sprintf("%s:%s:%t", q.QuestionID, q.QuestionDate, q.CrackedExercise))
			if q.CrackedExercise {
				stats.SolvedCount++
				stats.SolvedIDs = append(stats.SolvedIDs, q.QuestionID)
			} else {
				stats.FailedCount++
				stats.FailedIDs = append(stats.FailedIDs, q.QuestionID)
			}
		} else {
			stats.OutsideWindowIDs = append(stats.OutsideWindowIDs, q.QuestionID)
			stats.OutsideWindowDetails = append(stats.OutsideWindowDetails,
				fmt.Sprintf("%s:%s", q.QuestionID, q.QuestionDate))
		}
	}

	return stats
}

func logWindowStats(ws structstypes.WindowStats, windowLabel string) {
	if logger == nil {
		return
	}

	logger.Info("calculated window stats",
		zap.String("window", windowLabel),
		zap.Int("days", ws.Days),
		zap.Int("total_attempts", ws.TotalConsidered),
		zap.Int("solved_count", ws.SolvedCount),
		zap.Int("failed_count", ws.FailedCount),
		zap.Float64("success_rate", ws.SuccessRate()),
		zap.Float64("failure_rate", ws.FailureRate()),
		zap.Strings("considered_question_ids", ws.ConsideredIDs),
		zap.Strings("solved_question_ids", ws.SolvedIDs),
		zap.Strings("failed_question_ids", ws.FailedIDs),
		zap.Strings("outside_window_ids", ws.OutsideWindowIDs),
		zap.Strings("parse_error_question_ids", ws.ParseErrorQuestionIDs),
	)
}

func calculate_consistency_rate(questions []structstypes.Question) float64 {
	if len(questions) == 0 {
		return 0
	}

	activityDays := make(map[string]bool)
	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, -30)

	for _, q := range questions {
		parsed, err := parseQuestionDate(q.QuestionDate)
		if err != nil {
			continue
		}
		if parsed.After(cutoff) {
			activityDays[parsed.Format("2006-01-02")] = true
		}
	}

	activeDays := len(activityDays)
	return float64(activeDays) / 30.0
}

func calculate_last_activity_days_ago(questions []structstypes.Question) int {
	if len(questions) == 0 {
		return -1
	}

	latest := time.Time{}
	for _, q := range questions {
		parsed, err := parseQuestionDate(q.QuestionDate)
		if err != nil {
			continue
		}
		if parsed.After(latest) {
			latest = parsed
		}
	}

	if latest.IsZero() {
		return -1
	}
	return int(time.Since(latest).Hours() / 24)
}
