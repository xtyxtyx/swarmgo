package swarmgo

import (
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

func FunctionToDefinition(af AgentFunction) openai.FunctionDefinition {
	return openai.FunctionDefinition{
		Name:        af.Name,
		Description: af.Description,
		Parameters:  af.Parameters, // Assign directly without marshaling
	}
}
func ProcessAndPrintResponse(response Response) {
	for _, message := range response.Messages {
		if message.Role == "assistant" {
			fmt.Printf("\033[94m%s\033[0m: %s\n", message.Name, message.Content)
		} else if message.Role == "function" || message.Role == "tool" {
			fmt.Printf("\033[95mFunction %s\033[0m Output: %s\n", message.Name, message.Content)
		}
	}
}
