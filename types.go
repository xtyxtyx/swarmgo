package swarmgo

import (
	openai "github.com/sashabaranov/go-openai"
)

type Response struct {
	Messages         []openai.ChatCompletionMessage
	Agent            *Agent
	ContextVariables map[string]interface{}
}

type Result struct {
	Value            string
	Agent            *Agent
	ContextVariables map[string]interface{}
}
