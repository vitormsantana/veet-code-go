package typesandstructs

type UserProfile struct {
	UserID                 string  `json:"user_id"`
	TargetCompany          string  `json:"target_company"`
	DesiredRole            string  `json:"desired_role"`
	DesiredLevel           string  `json:"desired_level"`
	YearsOfExperience      int     `json:"years_of_experience"`
	MainStack              string  `json:"main_stack"`
	LeetCodeExperience     string  `json:"leetcode_experience"`
	InterviewExperience    string  `json:"interview_experience"`
	CountryTarget          string  `json:"country_target"`
	ScheduledInterview     string  `json:"scheduled_interview"`
	TopicsFamiliarity      string  `json:"topics_familiarity"`
	ProfileLastUpdated     string  `json:"profile_last_updated"`
	AIPersonalizationScore float64 `json:"-"`
}
