package buildprompt

import (
	"encoding/json"
	"fmt"

	"github.com/vitormsantana/veet-code-go/cognito_dynamo/read_openai_questions_recommendations/packages/structstypes"
)

func BuildPrompt(goal string, questionNames []string, stats structstypes.Statistics, profile *structstypes.UserProfile) (string, error) {
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
	prompt := fmt.Sprintf(`User's goal: "%s".
User's profile (consider this context): %s
User's historical data: %s

You are an AI assistant that helps people prepare for coding interviews.
Write a short and friendly **personalized introduction paragraph (2–3 lines)** before listing the recommendations.
This paragraph should:
- Reference the user’s experience level and pace inferred from profile and stats.
- Encourage them by summarizing what they’re doing well and what they can improve.
- Use a conversational but professional tone (e.g. “You’ve been making steady progress…”).

Then, list **3 new recommended questions** (that do not appear in the previously solved list) using the format below:

### Personalized Recommendations

**Intro (2–3 lines)**: <your motivational summary here>

**Suggested Questions**
1. <Category Name>
**Question**: <Exact question title>
**Reason**: <Why this question fits the user’s goals and experience>
2. <Category Name>
**Question**: <Exact question title>
**Reason**: <Why this question fits the user’s goals and experience>
3. <Category Name>
**Question**: <Exact question title>
**Reason**: <Why this question fits the user’s goals and experience>

Focus on interview relevance and progression difficulty.
Avoid recommending questions already solved recently.
Previously solved list: %s

Main topics reference:
'Arrays', 'Backtracking', 'String', 'Binary Search', 'Hash Tables', 'Linked Lists',
'Two Pointers', 'Sliding Window', 'Stacks', 'Queues', 'Heaps', 'Recursion',
'Tree', 'BST', 'Binary Tree', 'BFS', 'DFS', 'Sets', 'Sort', 'Dynamic Programming',
'Memoization', 'Graph', 'Math', 'Greedy'.`,
		goal, string(profileJSON), string(statsJSON), string(namesJSON),
	)

	return prompt, nil
}
