package db

import (
	"context"
	"fmt"
	"sort"
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
		count := tagCounts[tag]
		if count > 0 {
			avgMinutes[tag] = total / float64(count)
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

func calculate_avg_solved_last_days(questions []structstypes.Question, days int) float64 {
	if len(questions) == 0 || days <= 0 {
		if logger != nil {
			logger.Info("avg solved last days skipped",
				zap.Int("total_questions", len(questions)),
				zap.Int("days", days))
		}
		return 0
	}

	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, -days)
	count := 0
	considered := 0
	var (
		consideredIDs            []string
		consideredDetails        []string
		solvedIDs                []string
		notSolvedIDs             []string
		outsideWindowIDs         []string
		outsideWindowDetails     []string
		parseErrorQuestionIDs    []string
		parseErrorQuestionDetail []string
	)

	for _, q := range questions {
		parsed, err := parseQuestionDate(q.QuestionDate)
		if err != nil {
			parseErrorQuestionIDs = append(parseErrorQuestionIDs, q.QuestionID)
			parseErrorQuestionDetail = append(parseErrorQuestionDetail, fmt.Sprintf("%s:%s", q.QuestionID, q.QuestionDate))
			if logger != nil {
				logger.Warn("failed to parse question date for avg solved calculation",
					zap.String("question_id", q.QuestionID),
					zap.String("question_date", q.QuestionDate),
					zap.Error(err),
				)
			}
			continue
		}
		if parsed.After(cutoff) {
			considered++
			consideredIDs = append(consideredIDs, q.QuestionID)
			consideredDetails = append(consideredDetails, fmt.Sprintf("%s:%s:%t", q.QuestionID, q.QuestionDate, q.CrackedExercise))
			if q.CrackedExercise {
				count++
				solvedIDs = append(solvedIDs, q.QuestionID)
			} else {
				notSolvedIDs = append(notSolvedIDs, q.QuestionID)
			}
		} else {
			outsideWindowIDs = append(outsideWindowIDs, q.QuestionID)
			outsideWindowDetails = append(outsideWindowDetails, fmt.Sprintf("%s:%s", q.QuestionID, q.QuestionDate))
		}
	}

	avg := float64(count) / float64(days)
	if logger != nil {
		logger.Info("calculated avg solved last days",
			zap.Int("days", days),
			zap.Float64("average", avg),
			zap.Int("solved_count", count),
			zap.Int("considered_questions", considered),
			zap.Time("cutoff", cutoff),
			zap.Strings("considered_question_ids", consideredIDs),
			zap.Strings("considered_details", consideredDetails),
			zap.Strings("solved_question_ids", solvedIDs),
			zap.Strings("not_solved_question_ids", notSolvedIDs),
			zap.Strings("outside_window_ids", outsideWindowIDs),
			zap.Strings("outside_window_details", outsideWindowDetails),
			zap.Strings("parse_error_question_ids", parseErrorQuestionIDs),
			zap.Strings("parse_error_question_details", parseErrorQuestionDetail),
		)
	}

	return avg
}

func calculate_avg_failed_last_days(questions []structstypes.Question, days int) float64 {
	if len(questions) == 0 || days <= 0 {
		if logger != nil {
			logger.Info("avg failed last days skipped",
				zap.Int("total_questions", len(questions)),
				zap.Int("days", days))
		}
		return 0
	}

	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, -days)
	count := 0
	considered := 0
	var (
		consideredIDs            []string
		consideredDetails        []string
		failedIDs                []string
		notFailedIDs             []string
		outsideWindowIDs         []string
		outsideWindowDetails     []string
		parseErrorQuestionIDs    []string
		parseErrorQuestionDetail []string
	)

	for _, q := range questions {
		parsed, err := parseQuestionDate(q.QuestionDate)
		if err != nil {
			parseErrorQuestionIDs = append(parseErrorQuestionIDs, q.QuestionID)
			parseErrorQuestionDetail = append(parseErrorQuestionDetail, fmt.Sprintf("%s:%s", q.QuestionID, q.QuestionDate))
			if logger != nil {
				logger.Warn("failed to parse question date for avg failed calculation",
					zap.String("question_id", q.QuestionID),
					zap.String("question_date", q.QuestionDate),
					zap.Error(err),
				)
			}
			continue
		}
		if parsed.After(cutoff) {
			considered++
			consideredIDs = append(consideredIDs, q.QuestionID)
			consideredDetails = append(consideredDetails, fmt.Sprintf("%s:%s:%t", q.QuestionID, q.QuestionDate, q.CrackedExercise))
			if !q.CrackedExercise {
				count++
				failedIDs = append(failedIDs, q.QuestionID)
			} else {
				notFailedIDs = append(notFailedIDs, q.QuestionID)
			}
		} else {
			outsideWindowIDs = append(outsideWindowIDs, q.QuestionID)
			outsideWindowDetails = append(outsideWindowDetails, fmt.Sprintf("%s:%s", q.QuestionID, q.QuestionDate))
		}
	}

	avg := float64(count) / float64(days)
	if logger != nil {
		logger.Info("calculated avg failed last days",
			zap.Int("days", days),
			zap.Float64("average", avg),
			zap.Int("failed_count", count),
			zap.Int("considered_questions", considered),
			zap.Time("cutoff", cutoff),
			zap.Strings("considered_question_ids", consideredIDs),
			zap.Strings("considered_details", consideredDetails),
			zap.Strings("failed_question_ids", failedIDs),
			zap.Strings("not_failed_question_ids", notFailedIDs),
			zap.Strings("outside_window_ids", outsideWindowIDs),
			zap.Strings("outside_window_details", outsideWindowDetails),
			zap.Strings("parse_error_question_ids", parseErrorQuestionIDs),
			zap.Strings("parse_error_question_details", parseErrorQuestionDetail),
		)
	}

	return avg
}

