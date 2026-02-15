package buildprompt

import (
	"encoding/json"
	"fmt"

	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_openai_questions_recommendations/packages/structstypes"
)

func BuildPrompt(
	goal string,
	questionNames []string,
	stats structstypes.Statistics,
	profile *structstypes.UserProfile,
	metrics *structstypes.UserMetrics,
	feedbackContext string,
) (string, error) {
	type promptStats struct {
		QuestionsCrackedPerDay            map[string]int              `json:"questionsCrackedPerDay"`
		QuestionsCrackedPerDifficulty     map[string]int              `json:"questionsCrackedPerDifficulty"`
		QuestionsCrackedPerTag            map[string]int              `json:"questionsCrackedPerTag"`
		IncrementalQuestionsCrackedPerDay []structstypes.DayStatistic `json:"incrementalQuestionsCrackedPerDay"`
	}

	statsPayload := promptStats{
		QuestionsCrackedPerDay:            stats.QuestionsCrackedPerDay,
		QuestionsCrackedPerDifficulty:     stats.QuestionsCrackedPerDifficulty,
		QuestionsCrackedPerTag:            stats.QuestionsCrackedPerTag,
		IncrementalQuestionsCrackedPerDay: stats.IncrementalQuestionsCrackedPerDay,
	}

	statsJSON, err := json.Marshal(statsPayload)
	if err != nil {
		return "", err
	}

	namesJSON, err := json.Marshal(questionNames)
	if err != nil {
		return "", err
	}

	profileJSON, err := json.Marshal(profile)
	if err != nil {
		return "", err
	}

	var metricsJSON []byte
	if metrics != nil {
		metricsJSON, err = json.Marshal(metrics)
		if err != nil {
			return "", err
		}
	} else {
		metricsJSON = []byte("null")
	}

	prompt := fmt.Sprintf(`
User's goal: "%s".
User's profile (consider this context): %s
User's exercise performance metrics: %s
User's historical stats (per day, difficulty, tag): %s
Recent feedback trends from previous recommendations: %s

You are an AI interview preparation coach.
Your job is to analyze the user's behavior, pace, and consistency to create an adaptive learning plan.

### Your Output
1. **Start with a short, friendly, 2–3 line paragraph ("Progress Summary")**
   - Reflect on the user's recent performance and pace using the metrics.
   - Mention improvement or slowdown trends (e.g., rising average times, higher success rates, lower consistency).
   - Encourage them in a natural, motivational tone (never robotic).

   **Use metrics intelligently:**
   - "avg_minutes_per_tag" → comment on their pacing speed ("you're solving faster each week", "try to slow down for accuracy").
   - "help_rate_per_tag" → mention independence or reliance on help.
   - "solved_per_tag" / "failed_per_tag" → identify strong vs weak topics.
   - "avg_solved_last_short_window" / "avg_failed_last_short_window" → show short-term trend.
   - "consistency_rate" → highlight engagement or inactivity.
   - "last_activity_days_ago" → welcome them back if they paused.
   - "total_questions_analyzed" → reference their journey so far.
   - Be empathetic and positive, like a mentor giving encouragement.

2. **Then list 3 personalized next questions**, in this structure:

### Personalized Recommendations

**Progress Summary (2–3 lines)**: <motivational paragraph>

**Suggested Questions**
1. <Category Name>
**Question**: <Exact question title>
**Reason**: <Why it fits their current goals and focus>

2. <Category Name>
**Question**: <Exact question title>
**Reason**: <Why it matches their weak areas or pacing>

3. <Category Name>
**Question**: <Exact question title>
**Reason**: <Why it helps them consolidate recent learning>

### Guidance for generation:
- Avoid repeating any question in the solved list.
- Prioritize topics with high failure or help rates.
- Reinforce tags where average minutes are improving (indicates growing mastery).
- Adapt difficulty: for beginners, focus on Arrays/Strings/Hash Tables;
  for intermediates, mix in recursion, DP, or trees.
- Use an inspiring, teacher-like tone.
- Always mention a reason why each question matters for interviews.
- Do not reference LeetCode or any specific external platform; keep the language platform-neutral.

Previously solved list: %s

Main topics reference:
'Arrays', 'Backtracking', 'String', 'Binary Search', 'Hash Tables', 'Linked Lists',
'Two Pointers', 'Sliding Window', 'Stacks', 'Queues', 'Heaps', 'Recursion',
'Tree', 'BST', 'Binary Tree', 'BFS', 'DFS', 'Sets', 'Sort',
'Dynamic Programming', 'Memoization', 'Graph', 'Math', 'Greedy'.`,
		goal,
		string(profileJSON),
		string(metricsJSON),
		string(statsJSON),
		feedbackContext,
		string(namesJSON),
	)

	return prompt, nil
}
