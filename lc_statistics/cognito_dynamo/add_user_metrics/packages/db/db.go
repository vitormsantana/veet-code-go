package db

import (
	"time"

	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	structstypes "github.com/vitormsantana/veet-code-go/cognito_dynamo/add_user_metrics/packages/typesandstructs"
)

var Client *dynamodb.Client

const TableName = "hammocker_user_metrics_table"

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

func calculate_avg_solved_last_days(questions []structstypes.Question, days int) float64 {
	if len(questions) == 0 || days <= 0 {
		return 0
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	count := 0

	for _, q := range questions {
		parsed, err := time.Parse("2006-01-02", q.QuestionDate)
		if err != nil {
			continue
		}
		if parsed.After(cutoff) && q.CrackedExercise {
			count++
		}
	}

	return float64(count) / float64(days)
}

func calculate_avg_failed_last_days(questions []structstypes.Question, days int) float64 {
	if len(questions) == 0 || days <= 0 {
		return 0
	}

	cutoff := time.Now().AddDate(0, 0, -days)
	count := 0

	for _, q := range questions {
		parsed, err := time.Parse("2006-01-02", q.QuestionDate)
		if err != nil {
			continue
		}
		if parsed.After(cutoff) && !q.CrackedExercise {
			count++
		}
	}

	return float64(count) / float64(days)
}

func calculate_consistency_rate(questions []structstypes.Question) float64 {
	if len(questions) == 0 {
		return 0
	}

	activityDays := make(map[string]bool)
	cutoff := time.Now().AddDate(0, 0, -30)

	for _, q := range questions {
		parsed, err := time.Parse("2006-01-02", q.QuestionDate)
		if err != nil {
			continue
		}
		if parsed.After(cutoff) {
			day := parsed.Format("2006-01-02")
			activityDays[day] = true
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
		parsed, err := time.Parse("2006-01-02", q.QuestionDate)
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
