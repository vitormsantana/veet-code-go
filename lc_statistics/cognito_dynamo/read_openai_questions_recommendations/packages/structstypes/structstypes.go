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

type UserProfile struct {
	UserID                 string  `json:"user_id" dynamodbav:"user_id"`
	TargetCompany          string  `json:"target_company" dynamodbav:"target_company"`
	DesiredRole            string  `json:"desired_role" dynamodbav:"desired_role"`
	DesiredLevel           string  `json:"desired_level" dynamodbav:"desired_level"`
	YearsOfExperience      int     `json:"years_of_experience" dynamodbav:"years_of_experience"`
	MainStack              string  `json:"main_stack" dynamodbav:"main_stack"`
	LeetCodeExperience     string  `json:"leetcode_experience" dynamodbav:"leetcode_experience"`
	InterviewExperience    string  `json:"interview_experience" dynamodbav:"interview_experience"`
	CountryTarget          string  `json:"country_target" dynamodbav:"country_target"`
	ScheduledInterview     string  `json:"scheduled_interview" dynamodbav:"scheduled_interview"`
	TopicsFamiliarity      string  `json:"topics_familiarity" dynamodbav:"topics_familiarity"`
	ProfileLastUpdated     string  `json:"profile_last_updated" dynamodbav:"profile_last_updated"`
	AIPersonalizationScore float64 `json:"ai_personalization_score" dynamodbav:"ai_personalization_score"`
}

type UserMetrics struct {
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
