package timers

import (
	"encoding/json"
	"errors"
)

func ApplyAction(timer map[string]any, action string, nowMs int64) error {
	switch action {
	case "start":
		if running, _ := timer["running"].(bool); running {
			return nil
		}
		timer["running"] = true
		timer["startedAt"] = nowMs
		if mode, _ := timer["mode"].(string); mode == "countdown" {
			dur := ToInt(timer["durationSeconds"])
			acc := ToInt(timer["accumulatedSeconds"])
			remaining := dur - acc
			if remaining < 0 {
				remaining = 0
			}
			timer["endsAt"] = nowMs + int64(remaining)*1000
		}
	case "pause":
		if running, _ := timer["running"].(bool); !running {
			return nil
		}
		startedAt := ToInt64(timer["startedAt"])
		if startedAt > 0 {
			elapsed := (nowMs - startedAt) / 1000
			timer["accumulatedSeconds"] = ToInt(timer["accumulatedSeconds"]) + int(elapsed)
		}
		timer["running"] = false
		timer["startedAt"] = nil
		timer["endsAt"] = nil
	case "reset":
		timer["running"] = false
		timer["startedAt"] = nil
		timer["endsAt"] = nil
		timer["accumulatedSeconds"] = 0
	default:
		return errors.New("unknown action; use start, pause, or reset")
	}
	return nil
}

func ToInt(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	case int64:
		return int(n)
	case json.Number:
		i, _ := n.Int64()
		return int(i)
	default:
		return 0
	}
}

func ToInt64(v any) int64 {
	switch n := v.(type) {
	case float64:
		return int64(n)
	case int:
		return int64(n)
	case int64:
		return n
	case json.Number:
		i, _ := n.Int64()
		return i
	default:
		return 0
	}
}

func DefaultCalisthenicsTimers() map[string]any {
	return map[string]any{
		"rest": map[string]any{
			"id": "rest", "mode": "countdown", "label": "Rest",
			"durationSeconds": 90, "accumulatedSeconds": 0,
			"running": false, "startedAt": nil, "endsAt": nil,
		},
		"hold": map[string]any{
			"id": "hold", "mode": "stopwatch", "label": "Hold",
			"durationSeconds": 0, "accumulatedSeconds": 0,
			"running": false, "startedAt": nil, "endsAt": nil,
		},
	}
}

func GetTimer(timers map[string]any, id string) (map[string]any, bool) {
	raw, ok := timers[id]
	if !ok {
		return nil, false
	}
	m, ok := raw.(map[string]any)
	return m, ok
}

func StartTimerInMap(timers map[string]any, id string, nowMs int64) error {
	t, ok := GetTimer(timers, id)
	if !ok {
		return errors.New("timer not found")
	}
	return ApplyAction(t, "start", nowMs)
}
