package swarmgo

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

func RunDemoLoop(client *Swarm, agent *Agent) {
	// Create a new context for the operation
	ctx := context.Background()
	
	// Print a starting message to the console
	fmt.Println("Starting Swarm CLI 🐝")

	// Initialize a slice to store chat messages
	messages := []openai.ChatCompletionMessage{}
	
	// Create a new reader to read user input from the standard input
	reader := bufio.NewReader(os.Stdin)

	// Start an infinite loop to continuously read user input and process it
	for {
		// Prompt the user for input
		fmt.Print("\033[90mUser\033[0m: ")
		
		// Read the user's input from the console
		userInput, _ := reader.ReadString('\n')
		
		// Trim any leading or trailing whitespace from the input
		userInput = strings.TrimSpace(userInput)
		
		// Append the user's input as a new message to the messages slice
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    "user",
			Content: userInput,
		})

		// Call the client's Run method to process the messages and get a response
		response, err := client.Run(ctx, agent, messages, nil, "", false, false, 5, true)
		
		// If there is an error, log it and continue to the next iteration
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		// Process the response and print it to the console
		ProcessAndPrintResponse(response)
		
		// Append the response messages to the messages slice
		messages = append(messages, response.Messages...)
		
		// Update the agent with the new state from the response
		agent = response.Agent
	}
}
