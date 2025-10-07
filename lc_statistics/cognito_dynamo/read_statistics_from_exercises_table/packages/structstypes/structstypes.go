package structstypes

type Question struct {
	Name       string   `dynamodbav:"question_name"`
	Date       string   `dynamodbav:"question_solved_date"`
	Difficulty string   `dynamodbav:"difficulty"`
	Tags       []string `json:"tags"`
}

type Statistics struct {
	QuestionsCrackedPerDay        map[string]int `json:"questionsCrackedPerDay"`
	QuestionsCrackedPerDifficulty map[string]int `json:"questionsCrackedPerDifficulty"`
	QuestionsCrackedPerTag        map[string]int `json:"questionsCrackedPerTag"`
	TotalQuestionsCracked         int            `json:"totalQuestionsCracked"`
}
