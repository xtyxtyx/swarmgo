package main

import (
	"context"
	"fmt"
	swarmgo "github.com/prathyushnallamothu/swarmgo"
	openai "github.com/sashabaranov/go-openai"
)

func main() {

	client := swarmgo.NewSwarmWithConfig(swarmgo.ClientConfig{
		AuthToken: "sk-",
		BaseURL:   "https://api.deepseek.com",
	})

	agent := &swarmgo.Agent{
		Name:         "Agent",
		Instructions: "You are a helpful agent.",
		Model:        "deepseek-chat",
	}

	messages := []openai.ChatCompletionMessage{
		{Role: "user", Content: "Hi!"},
	}

	ctx := context.Background()
	response, err := client.Run(ctx, agent, messages, nil, "", false, false, 5, true)
	if err != nil {
		panic(err)
	}

	fmt.Println(response.Messages[len(response.Messages)-1].Content)
}
