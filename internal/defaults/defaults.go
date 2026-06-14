package defaults

import (
	"encoding/json"
	"time"
)

const DefaultRoomID = "default"

const defaultCode = `class TreeNode:
    def __init__(self, val=0, left=None, right=None):
        self.val = val
        self.left = left
        self.right = right

class Solution:
    def maxPathSum(self, root):
        self.max_sum = float('-inf')

        def dfs(node):
            if not node:
                return 0
            left = max(dfs(node.left), 0)
            right = max(dfs(node.right), 0)
            self.max_sum = max(
                self.max_sum,
                node.val + left + right
            )
            return node.val + max(left, right)

        dfs(root)
        return self.max_sum`

func timer(id, mode, label string, durationSeconds, accumulatedSeconds int) map[string]any {
	return map[string]any{
		"id":                 id,
		"mode":               mode,
		"label":              label,
		"durationSeconds":    durationSeconds,
		"accumulatedSeconds": accumulatedSeconds,
		"running":            false,
		"startedAt":          nil,
		"endsAt":             nil,
	}
}

func Branding() json.RawMessage {
	return mustJSON(map[string]any{
		"handle":            "@yourhandle",
		"brandTitle":        "LEETCODE LIVE",
		"motto":             "FOCUS • DISCIPLINE • CONSISTENCY",
		"calisthenicsMotto": "DISCIPLINE TODAY FREEDOM TOMORROW",
		"schedule":          "Mon – Fri · 10:00 AM EST",
		"social": map[string]string{
			"discord": "discord.gg/yourserver",
			"twitter": "@yourtwitter",
			"youtube": "youtube.com/@yourchannel",
			"kick":    "kick.com/yourchannel",
		},
	})
}

func Session() json.RawMessage {
	return mustJSON(map[string]any{
		"scene":     "live",
		"startedAt": nil,
		"streamEvents": map[string]string{
			"latestSubscriber": "code_with_me",
			"latestFollower":   "algorithm_lover",
			"latestDonation":   "Anonymous - $10",
		},
	})
}

func StreamTimer() json.RawMessage {
	return mustJSON(timer("stream", "stopwatch", "Stream Time", 0, 5077))
}

func LeetCodeState() json.RawMessage {
	now := time.Now().UTC().Format(time.RFC3339)
	return mustJSON(map[string]any{
		"revision": 1,
		"plan": []map[string]any{
			{"id": "plan-1", "label": "2 Hard Problems", "done": true, "order": 0},
			{"id": "plan-2", "label": "Dynamic Programming", "done": true, "order": 1},
			{"id": "plan-3", "label": "Graphs", "done": false, "order": 2},
		},
		"problems": []map[string]any{
			{
				"id": 124, "title": "Binary Tree Maximum Path Sum", "difficulty": "hard",
				"description": "A path in the binary tree is a sequence of nodes where each pair of adjacent nodes in the sequence has an edge connecting them.",
				"status": "active", "solvedAt": nil, "order": 0,
			},
			{
				"id": 146, "title": "LRU Cache", "difficulty": "medium",
				"description": "", "status": "solved", "solvedAt": now, "order": 1,
			},
			{
				"id": 199, "title": "Binary Tree Right Side View", "difficulty": "medium",
				"description": "", "status": "solved", "solvedAt": now, "order": 2,
			},
			{
				"id": 239, "title": "Sliding Window Maximum", "difficulty": "hard",
				"description": "", "status": "queued", "solvedAt": nil, "order": 3,
			},
		},
		"code": map[string]string{
			"fileName": "solution.py",
			"content":  defaultCode,
		},
		"whiteboard": map[string]any{
			"title": "Binary Tree Maximum Path Sum",
			"bullets": []string{
				"For each node, compute max gain from left & right subtrees",
				"Path through node = left_gain + right_gain + node.val",
				"Update global max at each node (post-order DFS)",
				"Ignore negative gains (use max(0, gain))",
			},
			"notes":    []string{"Post-order DFS", "Track global max", "Skip negative paths"},
			"approach": "Using DFS (Post-order)",
		},
		"goals": map[string]int{
			"dailyTarget": 5, "weeklyTarget": 30, "streak": 12,
		},
		"copy": map[string]string{
			"startingSoonSubtext": "SOLVING PROBLEMS • IMPROVING EVERY DAY • BECOMING 1% BETTER",
			"brbSubtext":          "GRABBING WATER • STRETCHING • PREPARING THE NEXT PROBLEM",
			"brbMessage":          "I'LL BE BACK IN",
			"upNextLabel":         "Dynamic Programming Problem",
		},
		"loadingProgress": 87,
		"timers": map[string]any{
			"startingSoon": timer("startingSoon", "countdown", "Starting Soon", 300, 0),
			"brb":          timer("brb", "countdown", "BRB", 300, 0),
			"focus":        timer("focus", "countdown", "Focus", 1500, 0),
		},
	})
}

func CalisthenicsState() json.RawMessage {
	return mustJSON(map[string]any{
		"revision":    1,
		"workoutType": "PULL DAY",
		"exercises": []map[string]any{
			{
				"id": "ex-1", "name": "PULL-UPS", "sets": 5, "repTarget": 10,
				"completedSets": 2, "repsInCurrentSet": 8, "totalReps": 28,
				"status": "active", "order": 0,
			},
			{
				"id": "ex-2", "name": "CHIN-UPS", "sets": 4, "repTarget": 10,
				"completedSets": 0, "repsInCurrentSet": 0, "totalReps": 0,
				"status": "pending", "order": 1,
			},
		},
		"todayGoal": map[string]any{
			"label":    "COMPLETE THE WORKOUT",
			"progress": 75,
		},
		"timers": map[string]any{
			"rest": timer("rest", "countdown", "Rest", 90, 0),
		},
	})
}

func mustJSON(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}
