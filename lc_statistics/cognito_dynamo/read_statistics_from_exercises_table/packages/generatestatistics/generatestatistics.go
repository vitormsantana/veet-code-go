package generatestatistics

import "github.com/vitormsantana/veet-code-go/cognito_dynamo/read_statistics_from_exercises_table/packages/structstypes"

func GenerateStatistics(questions []structstypes.Question) structstypes.Statistics {
	stats := structstypes.Statistics{
		QuestionsCrackedPerDay:        make(map[string]int),
		QuestionsCrackedPerDifficulty: make(map[string]int),
		QuestionsCrackedPerTag:        make(map[string]int),
		TotalQuestionsCracked:         0,
	}

	for _, q := range questions {
		stats.QuestionsCrackedPerDay[q.Date]++

		stats.QuestionsCrackedPerDifficulty[q.Difficulty]++

		for _, tag := range q.Tags {
			stats.QuestionsCrackedPerTag[tag]++
		}

		stats.TotalQuestionsCracked++
	}
	return stats
}
