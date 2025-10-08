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
