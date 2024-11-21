package main

import (
	"context"
	"fmt"
	"log"
	"os"

	dotenv "github.com/joho/godotenv"
	swarmgo "github.com/prathyushnallamothu/swarmgo"
	openai "github.com/sashabaranov/go-openai"
)

// CustomStreamHandler implements the StreamHandler interface with custom behavior
type CustomStreamHandler struct {
	tokens []string
}

func (h *CustomStreamHandler) OnStart() {
	fmt.Println("🚀 Starting streaming response...")
}

func (h *CustomStreamHandler) OnToken(token string) {
	h.tokens = append(h.tokens, token)
	fmt.Print(token) // Print tokens as they arrive
}

func (h *CustomStreamHandler) OnToolCall(toolCall openai.ToolCall) {
	fmt.Printf("\n🛠️ Tool called: %s\n", toolCall.Function.Name)
}

func (h *CustomStreamHandler) OnComplete(message openai.ChatCompletionMessage) {
	fmt.Printf("\n✅ Complete message received! Total tokens: %d\n", len(h.tokens))
}

func (h *CustomStreamHandler) OnError(err error) {
	fmt.Printf("\n❌ Error: %v\n", err)
}

func main() {
	dotenv.Load()
	// Get API key from environment variable
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is not set")
	}

	// Initialize the Swarm client
	client := swarmgo.NewSwarm(apiKey)

	// Create an agent with a simple calculator function
	agent := &swarmgo.Agent{
		Name:         "Calculator",
		Instructions: "You are a helpful calculator assistant. Use the calculate function when needed.",
		Model:        "gpt-4o",
		Functions: []swarmgo.AgentFunction{
			{
				Name:        "calculate",
				Description: "Perform a calculation",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"operation": map[string]interface{}{
							"type": "string",
							"enum": []string{"add", "subtract", "multiply", "divide"},
						},
						"x": map[string]interface{}{
							"type": "number",
						},
						"y": map[string]interface{}{
							"type": "number",
						},
					},
					"required": []string{"operation", "x", "y"},
				},
				Function: func(args map[string]interface{}, contextVariables map[string]interface{}) swarmgo.Result {
					op := args["operation"].(string)
					x := args["x"].(float64)
					y := args["y"].(float64)

					var result float64
					var err error

					switch op {
					case "add":
						result = x + y
					case "subtract":
						result = x - y
					case "multiply":
						result = x * y
					case "divide":
						if y == 0 {
							return swarmgo.Result{Value: "Error: division by zero"}
						}
						result = x / y
					default:
						return swarmgo.Result{Value: fmt.Sprintf("Error: unknown operation: %s", op)}
					}

					if err != nil {
						return swarmgo.Result{Value: fmt.Sprintf("Error: %v", err)}
					}

					return swarmgo.Result{Value: fmt.Sprintf("%.2f", result)}
				},
			},
		},
	}

	// Create initial messages
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "What is 42 divided by 56?",
		},
	}

	// Create a custom stream handler
	handler := &CustomStreamHandler{
		tokens: make([]string, 0),
	}

	// Start streaming with context
	ctx := context.Background()
	err := client.StreamingResponse(ctx, agent, messages, nil, "", handler, true)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}
}
