package swarmgo

// Agent represents an entity with specific attributes and behaviors.
type Agent struct {
	Name              string                                               // The name of the agent.
	Model             string                                               // The model.
	BaseURL           string                                               // API URLv1
	AuthToken         string                                               // API Key
	Instructions      string                                               // Static instructions for the agent.
	InstructionsFunc  func(contextVariables map[string]interface{}) string // Function to generate dynamic instructions based on context.
	Functions         []AgentFunction                                      // A list of functions the agent can perform.
	ToolChoice        string                                               // The tool or method chosen by the agent.
	ParallelToolCalls bool                                                 // Indicates if the agent can call tools in parallel.
	Memory            *MemoryStore                                         // Memory store for the agent
}

// AgentFunction represents a function that an agent can perform.
type AgentFunction struct {
	Name        string                                                                            // The name of the function.
	Description string                                                                            // A brief description of what the function does.
	Parameters  map[string]interface{}                                                            // Parameters required by the function.
	Function    func(args map[string]interface{}, contextVariables map[string]interface{}) Result // The actual function implementation.
}

// NewAgent creates a new agent with initialized memory store
func NewAgent(name, model string) *Agent {
	return &Agent{
		Name:     name,
		Model:    model,
		Memory:   NewMemoryStore(100), // Default to 100 short-term memories
	}
}
