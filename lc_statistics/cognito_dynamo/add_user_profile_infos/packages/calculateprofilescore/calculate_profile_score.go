package calculateprofilescore

import (
	"math"
	"strings"

	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_user_profile_infos/packages/typesandstructs"
)

func CalculateProfileScore(profile *typesandstructs.UserProfile) float64 {
	fields := []string{
		profile.TargetCompany,
		profile.DesiredRole,
		profile.DesiredLevel,
		profile.MainStack,
		profile.LeetCodeExperience,
		profile.InterviewExperience,
		profile.CountryTarget,
		profile.ScheduledInterview,
		profile.TopicsFamiliarity,
	}

	filled := 0
	for _, f := range fields {
		if strings.TrimSpace(f) != "" {
			filled++
		}
	}

	completeness := (float64(filled) / float64(len(fields))) * 100
	rawScore := math.Min(100, completeness)
	weighted := rawScore * 0.45
	return math.Round(weighted*100) / 100
}
