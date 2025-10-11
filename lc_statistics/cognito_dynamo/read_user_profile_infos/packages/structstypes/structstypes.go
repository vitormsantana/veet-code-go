package structstypes

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
