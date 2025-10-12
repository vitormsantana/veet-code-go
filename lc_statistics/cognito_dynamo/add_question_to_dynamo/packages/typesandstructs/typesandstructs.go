package typesandstructs

import (
	"encoding/json"
	"strings"
)

type Request struct {
	QuestionName       string   `json:"name"`
	QuestionDate       string   `json:"date"`
	QuestionDifficulty string   `json:"difficulty"`
	QuestionTags       []string `json:"tags"`
	MinutesTaken       int      `json:"minutes_taken"`
	NeededHelp         bool     `json:"needed_help"`
	Observation        string   `json:"obs"`
	CrackedExercise    bool     `json:"cracked_exercise"`
}

func (r *Request) UnmarshalJSON(data []byte) error {
	type Alias Request
	aux := &struct {
		Obs         *string `json:"obs"`
		Observation *string `json:"observation"`
		*Alias
	}{
		Alias: (*Alias)(r),
	}

	if err := json.Unmarshal(data, aux); err != nil {
		return err
	}

	switch {
	case aux.Obs != nil:
		r.Observation = strings.TrimSpace(*aux.Obs)
	case aux.Observation != nil:
		r.Observation = strings.TrimSpace(*aux.Observation)
	default:
		r.Observation = ""
	}

	return nil
}
