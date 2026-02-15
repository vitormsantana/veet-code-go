package typesandstructs

type APIInfo struct {
	Name       string `json:"name"`
	Endpoint   string `json:"endpoint"`
	Method     string `json:"method"`
	StatusCode *int   `json:"statusCode,omitempty"`
	Outcome    string `json:"outcome,omitempty"`
}

type UserInfo struct {
	Sub   string `json:"sub,omitempty"`
	Email string `json:"email,omitempty"`
}

type AnalyticsEvent struct {
	EventID   string                 `json:"eventId"`
	Timestamp string                 `json:"timestamp"`
	App       string                 `json:"app"`
	EventType string                 `json:"eventType"`
	Phase     string                 `json:"phase"`
	Source    string                 `json:"source"`
	Feature   string                 `json:"feature"`
	Page      string                 `json:"page"`
	Label     string                 `json:"label,omitempty"`
	API       *APIInfo               `json:"api,omitempty"`
	User      *UserInfo              `json:"user,omitempty"`
	Metadata  map[string]interface{} `json:"metadata,omitempty"`
}

type AnalyticsBatchRequest struct {
	BatchID string           `json:"batchId"`
	SentAt  string           `json:"sentAt,omitempty"`
	Reason  string           `json:"reason,omitempty"`
	App     string           `json:"app,omitempty"`
	Events  []AnalyticsEvent `json:"events"`
}

type PersistedAnalyticsEvent struct {
	EventID            string                 `json:"eventId"`
	EventTimestamp     string                 `json:"eventTimestamp"`
	IngestionTimestamp string                 `json:"ingestionTimestamp"`
	Year               string                 `json:"year"`
	Month              string                 `json:"month"`
	Day                string                 `json:"day"`
	App                string                 `json:"app"`
	EventType          string                 `json:"eventType"`
	Phase              string                 `json:"phase"`
	Source             string                 `json:"source"`
	Feature            string                 `json:"feature"`
	Page               string                 `json:"page"`
	Label              string                 `json:"label,omitempty"`
	APIName            string                 `json:"apiName,omitempty"`
	APIEndpoint        string                 `json:"apiEndpoint,omitempty"`
	APIMethod          string                 `json:"apiMethod,omitempty"`
	APIStatusCode      *int                   `json:"apiStatusCode,omitempty"`
	APIOutcome         string                 `json:"apiOutcome,omitempty"`
	UserSub            string                 `json:"userSub,omitempty"`
	UserEmail          string                 `json:"userEmail,omitempty"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}
