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
		ProcessAndPrintResponse(response)
		
		// Only keep the last assistant message if it has content
		for i := len(response.Messages) - 1; i >= 0; i-- {
			if response.Messages[i].Role == llm.RoleAssistant && response.Messages[i].Content != "" {
				messages = append(messages, response.Messages[i])
				break
			}
		}

		if response.Agent != nil && response.Agent.Name != activeAgent.Name {
			fmt.Printf("Transferring conversation to %s.\n", response.Agent.Name)
			activeAgent = response.Agent
		}
	}
}
