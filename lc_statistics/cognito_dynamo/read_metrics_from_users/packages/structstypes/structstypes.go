package structstypes

type UserMetrics struct {
	UserID                 string             `json:"user_id" dynamodbav:"user_id"`
	Date                   string             `json:"date" dynamodbav:"date"`
	ShortWindowDays        int                `json:"short_window_days" dynamodbav:"short_window_days"`
	LongWindowDays         int                `json:"long_window_days" dynamodbav:"long_window_days"`
	AvgMinutesPerTag       map[string]float64 `json:"avg_minutes_per_tag" dynamodbav:"avg_minutes_per_tag"`
	HelpRatePerTag         map[string]float64 `json:"help_rate_per_tag" dynamodbav:"help_rate_per_tag"`
	SolvedPerTag           map[string]int     `json:"solved_per_tag" dynamodbav:"solved_per_tag"`
	FailedPerTag           map[string]int     `json:"failed_per_tag" dynamodbav:"failed_per_tag"`
	AvgSolvedLastShortDays float64            `json:"avg_solved_last_short_window" dynamodbav:"avg_solved_last_short_window"`
	AvgSolvedLastLongDays  float64            `json:"avg_solved_last_long_window" dynamodbav:"avg_solved_last_long_window"`
	AvgFailedLastShortDays float64            `json:"avg_failed_last_short_window" dynamodbav:"avg_failed_last_short_window"`
	AvgFailedLastLongDays  float64            `json:"avg_failed_last_long_window" dynamodbav:"avg_failed_last_long_window"`
	ConsistencyRate        float64            `json:"consistency_rate" dynamodbav:"consistency_rate"`
	LastActivityDaysAgo    int                `json:"last_activity_days_ago" dynamodbav:"last_activity_days_ago"`
	TotalQuestionsAnalyzed int                `json:"total_questions_analyzed" dynamodbav:"total_questions_analyzed"`
	CalculatedAtUTC        string             `json:"calculated_at_utc" dynamodbav:"calculated_at_utc"`
}
