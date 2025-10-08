package structstypes

type Question struct {
	QuestionID   string   `json:"question_id"`
	UserID       string   `json:"user_id"`
	Name         string   `json:"name"`
	Date         string   `json:"date"`
	Difficulty   string   `json:"difficulty"`
	Tags         []string `json:"tags"`
	MinutesTaken int      `json:"minutes_taken"`
	NeededHelp   bool     `json:"needed_help"`
	Observation  string   `json:"obs"`
}

type DayStatistic struct {
	Date  string `json:"date"`
	Count int    `json:"count"`
}

type Statistics struct {
	QuestionsCrackedPerDay            map[string]int `json:"questionsCrackedPerDay"`
	OrderedQuestionsCrackedPerDay     []DayStatistic `json:"orderedQuestionsCrackedPerDay"`
	IncrementalQuestionsCrackedPerDay []DayStatistic `json:"incrementalQuestionsCrackedPerDay"`
	QuestionsCrackedPerDifficulty     map[string]int `json:"questionsCrackedPerDifficulty"`
	QuestionsCrackedPerTag            map[string]int `json:"questionsCrackedPerTag"`
	TotalQuestionsCracked             int            `json:"totalQuestionsCracked"`
}
