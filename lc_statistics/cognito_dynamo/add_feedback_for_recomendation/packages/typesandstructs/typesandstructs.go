package typesandstructs

type Feedback struct {
	RecomendationID string `json:"recomendation_id"`
	FeedbackValue   int    `json:"feedback_value"`
	FeedbackComment string `json:"feedback_comment,omitempty"`
}
