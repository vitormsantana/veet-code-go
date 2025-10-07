package generatestatistics

import (
	"log"
	"sort"
	"time"

	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_statistics_from_exercises_table/packages/structstypes"
)

const dateLayout = "02/01/2006"

func GenerateStatistics(questions []structstypes.Question) structstypes.Statistics {
	stats := structstypes.Statistics{
		QuestionsCrackedPerDay:            make(map[string]int),
		QuestionsCrackedPerDifficulty:     make(map[string]int),
		QuestionsCrackedPerTag:            make(map[string]int),
		OrderedQuestionsCrackedPerDay:     make([]structstypes.DayStatistic, 0),
		IncrementalQuestionsCrackedPerDay: make([]structstypes.DayStatistic, 0),
		TotalQuestionsCracked:             0,
	}

	for _, q := range questions {
		stats.QuestionsCrackedPerDay[q.Date]++
		stats.QuestionsCrackedPerDifficulty[q.Difficulty]++
		for _, tag := range q.Tags {
			stats.QuestionsCrackedPerTag[tag]++
		}
		stats.TotalQuestionsCracked++
	}

	sortedDates := getSortedDates(stats.QuestionsCrackedPerDay)

	runningTotal := 0
	for _, date := range sortedDates {
		count := stats.QuestionsCrackedPerDay[date]
		stats.OrderedQuestionsCrackedPerDay = append(stats.OrderedQuestionsCrackedPerDay, structstypes.DayStatistic{Date: date, Count: count})
		runningTotal += count
		stats.IncrementalQuestionsCrackedPerDay = append(stats.IncrementalQuestionsCrackedPerDay, structstypes.DayStatistic{Date: date, Count: runningTotal})
	}

	return stats
}

func getSortedDates(dateMap map[string]int) []string {
	dates := make([]string, 0, len(dateMap))
	for date := range dateMap {
		dates = append(dates, date)
	}

	sort.SliceStable(dates, func(i, j int) bool {
		d1, err1 := time.Parse(dateLayout, dates[i])
		d2, err2 := time.Parse(dateLayout, dates[j])
		if err1 != nil || err2 != nil {
			if err1 != nil {
				log.Printf("Failed to parse date %s: %v", dates[i], err1)
			}
			if err2 != nil {
				log.Printf("Failed to parse date %s: %v", dates[j], err2)
			}
			return dates[i] < dates[j]
		}
		return d1.Before(d2)
	})

	return dates
}
