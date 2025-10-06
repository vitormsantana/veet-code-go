package typesandstructs

type Request struct {
	QuestionName       string   `json:"name"`
	QuestionDate       string   `json:"date"`
	QuestionDifficulty string   `json:"difficulty"`
	QuestionTags       []string `json:"tags"`
	MinutesTaken       int      `json:"minutes_taken"`
	NeededHelp         bool     `json:"needed_help"`
}
