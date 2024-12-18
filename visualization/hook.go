package visualization

import (
	"time"

	"github.com/prathyushnallamothu/swarmgo"
)

// Hook implements visualization hooks for the workflow
type Hook struct {
	server *Server
}

// NewHook creates a new visualization hook
func NewHook(port int) *Hook {
	server := NewServer()
	go server.Start(port)
	return &Hook{server: server}
}

// OnWorkflowStart is called when the workflow starts
func (h *Hook) OnWorkflowStart(workflow *swarmgo.Workflow) {
	// Convert workflow data to visualization format
	data := WorkflowData{
		Agents:      make([]string, 0),
		Connections: make(map[string][]string),
		Teams:       make(map[string][]string),
		TeamLeaders: make(map[string]string),
	}

	// Add agents
	for name := range workflow.GetAgents() {
		data.Agents = append(data.Agents, name)
	}

	// Add connections
	for from, toList := range workflow.GetConnections() {
		data.Connections[from] = toList
	}

	// Add teams and leaders
	for team, agents := range workflow.GetTeams() {
		teamAgents := make([]string, len(agents))
		for i, agent := range agents {
			teamAgents[i] = agent.Name
		}
		data.Teams[string(team)] = teamAgents
	}
	for team, leader := range workflow.GetTeamLeaders() {
		data.TeamLeaders[string(team)] = leader
	}

	h.server.BroadcastEvent(Event{
		Type:      EventWorkflowStarted,
		Data:      data,
		Timestamp: time.Now(),
	})
}

// OnAgentStart is called when an agent starts processing
func (h *Hook) OnAgentStart(agentName string, step int) {
	h.server.BroadcastEvent(Event{
		Type: EventAgentStarted,
		Data: AgentStartedData{
			AgentName: agentName,
			Step:      step,
		},
		Timestamp: time.Now(),
	})
}

// OnAgentComplete is called when an agent completes processing
func (h *Hook) OnAgentComplete(agentName string, step int, duration time.Duration) {
	h.server.BroadcastEvent(Event{
		Type: EventAgentCompleted,
		Data: AgentCompletedData{
			AgentName: agentName,
			Step:      step,
			Duration:  duration,
		},
		Timestamp: time.Now(),
	})
}

// OnMessageSent is called when a message is sent between agents
func (h *Hook) OnMessageSent(fromAgent, toAgent, content string) {
	h.server.BroadcastEvent(Event{
		Type: EventMessageSent,
		Data: MessageSentData{
			FromAgent: fromAgent,
			ToAgent:   toAgent,
			Content:   content,
		},
		Timestamp: time.Now(),
	})
}

// OnCycleDetected is called when a cycle is detected in the workflow
func (h *Hook) OnCycleDetected(fromAgent, toAgent string, count int) {
	h.server.BroadcastEvent(Event{
		Type: EventCycleDetected,
		Data: CycleDetectedData{
			FromAgent: fromAgent,
			ToAgent:   toAgent,
			Count:     count,
		},
		Timestamp: time.Now(),
	})
}

// OnWorkflowEnd is called when the workflow completes
func (h *Hook) OnWorkflowEnd(workflow *swarmgo.Workflow) {
	h.server.BroadcastEvent(Event{
		Type:      EventWorkflowEnded,
		Timestamp: time.Now(),
	})
}