func calculate_consistency_rate(questions []structstypes.Question) float64 {
	if len(questions) == 0 {
		if logger != nil {
			logger.Info("consistency rate skipped", zap.Int("total_questions", len(questions)))
		}
		return 0
	}

	activityDays := make(map[string]bool)
	now := time.Now().UTC()
	cutoff := now.AddDate(0, 0, -30)

	for _, q := range questions {
		parsed, err := parseQuestionDate(q.QuestionDate)
		if err != nil {
			if logger != nil {
				logger.Warn("failed to parse question date for consistency calculation",
					zap.String("question_id", q.QuestionID),
					zap.String("question_date", q.QuestionDate),
					zap.Error(err),
				)
			}
			continue
		}
		if parsed.After(cutoff) {
			day := parsed.Format("2006-01-02")
			activityDays[day] = true
		}
	}

	activeDays := len(activityDays)
	rate := float64(activeDays) / 30.0
	if logger != nil {
		days := make([]string, 0, len(activityDays))
		for day := range activityDays {
			days = append(days, day)
		}
		sort.Strings(days)
		logger.Info("calculated consistency rate",
			zap.Float64("rate", rate),
			zap.Int("active_days", activeDays),
			zap.Time("cutoff", cutoff),
			zap.Strings("active_day_list", days),
			zap.Int("total_questions", len(questions)),
		)
	}

	return rate
}

func calculate_last_activity_days_ago(questions []structstypes.Question) int {
	if len(questions) == 0 {
		return -1
	}

	latest := time.Time{}
	for _, q := range questions {
		parsed, err := parseQuestionDate(q.QuestionDate)
		if err != nil {
			if logger != nil {
				logger.Warn("failed to parse question date for last activity calculation",
					zap.String("question_id", q.QuestionID),
					zap.String("question_date", q.QuestionDate),
					zap.Error(err),
				)
			}
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
