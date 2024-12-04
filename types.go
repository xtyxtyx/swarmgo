package swarmgo

import (
	"github.com/prathyushnallamothu/swarmgo/llm"
)

// Response represents the response from an agent
type Response struct {
	Messages         []llm.Message
	Agent            *Agent
	ContextVariables map[string]interface{}
}

// Result represents the result of a function execution
type Result struct {
	Success bool        // Whether the function execution was successful
	Data    interface{} // Any data returned by the function
	Error   error       // Any error that occurred during execution
	Agent   *Agent      // Active agent
}
