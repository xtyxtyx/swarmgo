package swarmgo

type Agent struct {
	Name              string
	Model             string
	Instructions      string
	InstructionsFunc  func(contextVariables map[string]interface{}) string
	Functions         []AgentFunction
	ToolChoice        string
	ParallelToolCalls bool
}

type AgentFunction struct {
	Name        string
	Description string
	Parameters  map[string]interface{}
	Function    func(args map[string]interface{}, contextVariables map[string]interface{}) Result
}
