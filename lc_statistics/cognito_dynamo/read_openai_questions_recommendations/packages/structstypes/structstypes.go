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

type RecommendationItem struct {
	Category string `json:"category" dynamodbav:"category"`
	Question string `json:"question" dynamodbav:"question"`
	Reason   string `json:"reason" dynamodbav:"reason"`
}

type StructuredRecommendation struct {
	Intro           string               `json:"intro" dynamodbav:"intro"`
	Recommendations []RecommendationItem `json:"recommendations" dynamodbav:"recommendations"`
}

type RecommendationRecord struct {
	RecommendationID         string                   `json:"recommendation_id" dynamodbav:"recommendation_id"`
	UserID                   string                   `json:"user_id" dynamodbav:"user_id"`
	GeneratedAtUTC           string                   `json:"generated_at_utc" dynamodbav:"generated_at_utc"`
	CreatedAtUTC             string                   `json:"created_at_utc" dynamodbav:"created_at_utc"`
	Goal                     string                   `json:"goal" dynamodbav:"goal"`
	PromptUsed               string                   `json:"prompt_used" dynamodbav:"prompt_used"`
	ModelUsed                string                   `json:"model_used" dynamodbav:"model_used"`
	TextRecommendation       string                   `json:"text_recommendation" dynamodbav:"text_recommendation"`
	StructuredRecommendation StructuredRecommendation `json:"structured_recommendation" dynamodbav:"structured_recommendation"`
	Metadata                 RecommendationMetadata   `json:"metadata" dynamodbav:"metadata"`
	MetricsID                string                   `json:"metric_id,omitempty" dynamodbav:"metric_id,omitempty"`
	PreviouslySolved         []string                 `json:"previously_solved" dynamodbav:"previously_solved"`
	ProfileSnapshot          *UserProfile             `json:"user_profile_snapshot,omitempty" dynamodbav:"user_profile_snapshot,omitempty"`
	FeedbackStatus           string                   `json:"feedback_status" dynamodbav:"feedback_status"`
}

type RecommendationMetadata struct {
	FocusTags       []string `json:"focus_tags" dynamodbav:"focus_tags"`
	ConfidenceLevel string   `json:"confidence_level" dynamodbav:"confidence_level"`
	Tone            string   `json:"tone" dynamodbav:"tone"`
	DifficultyBand  []string `json:"difficulty_band" dynamodbav:"difficulty_band"`
	UserStatus      string   `json:"user_status" dynamodbav:"user_status"`
}

type Feedback struct {
	FeedbackID        string `json:"feedback_id" dynamodbav:"feedback_id"`
	FeedbackTimestamp string `json:"feedback_timestamp" dynamodbav:"feedback_timestamp"`
	FeedbackValue     int    `json:"feedback_value" dynamodbav:"feedback_value"`
	FeedbackComment   string `json:"feedback_comment,omitempty" dynamodbav:"feedback_comment,omitempty"`
	RecomendationID   string `json:"recomendation_id" dynamodbav:"recomendation_id"`
	UserID            string `json:"user_id" dynamodbav:"user_id"`
}

type ProcessedFeedbackSummary struct {
	SummaryID          string                   `json:"summary_id" dynamodbav:"summary_id"`
	UserID             string                   `json:"user_id" dynamodbav:"user_id"`
	GeneratedAtUTC     string                   `json:"generated_at_utc" dynamodbav:"generated_at_utc"`
	AnalyzedAtUTC      string                   `json:"analyzed_at_utc" dynamodbav:"analyzed_at_utc"`
	FeedbackCount      int                      `json:"feedback_count" dynamodbav:"feedback_count"`
	FeedbackIDs        []string                 `json:"feedback_ids,omitempty" dynamodbav:"feedback_ids,omitempty"`
	RecommendationIDs  []string                 `json:"recommendation_ids,omitempty" dynamodbav:"recommendation_ids,omitempty"`
	PromptUsed         string                   `json:"prompt_used" dynamodbav:"prompt_used"`
	ModelUsed          string                   `json:"model_used" dynamodbav:"model_used"`
	SummaryText        string                   `json:"summary_text" dynamodbav:"summary_text"`
	StructuredSummary  ProcessedFeedbackDetails `json:"structured_summary" dynamodbav:"structured_summary"`
	RawModelResponse   string                   `json:"raw_model_response,omitempty" dynamodbav:"raw_model_response,omitempty"`
	AdditionalMetadata map[string]string        `json:"additional_metadata,omitempty" dynamodbav:"additional_metadata,omitempty"`
}

type ProcessedFeedbackDetails struct {
	Narrative       string   `json:"narrative" dynamodbav:"narrative"`
	PriorityTopics  []string `json:"priority_topics,omitempty" dynamodbav:"priority_topics,omitempty"`
	AvoidTopics     []string `json:"avoid_topics,omitempty" dynamodbav:"avoid_topics,omitempty"`
	ToneGuidance    []string `json:"tone_guidance,omitempty" dynamodbav:"tone_guidance,omitempty"`
	CoachingActions []string `json:"coaching_actions,omitempty" dynamodbav:"coaching_actions,omitempty"`
	Confidence      string   `json:"confidence,omitempty" dynamodbav:"confidence,omitempty"`
}
