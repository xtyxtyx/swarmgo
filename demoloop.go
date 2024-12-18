package swarmgo

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/prathyushnallamothu/swarmgo/llm"
)

func RunDemoLoop(client *Swarm, agent *Agent) {
	// Create a new context for the operation
	ctx := context.Background()

	// Print a starting message to the console
	fmt.Println("Starting Swarm CLI ")

	// Initialize a slice to store chat messages
	messages := []llm.Message{}

	// Create a new reader to read user input from the standard input
	reader := bufio.NewReader(os.Stdin)

	activeAgent := agent

	for {
		// Prompt the user for input
		fmt.Print("\033[90mUser\033[0m: ")

		// Read the user's input from the console
		userInput, _ := reader.ReadString('\n')

		// Trim any leading or trailing whitespace from the input
		userInput = strings.TrimSpace(userInput)

		// Append the user's input as a new message to the messages slice
		messages = append(messages, llm.Message{
			Role:    llm.RoleUser,
			Content: userInput,
		})

		response, err := client.Run(ctx, activeAgent, messages, nil, "", false, false, 5, true)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		// Process the response and print it to the console
		var lastAssistantMessage llm.Message
		var functionMessages []llm.Message
		for _, msg := range response.Messages {
			switch msg.Role {
			case llm.RoleAssistant:
				if msg.Content != "" {
					fmt.Printf("\033[94m%s\033[0m: %s\n", response.Agent.Name, msg.Content)
					lastAssistantMessage = msg
				}
			case llm.RoleFunction:
				fmt.Printf("\033[92m%s function Result\033[0m: %s\n", msg.Name, msg.Content)
				functionMessages = append(functionMessages, msg)
			}
		}

		// Add function results to history first, then the assistant message
		messages = append(messages, functionMessages...)
		messages = append(messages, lastAssistantMessage)

		if response.Agent != nil && response.Agent.Name != activeAgent.Name {
			fmt.Printf("Transferring conversation to %s.\n", response.Agent.Name)
			activeAgent = response.Agent
		}
	}
}
