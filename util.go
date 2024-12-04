package swarmgo

import (
	"fmt"
)

// ProcessAndPrintResponse processes a Response object and prints messages based on their role.
// It uses different colors for different roles: blue for "assistant" and magenta for "function" or "tool".
func ProcessAndPrintResponse(response Response) {
	for _, message := range response.Messages {
		if message.Role == "assistant" {
			// Print assistant messages in blue, use agent name if available
			name := "Assistant"
			if response.Agent != nil && response.Agent.Name != "" {
				name = response.Agent.Name
			}
			if message.Content != "" {
				fmt.Printf("\033[94m%s\033[0m: %s\n", name, message.Content)
			}
		} else if message.Role == "function" || message.Role == "tool" {
			// Print function or tool messages in magenta
			fmt.Printf("\033[95m%s\033[0m: %s\n", message.Name, message.Content)
		}
	}
}
