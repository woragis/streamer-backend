package platform

import "encoding/json"

type User struct {
	ID          string `json:"id"`
	RoomID      string `json:"roomId"`
	Platform    string `json:"platform"`
	Username    string `json:"username"`
	DisplayName string `json:"displayName"`
	FirstSeenAt string `json:"firstSeenAt"`
	LastSeenAt  string `json:"lastSeenAt"`
}

type Message struct {
	ID            string  `json:"id"`
	RoomID        string  `json:"roomId"`
	UserID        string  `json:"userId"`
	Platform      string  `json:"platform"`
	Username      string  `json:"username"`
	DisplayName   string  `json:"displayName"`
	LiveSessionID *string `json:"liveSessionId,omitempty"`
	Content       string  `json:"content"`
	CreatedAt     string  `json:"createdAt"`
	Deleted       bool    `json:"deleted"`
}

type StreamEvent struct {
	ID            string          `json:"id"`
	RoomID        string          `json:"roomId"`
	LiveSessionID *string         `json:"liveSessionId,omitempty"`
	Type          string          `json:"type"`
	Platform      string          `json:"platform"`
	Username      string          `json:"username"`
	Payload       json.RawMessage `json:"payload"`
	CreatedAt     string          `json:"createdAt"`
}

type BotRule struct {
	ID            string          `json:"id"`
	RoomID        string          `json:"roomId"`
	Name          string          `json:"name"`
	Enabled       bool            `json:"enabled"`
	TriggerType   string          `json:"triggerType"`
	TriggerValue  string          `json:"triggerValue"`
	ActionType    string          `json:"actionType"`
	ActionPayload json.RawMessage `json:"actionPayload"`
	CreatedAt     string          `json:"createdAt"`
}

type IngestMessageInput struct {
	Platform      string  `json:"platform"`
	Username      string  `json:"username"`
	DisplayName   string  `json:"displayName,omitempty"`
	Content       string  `json:"content"`
	LiveSessionID *string `json:"liveSessionId,omitempty"`
}

type IngestEventInput struct {
	Type          string          `json:"type"`
	Platform      string          `json:"platform"`
	Username      string          `json:"username"`
	LiveSessionID *string         `json:"liveSessionId,omitempty"`
	Payload       json.RawMessage `json:"payload,omitempty"`
}

type CreateRuleInput struct {
	Name          string          `json:"name"`
	TriggerType   string          `json:"triggerType,omitempty"`
	TriggerValue  string          `json:"triggerValue"`
	ActionType    string          `json:"actionType"`
	ActionPayload json.RawMessage `json:"actionPayload"`
	Enabled       *bool           `json:"enabled,omitempty"`
}

type UpdateRuleInput struct {
	Name          *string          `json:"name,omitempty"`
	TriggerValue  *string          `json:"triggerValue,omitempty"`
	ActionType    *string          `json:"actionType,omitempty"`
	ActionPayload *json.RawMessage `json:"actionPayload,omitempty"`
	Enabled       *bool            `json:"enabled,omitempty"`
}

type RuleResult struct {
	RuleID     string          `json:"ruleId"`
	RuleName   string          `json:"ruleName"`
	ActionType string          `json:"actionType"`
	Applied    bool            `json:"applied"`
	Detail     json.RawMessage `json:"detail,omitempty"`
}

type IngestResult struct {
	Message      Message      `json:"message"`
	TriggeredRules []RuleResult `json:"triggeredRules,omitempty"`
}

type Dashboard struct {
	Month              string              `json:"month"`
	LeetCode           DashboardLeetCode   `json:"leetcode"`
	Calisthenics       DashboardCal        `json:"calisthenics"`
	Chat               DashboardChat       `json:"chat"`
	ProficiencyTimeline []TimelinePoint    `json:"proficiencyTimeline"`
}

type DashboardLeetCode struct {
	SolvedCount   int `json:"solvedCount"`
	LiveSessions  int `json:"liveSessions"`
	Streak        int `json:"streak"`
}

type DashboardCal struct {
	AcquisitionCount int `json:"acquisitionCount"`
	Workouts         int `json:"workouts"`
}

type DashboardChat struct {
	MessageCount   int `json:"messageCount"`
	UniqueViewers  int `json:"uniqueViewers"`
}

type TimelinePoint struct {
	Date   string `json:"date"`
	Label  string `json:"label"`
	Count  int    `json:"count"`
	Domain string `json:"domain"`
}
