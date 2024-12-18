package visualization

import (
	"time"
)

// EventType represents different types of visualization events
type EventType string

const (
	EventAgentStarted    EventType = "agent_started"
	EventAgentCompleted  EventType = "agent_completed"
	EventMessageSent     EventType = "message_sent"
	EventCycleDetected   EventType = "cycle_detected"
	EventWorkflowStarted EventType = "workflow_started"
	EventWorkflowEnded   EventType = "workflow_ended"
)

// Event represents a visualization event
type Event struct {
	Type      EventType   `json:"type"`
	Data      interface{} `json:"data"`
	Timestamp time.Time   `json:"timestamp"`
}

// AgentStartedData represents data for agent started event
type AgentStartedData struct {
	AgentName string `json:"agent_name"`
	Step      int    `json:"step"`
}

// AgentCompletedData represents data for agent completed event
type AgentCompletedData struct {
	AgentName string        `json:"agent_name"`
	Step      int           `json:"step"`
	Duration  time.Duration `json:"duration"`
}

// MessageSentData represents data for message sent event
type MessageSentData struct {
	FromAgent string `json:"from_agent"`
	ToAgent   string `json:"to_agent"`
	Content   string `json:"content"`
}

// CycleDetectedData represents data for cycle detection event
type CycleDetectedData struct {
	FromAgent string `json:"from_agent"`
	ToAgent   string `json:"to_agent"`
	Count     int    `json:"count"`
}

// WorkflowData represents data for workflow events
type WorkflowData struct {
	Agents      []string            `json:"agents"`
	Connections map[string][]string `json:"connections"`
	Teams       map[string][]string `json:"teams"`
	TeamLeaders map[string]string   `json:"team_leaders"`
}
