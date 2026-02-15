package feedbacksummary

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	"github.com/vitormsantana/veet-code-go/cognito_dynamo/add_feedback_for_recomendation/packages/typesandstructs"
)

var feedbackTagAliases = map[string][]string{
	"Arrays":              {"arrays", "array"},
	"Backtracking":        {"backtracking"},
	"String":              {"string", "strings"},
	"Binary Search":       {"binary search"},
	"Hash Tables":         {"hash table", "hash tables", "hashmap", "hash map"},
	"Linked Lists":        {"linked list", "linked lists"},
	"Two Pointers":        {"two pointers", "two-pointer"},
	"Sliding Window":      {"sliding window"},
	"Stacks":              {"stack", "stacks"},
	"Queues":              {"queue", "queues"},
	"Heaps":               {"heap", "heaps", "priority queue"},
	"Recursion":           {"recursion", "recursive"},
	"Tree":                {"tree", "trees"},
	"BST":                 {"bst", "binary search tree"},
	"Binary Tree":         {"binary tree", "binary trees"},
	"BFS":                 {"bfs", "breadth-first"},
	"DFS":                 {"dfs", "depth-first"},
	"Sets":                {"set", "sets"},
	"Sort":                {"sort", "sorting"},
	"Dynamic Programming": {"dynamic programming", "dp"},
	"Memoization":         {"memoization", "memoize"},
	"Graph":               {"graph", "graphs"},
	"Math":                {"math", "mathematics"},
	"Greedy":              {"greedy"},
}

type PromptSnapshot struct {
	UserProfile    *typesandstructs.UserProfile `json:"user_profile"`
	FeedbackTotals FeedbackTotals               `json:"feedback_totals"`
	TagSignals     TagSignals                   `json:"tag_signals"`
	FeedbackItems  []PromptFeedbackItem         `json:"feedback_items"`
}

type FeedbackTotals struct {
	Total            int `json:"total"`
	Positive         int `json:"positive"`
	Negative         int `json:"negative"`
	Neutral          int `json:"neutral"`
	WithCommentCount int `json:"with_comment_count"`
}

type TagSignals struct {
	PriorityTopics []string `json:"priority_topics"`
	AvoidTopics    []string `json:"avoid_topics"`
}

type PromptFeedbackItem struct {
	FeedbackID      string                       `json:"feedback_id"`
	Timestamp       string                       `json:"timestamp"`
	FeedbackValue   int                          `json:"feedback_value"`
	Comment         string                       `json:"comment"`
	Recommendation  *PromptRecommendationSummary `json:"recommendation,omitempty"`
	PrioritySignals []string                     `json:"priority_signals,omitempty"`
	AvoidSignals    []string                     `json:"avoid_signals,omitempty"`
}

type PromptRecommendationSummary struct {
	RecommendationID string   `json:"recommendation_id"`
	FocusTags        []string `json:"focus_tags,omitempty"`
	DifficultyBand   []string `json:"difficulty_band,omitempty"`
	Tone             string   `json:"tone,omitempty"`
	UserStatus       string   `json:"user_status,omitempty"`
}

