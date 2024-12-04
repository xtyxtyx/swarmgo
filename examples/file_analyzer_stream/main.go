package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
	"github.com/prathyushnallamothu/swarmgo"
	"github.com/prathyushnallamothu/swarmgo/llm"
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

func (h *CustomStreamHandler) OnComplete(msg llm.Message) {
	fmt.Printf("\n\n✅ Analysis complete! Total tokens processed: %d\n", h.totalTokens)
}

func (h *CustomStreamHandler) OnError(err error) {
	fmt.Printf("❌ Error: %v\n", err)
}

func (h *CustomStreamHandler) OnToolCall(toolCall llm.ToolCall) {
	fmt.Printf("\n🔧 Using tool: %s\n", toolCall.Function.Name)
}

// FileProcessor handles file operations
type FileProcessor struct{}

func (fp *FileProcessor) ReadFile(path string) (string, error) {
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return "", fmt.Errorf("file does not exist: %s", path)
	}
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

	// Get the project root directory
	projectRoot := "/Users/prathyushnallamothu/Desktop/Projects/swarmgo"
	readmePath := filepath.Join(projectRoot, "Readme.md")

	// Verify the file exists
	if _, err := os.Stat(readmePath); os.IsNotExist(err) {
		log.Fatalf("README.md not found at %s", readmePath)
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
				path, ok := args["path"].(string)
				fmt.Println(path)
				if !ok {
					return swarmgo.Result{
						Error: fmt.Errorf("invalid path argument"),
					}
				}
				content, err := fileProcessor.ReadFile(path)
				if err != nil {
					return swarmgo.Result{
						Error: err,
						Data:  fmt.Sprintf("Error reading file: %v", err),
					}
				}
				return swarmgo.Result{
					Data: content,
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
				text, ok := args["text"].(string)
				if !ok {
					return swarmgo.Result{
						Error: fmt.Errorf("invalid text argument"),
					}
				}
				wordCount := fileProcessor.CountWords(text)
				return swarmgo.Result{
					Data: strconv.Itoa(wordCount),
				}
			},
		},
	}

	// Create a new agent with file analysis capabilities
	agent := &swarmgo.Agent{
		Name: "FileAnalyzer",
		Instructions: `You are a file analysis assistant that uses tools to analyze files.

Available tools:
1. read_file: Takes a file path and returns its contents
2. count_words: Takes a text string and returns the word count

Follow these steps exactly:
1. First use read_file to read the content of the file
2. Then use count_words to count the words in the file content
3. Finally, provide a detailed analysis including:
   - Word count
   - Key themes and topics
   - Structure and organization
   - Main points and findings

Always use the tools in this exact order and wait for each tool's result before proceeding.`,
		Functions: functions,
		Model:     openai.GPT4,
	}

	// Create a new swarm instance
	swarm := swarmgo.NewSwarm(os.Getenv("OPENAI_API_KEY"), llm.OpenAI)

	// Create a custom stream handler
	handler := &CustomStreamHandler{}

	// Example message to analyze a file
	messages := []llm.Message{
		{
			Role:    llm.RoleUser,
			Content: fmt.Sprintf("Please analyze %s by following these steps:\n1. Use read_file to read %s\n2. Use count_words on the file content\n3. Provide a detailed analysis", readmePath, readmePath),
		},
	}

	// Start streaming analysis
	ctx := context.Background()
	if err := swarm.StreamingResponse(ctx, agent, messages, nil, "", handler, true); err != nil {
		log.Fatalf("Error in streaming response: %v", err)
	}
}
