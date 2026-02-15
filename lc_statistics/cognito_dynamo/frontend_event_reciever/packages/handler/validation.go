package handler

import (
	"fmt"
	"strings"
	"time"

	"github.com/vitormsantana/veet-code-go/lc_statistics/cognito_dynamo/frontend_event_reciever/packages/typesandstructs"
)

func validateEvent(ev typesandstructs.AnalyticsEvent) error {
	if strings.TrimSpace(ev.EventID) == "" {
		return fmt.Errorf("eventId is required")
	}
	if strings.TrimSpace(ev.Timestamp) == "" {
		return fmt.Errorf("timestamp is required")
	}
	if _, err := time.Parse(time.RFC3339, ev.Timestamp); err != nil {
		return fmt.Errorf("timestamp must be a valid RFC3339 datetime")
	}
	if strings.TrimSpace(ev.App) == "" {
		return fmt.Errorf("app is required")
	}
	if strings.TrimSpace(ev.EventType) == "" {
		return fmt.Errorf("eventType is required")
	}
	if !contains([]string{"page_access", "button_click", "api_call"}, ev.EventType) {
		return fmt.Errorf("eventType must be one of [page_access, button_click, api_call]")
	}
	if strings.TrimSpace(ev.Phase) == "" {
		return fmt.Errorf("phase is required")
	}
	if !contains([]string{"start", "response"}, ev.Phase) {
		return fmt.Errorf("phase must be one of [start, response]")
	}
	if strings.TrimSpace(ev.Source) == "" {
		return fmt.Errorf("source is required")
	}
	if !contains([]string{"page_load", "user_click"}, ev.Source) {
		return fmt.Errorf("source must be one of [page_load, user_click]")
	}
	if strings.TrimSpace(ev.Feature) == "" {
		return fmt.Errorf("feature is required")
	}
	if strings.TrimSpace(ev.Page) == "" {
		return fmt.Errorf("page is required")
	}

	if ev.EventType == "api_call" && ev.API == nil {
		return fmt.Errorf("api object is required when eventType is api_call")
	}

	// Transition rule:
	// - button_click must not carry api payload (send a separate api_call event)
	// - page_access may still carry api during frontend migration
	if ev.EventType == "button_click" && ev.API != nil {
		return fmt.Errorf("api object is allowed only when eventType is api_call")
	}

	if ev.API != nil {
		if strings.TrimSpace(ev.API.Name) == "" {
			return fmt.Errorf("api.name is required when api object is provided")
		}
		if strings.TrimSpace(ev.API.Endpoint) == "" {
			return fmt.Errorf("api.endpoint is required when api object is provided")
		}
		if strings.TrimSpace(ev.API.Method) == "" {
			return fmt.Errorf("api.method is required when api object is provided")
		}
		if !contains([]string{"GET", "POST", "PUT", "DELETE"}, strings.ToUpper(ev.API.Method)) {
			return fmt.Errorf("api.method must be one of [GET, POST, PUT, DELETE]")
		}
		if ev.API.Outcome != "" && !contains([]string{"success", "error"}, ev.API.Outcome) {
			return fmt.Errorf("api.outcome must be one of [success, error]")
		}
	}

	return nil
}

func contains(options []string, value string) bool {
	for _, option := range options {
		if option == value {
			return true
		}
	}
	return false
}