func BuildPrompt(profile *typesandstructs.UserProfile, pairs []feedbackRecommendationPair) (string, error) {
	if len(pairs) == 0 {
		return "", fmt.Errorf("no feedback records to summarize")
	}

	var totals FeedbackTotals
	priorityCounts := make(map[string]int)
	avoidCounts := make(map[string]int)
	items := make([]PromptFeedbackItem, 0, len(pairs))

	var sampleRecommendation *typesandstructs.RecommendationRecord

	for _, pair := range pairs {
		item := PromptFeedbackItem{
			FeedbackID:    pair.Feedback.FeedbackID,
			Timestamp:     pair.Feedback.FeedbackTimestamp,
			FeedbackValue: pair.Feedback.FeedbackValue,
			Comment:       strings.TrimSpace(pair.Feedback.FeedbackComment),
		}

		switch {
		case pair.Feedback.FeedbackValue > 0:
			totals.Positive++
		case pair.Feedback.FeedbackValue < 0:
			totals.Negative++
		default:
			totals.Neutral++
		}

		if item.Comment != "" {
			totals.WithCommentCount++
			preferred, avoid := classifyTagsFromComment(item.Comment)
			if len(preferred) > 0 {
				item.PrioritySignals = append(item.PrioritySignals, preferred...)
				for _, tag := range preferred {
					priorityCounts[tag]++
				}
			}
			if len(avoid) > 0 {
				item.AvoidSignals = append(item.AvoidSignals, avoid...)
				for _, tag := range avoid {
					avoidCounts[tag]++
				}
			}
		}

		if pair.Recommendation != nil {
			if sampleRecommendation == nil {
				copy := *pair.Recommendation
				sampleRecommendation = &copy
			}

			meta := pair.Recommendation.Metadata
			item.Recommendation = &PromptRecommendationSummary{
				RecommendationID: pair.Recommendation.RecommendationID,
				FocusTags:        append([]string(nil), meta.FocusTags...),
				DifficultyBand:   append([]string(nil), meta.DifficultyBand...),
				Tone:             meta.Tone,
				UserStatus:       meta.UserStatus,
			}

			for _, tag := range meta.FocusTags {
				if pair.Feedback.FeedbackValue > 0 {
					priorityCounts[tag]++
				} else if pair.Feedback.FeedbackValue < 0 {
					avoidCounts[tag]++
				}
			}
		}

		items = append(items, item)
	}

	totals.Total = len(pairs)

	snapshot := PromptSnapshot{
		UserProfile:    profile,
		FeedbackTotals: totals,
		TagSignals: TagSignals{
			PriorityTopics: sortedKeysByCount(priorityCounts),
			AvoidTopics:    sortedKeysByCount(avoidCounts),
		},
		FeedbackItems: items,
	}

	payload, err := json.MarshalIndent(snapshot, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal prompt payload: %w", err)
	}

	var exampleBlock string
	if sampleRecommendation != nil {
		if sampleBytes, err := json.MarshalIndent(sampleRecommendation, "", "  "); err == nil {
			exampleBlock = fmt.Sprintf("\nExample recommendation record (for reference):\n%s\n", string(sampleBytes))
		}
	}

	formatSpec := `{
  "narrative": string,                        // concise paragraph summarizing themes and progress.
  "priority_topics": [string],                // 1-4 topics to emphasize next. Use canonical topic names.
  "avoid_topics": [string],                   // 0-4 topics to pause/avoid.
  "tone_guidance": [string],                  // words describing tone adjustments (e.g., "supportive", "direct").
  "coaching_actions": [string],               // actionable coaching reminders for recommendation generation.
  "confidence": "low" | "medium" | "high"     // confidence in these conclusions.
}`

	prompt := fmt.Sprintf(
		`You are processing user feedback about coding interview study plans.
Input payload (JSON):
%s
%s

Summarize the insights for downstream recommendation generation.

Instructions:
- Carefully read comments and tag signals to understand what the user wants more of or less of.
- Be explicit when the user asks to avoid a topic (e.g., Dynamic Programming).
- Focus on clarity—make it easy for another system to consume.
- Reflect negative feedback as opportunities to adjust focus or tone.

Output requirements:
- Respond with a JSON object matching exactly the schema below.
- Use lowercase keys and double-quoted strings.
- Limit arrays to at most four unique items each.
- Do not include extra commentary or Markdown.

Schema:
%s
`,
		string(payload),
		exampleBlock,
		formatSpec,
	)

	return prompt, nil
}

func classifyTagsFromComment(comment string) (preferred []string, avoid []string) {
	if comment == "" {
		return nil, nil
	}

	lower := strings.ToLower(comment)
	for canonical, aliases := range feedbackTagAliases {
		for _, alias := range aliases {
			idx := strings.Index(lower, alias)
			if idx == -1 {
				continue
			}

			windowStart := idx - 20
			if windowStart < 0 {
				windowStart = 0
			}
			windowEnd := idx + len(alias) + 20
			if windowEnd > len(lower) {
				windowEnd = len(lower)
			}
			context := lower[windowStart:windowEnd]

			if containsAny(context, []string{"not ", "avoid", "less ", "skip", "without"}) {
				avoid = append(avoid, canonical)
				continue
			}
			if containsAny(context, []string{"focus", "more ", "prefer", "prioritize", "need", "practice"}) {
				preferred = append(preferred, canonical)
			}
		}
	}

	return uniqueStrings(preferred), uniqueStrings(avoid)
}

func containsAny(s string, needles []string) bool {
	for _, n := range needles {
		if strings.Contains(s, n) {
			return true
		}
	}
	return false
}

func uniqueStrings(in []string) []string {
	m := make(map[string]struct{})
	var out []string
	for _, s := range in {
		if _, exists := m[s]; exists {
			continue
		}
		m[s] = struct{}{}
		out = append(out, s)
	}
	return out
}

func sortedKeysByCount(counts map[string]int) []string {
	if len(counts) == 0 {
		return nil
	}
	type kv struct {
		Key   string
		Count int
	}
	rows := make([]kv, 0, len(counts))
	for k, v := range counts {
		if v <= 0 {
			continue
		}
		rows = append(rows, kv{Key: k, Count: v})
	}
	if len(rows) == 0 {
		return nil
	}
	sort.Slice(rows, func(i, j int) bool {
		if rows[i].Count == rows[j].Count {
			return rows[i].Key < rows[j].Key
		}
		return rows[i].Count > rows[j].Count
	})
	out := make([]string, 0, len(rows))
	for _, row := range rows {
		out = append(out, row.Key)
	}
	return out
}
