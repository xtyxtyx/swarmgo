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
	ctx := context.Background()
	fmt.Println("Starting Swarm CLI 🐝")

	messages := []openai.ChatCompletionMessage{}
	reader := bufio.NewReader(os.Stdin)

	for {
		fmt.Print("\033[90mUser\033[0m: ")
		userInput, _ := reader.ReadString('\n')
		userInput = strings.TrimSpace(userInput)
		messages = append(messages, openai.ChatCompletionMessage{
			Role:    "user",
			Content: userInput,
		})

		response, err := client.Run(ctx, agent, messages, nil, "", false, false, 5, true)
		if err != nil {
			log.Printf("Error: %v", err)
			continue
		}

		ProcessAndPrintResponse(response)
		messages = append(messages, response.Messages...)
		agent = response.Agent
	}
}


