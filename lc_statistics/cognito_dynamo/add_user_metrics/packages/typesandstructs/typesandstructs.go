package typesandstructs

type Request struct {
	MetricsDate     string `json:"date"`
	ShortWindowDays int    `json:"short_window_days"`
	LongWindowDays  int    `json:"long_window_days"`
}

type Question struct {
	QuestionID      string   `json:"question_id"`
	UserID          string   `json:"user_id"`
	QuestionName    string   `json:"name"`
	QuestionDate    string   `json:"date"`
	QuestionTags    []string `json:"tags"`
	MinutesTaken    int      `json:"minutes_taken"`
	NeededHelp      bool     `json:"needed_help"`
	Observation     string   `json:"obs"`
	CrackedExercise bool     `json:"cracked_exercise"`
}

type UserMetrics struct {
	MetricID                      string             `json:"metric_id" dynamodbav:"metric_id"`
	UserID                        string             `json:"user_id" dynamodbav:"user_id"`
	Date                          string             `json:"date" dynamodbav:"date"`
	ShortWindowDays               int                `json:"short_window_days" dynamodbav:"short_window_days"`
	LongWindowDays                int                `json:"long_window_days" dynamodbav:"long_window_days"`
	AvgMinutesPerTag              map[string]float64 `json:"avg_minutes_per_tag" dynamodbav:"avg_minutes_per_tag"`
	HelpRatePerTag                map[string]float64 `json:"help_rate_per_tag" dynamodbav:"help_rate_per_tag"`
	SolvedPerTag                  map[string]int     `json:"solved_per_tag" dynamodbav:"solved_per_tag"`
	FailedPerTag                  map[string]int     `json:"failed_per_tag" dynamodbav:"failed_per_tag"`
	AvgSolvedLastShortDays        float64            `json:"avg_solved_last_short_window" dynamodbav:"avg_solved_last_short_window"`
	AvgSolvedLastLongDays         float64            `json:"avg_solved_last_long_window" dynamodbav:"avg_solved_last_long_window"`
	AvgFailedLastShortDays        float64            `json:"avg_failed_last_short_window" dynamodbav:"avg_failed_last_short_window"`
	AvgFailedLastLongDays         float64            `json:"avg_failed_last_long_window" dynamodbav:"avg_failed_last_long_window"`
	ExercisesTriedLastShortWindow int                `json:"exercises_tried_last_short_window" dynamodbav:"exercises_tried_last_short_window"`
	ExercisesTriedLastLongWindow  int                `json:"exercises_tried_last_long_window" dynamodbav:"exercises_tried_last_long_window"`
	ConsistencyRate               float64            `json:"consistency_rate" dynamodbav:"consistency_rate"`
	LastActivityDaysAgo           int                `json:"last_activity_days_ago" dynamodbav:"last_activity_days_ago"`
	TotalQuestionsAnalyzed        int                `json:"total_questions_analyzed" dynamodbav:"total_questions_analyzed"`
	CalculatedAtUTC               string             `json:"calculated_at_utc" dynamodbav:"calculated_at_utc"`
}

type WindowStats struct {
	Days                    int
	TotalConsidered         int
	SolvedCount             int
	FailedCount             int
	ConsideredIDs           []string
	ConsideredDetails       []string
	SolvedIDs               []string
	FailedIDs               []string
	OutsideWindowIDs        []string
	OutsideWindowDetails    []string
	ParseErrorQuestionIDs   []string
	ParseErrorQuestionNotes []string
}

func (ws WindowStats) SuccessRate() float64 {
	if ws.TotalConsidered == 0 {
		return 0
	}
	return float64(ws.SolvedCount) / float64(ws.TotalConsidered)
}

func (ws WindowStats) FailureRate() float64 {
	if ws.TotalConsidered == 0 {
		return 0
	}
	return float64(ws.FailedCount) / float64(ws.TotalConsidered)
}
