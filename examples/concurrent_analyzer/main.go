package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	dotenv "github.com/joho/godotenv"
	"github.com/prathyushnallamothu/swarmgo"
	"github.com/sashabaranov/go-openai"
)

// CodeAnalysisResult represents the analysis result from an agent
type CodeAnalysisResult struct {
	Category    string   `json:"category"`
	Findings    []string `json:"findings"`
	Suggestions []string `json:"suggestions"`
}

// createAnalysisAgent creates an agent with specific analysis focus
func createAnalysisAgent(category string, instructions string) *swarmgo.Agent {
	return &swarmgo.Agent{
		Name:         fmt.Sprintf("%sAnalyzer", category),
		Model:        "gpt-4o-mini",
		Instructions: instructions,
		Functions: []swarmgo.AgentFunction{
			{
				Name:        "report_findings",
				Description: "Report analysis findings and suggestions",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"findings": map[string]interface{}{
							"type":        "array",
							"description": "List of findings from the analysis",
							"items": map[string]interface{}{
								"type": "string",
							},
						},
						"suggestions": map[string]interface{}{
							"type":        "array",
							"description": "List of suggestions for improvement",
							"items": map[string]interface{}{
								"type": "string",
							},
						},
					},
					"required": []interface{}{"findings", "suggestions"},
				},
				Function: func(args map[string]interface{}, contextVars map[string]interface{}) swarmgo.Result {
					findings := args["findings"].([]interface{})
					suggestions := args["suggestions"].([]interface{})

					// Convert interface slices to string slices
					findingsStr := make([]string, len(findings))
					for i, f := range findings {
						findingsStr[i] = f.(string)
					}
					suggestionsStr := make([]string, len(suggestions))
					for i, s := range suggestions {
						suggestionsStr[i] = s.(string)
					}

					result := CodeAnalysisResult{
						Category:    category,
						Findings:    findingsStr,
						Suggestions: suggestionsStr,
					}
					jsonResult, err := json.Marshal(result)
					if err != nil {
						return swarmgo.Result{Value: fmt.Sprintf("Error marshaling result: %v", err)}
					}
					return swarmgo.Result{Value: string(jsonResult)}
				},
			},
		},
	}
}

// AnalyzeCodebase performs concurrent analysis of a codebase using multiple specialized agents
func AnalyzeCodebase(apiKey string, codePath string) error {
	// Create a concurrent swarm
	cs := swarmgo.NewConcurrentSwarm(apiKey)

	// Create specialized analysis agents
	securityAgent := createAnalysisAgent("Security", `You are a security-focused code analyzer. 
		Analyze the code for security vulnerabilities, unsafe practices, and potential threats. 
		Look for issues like:
		- Insecure data handling
		- Authentication/authorization weaknesses
		- Input validation issues
		- Sensitive data exposure
		Report findings and provide specific suggestions for improvement using the report_findings function. Make sure to respond in JSON format ONLY as follows start with {}:
		{
			"category": "Security",
			"findings": ["Finding 1", "Finding 2"],
			"suggestions": ["Suggestion 1", "Suggestion 2"]
		}`)

	performanceAgent := createAnalysisAgent("Performance", `You are a performance-focused code analyzer.
		Analyze the code for performance bottlenecks and optimization opportunities.
		Look for issues like:
		- Inefficient algorithms
		- Resource leaks
		- Unnecessary computations
		- Concurrency issues
		Report findings and provide specific suggestions for improvement using the report_findings function. Make sure to respond in JSON format ONLY as follows start with {}:
		{
			"category": "Performance",
			"findings": ["Finding 1", "Finding 2"],
			"suggestions": ["Suggestion 1", "Suggestion 2"]
		}`)

	maintainabilityAgent := createAnalysisAgent("Maintainability", `You are a maintainability-focused code analyzer.
		Analyze the code for maintainability and code quality issues.
		Look for issues like:
		- Code duplication
		- Complex or unclear logic
		- Poor documentation
		- Violation of SOLID principles
		Report findings and provide specific suggestions for improvement using the report_findings function. Make sure to respond in JSON format ONLY as follows start with {}:
		{
			"category": "Maintainability",
			"findings": ["Finding 1", "Finding 2"],
			"suggestions": ["Suggestion 1", "Suggestion 2"]
		}`)

	// Read and prepare code content
	var codeContent string
	err := filepath.Walk(codePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() && strings.HasSuffix(path, ".go") {
			content, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			codeContent += fmt.Sprintf("=== %s ===\n%s\n\n", path, string(content))
		}
		return nil
	})
	if err != nil {
		return fmt.Errorf("error reading codebase: %v", err)
	}

	// Configure agents for concurrent execution
	configs := map[string]swarmgo.AgentConfig{
		"security": {
			Agent: securityAgent,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Analyze this codebase for security issues:\n\n%s", codeContent),
				},
			},
			MaxTurns:     1,
			ExecuteTools: true,
		},
		"performance": {
			Agent: performanceAgent,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Analyze this codebase for performance issues:\n\n%s", codeContent),
				},
			},
			MaxTurns:     1,
			ExecuteTools: true,
		},
		"maintainability": {
			Agent: maintainabilityAgent,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleUser,
					Content: fmt.Sprintf("Analyze this codebase for maintainability issues:\n\n%s", codeContent),
				},
			},
			MaxTurns:     1,
			ExecuteTools: true,
		},
	}

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	// Run concurrent analysis
	fmt.Println("Starting concurrent code analysis...")
	startTime := time.Now()

	results := cs.RunConcurrent(ctx, configs)

	// Process and display results
	fmt.Printf("\nAnalysis completed in %v\n\n", time.Since(startTime))

	for _, result := range results {
		if result.Error != nil {
			log.Printf("Error in %s analysis: %v\n", result.AgentName, result.Error)
			continue
		}

		var analysisResult CodeAnalysisResult
		if err := json.Unmarshal([]byte(result.Response.Messages[len(result.Response.Messages)-1].Content), &analysisResult); err != nil {
			log.Printf("Error parsing %s analysis results: %v\nRaw response: %s\n",
				result.AgentName, err, result.Response.Messages[len(result.Response.Messages)-1].Content)
			continue
		}

		fmt.Printf("=== %s Analysis ===\n", analysisResult.Category)
		fmt.Println("\nFindings:")
		for _, finding := range analysisResult.Findings {
			fmt.Printf("- %s\n", finding)
		}

		fmt.Println("\nSuggestions:")
		for _, suggestion := range analysisResult.Suggestions {
			fmt.Printf("- %s\n", suggestion)
		}
		fmt.Println()
	}

	return nil
}

func main() {
	dotenv.Load()
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	err := AnalyzeCodebase(apiKey, ".")
	if err != nil {
		log.Fatalf("Error analyzing codebase: %v", err)
	}
}
