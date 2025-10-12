package structstypes

type Question struct {
	QuestionName       string   `json:"name"`
	QuestionDate       string   `json:"date"`
	QuestionDifficulty string   `json:"difficulty"`
	QuestionTags       []string `json:"tags"`
	MinutesTaken       int      `json:"minutes_taken"`
	NeededHelp         bool     `json:"needed_help"`
	Observation        string   `json:"obs"`
	CrackedExercise    bool     `json:"cracked_exercise"`
}
