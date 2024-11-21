package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/prathyushnallamothu/swarmgo"
	"github.com/sashabaranov/go-openai"
)

// CustomStreamHandler handles streaming responses with progress updates
type CustomStreamHandler struct {
	totalTokens int
}

func (h *CustomStreamHandler) OnStart() {
	fmt.Println("🚀 Starting file analysis...")
}

func (h *CustomStreamHandler) OnToken(token string) {
	h.totalTokens++
	// Print progress every 20 tokens
	if h.totalTokens%20 == 0 {
		fmt.Printf("📝 Processing... (%d tokens)\n", h.totalTokens)
	}
	fmt.Print(token)
}

func (h *CustomStreamHandler) OnComplete(msg openai.ChatCompletionMessage) {
	fmt.Printf("\n\n✅ Analysis complete! Total tokens processed: %d\n", h.totalTokens)
}

func (h *CustomStreamHandler) OnError(err error) {
	fmt.Printf("❌ Error: %v\n", err)
}

func (h *CustomStreamHandler) OnToolCall(tool openai.ToolCall) {
	fmt.Printf("\n🔧 Using tool: %s\n", tool.Function.Name)
}

// FileProcessor handles file operations
type FileProcessor struct{}

func (fp *FileProcessor) ReadFile(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("error reading file: %v", err)
	}
	return string(content), nil
}

func (fp *FileProcessor) CountWords(text string) int {
	words := strings.Fields(text)
	return len(words)
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	fileProcessor := &FileProcessor{}

	// Define agent functions
	functions := []swarmgo.AgentFunction{
		{
			Name:        "read_file",
			Description: "Read content from a file",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"path": map[string]interface{}{
						"type":        "string",
						"description": "Path to the file to read",
					},
				},
				"required": []string{"path"},
			},
			Function: func(args map[string]interface{}, context map[string]interface{}) swarmgo.Result {
				path, _ := args["path"].(string)
				content, err := fileProcessor.ReadFile(path)
				if err != nil {
					return swarmgo.Result{
						Value: fmt.Sprintf("Error reading file: %v", err),
					}
				}
				return swarmgo.Result{
					Value: content,
				}
			},
		},
		{
			Name:        "count_words",
			Description: "Count words in a text",
			Parameters: map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"text": map[string]interface{}{
						"type":        "string",
						"description": "Text to analyze",
					},
				},
				"required": []string{"text"},
			},
			Function: func(args map[string]interface{}, context map[string]interface{}) swarmgo.Result {
				text, _ := args["text"].(string)
				wordCount := fileProcessor.CountWords(text)
				return swarmgo.Result{
					Value: strconv.Itoa(wordCount),
				}
			},
		},
	}

	// Create a new agent with file analysis capabilities
	agent := &swarmgo.Agent{
		Name: "FileAnalyzer",
		Instructions: `You are a file analysis assistant. Your tasks include:
1. Reading and analyzing text files
2. Providing detailed summaries of file content
3. Analyzing text structure and statistics
4. Identifying key themes and patterns

When analyzing files:
- Start with a brief overview
- Break down the content into meaningful sections
- Provide word counts and other relevant metrics
- Highlight important findings
- Maintain a professional and clear communication style`,
		Functions: functions,
		Model:     openai.GPT4TurboPreview,
	}

	// Create a new swarm instance
	swarm := swarmgo.NewSwarm(os.Getenv("OPENAI_API_KEY"))

	// Create a custom stream handler
	handler := &CustomStreamHandler{}

	// Example message to analyze a file
	messages := []openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleUser,
			Content: "Please analyze the content of README.md in the current directory. Provide a detailed summary with word count and key points.",
		},
	}

	// Start streaming analysis
	ctx := context.Background()
	if err := swarm.StreamingResponse(ctx, agent, messages, nil, "", handler, true); err != nil {
		log.Fatalf("Error in streaming response: %v", err)
	}
}
