package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	swarmgo "github.com/prathyushnallamothu/swarmgo"
)

// Define worker functions
func researchFunction(args map[string]interface{}, contextVariables map[string]interface{}) swarmgo.Result {
	topic := args["topic"].(string)
	fmt.Printf("Researching topic: %s\n", topic)
	return swarmgo.Result{
		Value: fmt.Sprintf("Research completed on %s. Here are the findings...", topic),
	}
}

func codeFunction(args map[string]interface{}, contextVariables map[string]interface{}) swarmgo.Result {
	code := args["code"].(string)
	fmt.Printf("Executing code: %s\n", code)
	return swarmgo.Result{
		Value: fmt.Sprintf("Code executed successfully: %s", code),
	}
}

func main() {
	if err := godotenv.Load(); err != nil {
		log.Printf("Warning: Error loading .env file: %v", err)
	}

	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		log.Fatal("OPENAI_API_KEY environment variable is required")
	}

	// Create the swarm client
	client := swarmgo.NewSwarm(apiKey)

	// Create supervisor
	supervisor := swarmgo.NewSupervisorAgent("MainSupervisor", "gpt-4")
	supervisor.RoutingConfig.LoadBalancing = true
	supervisor.RoutingConfig.MaxRetries = 3

	// Create a researcher using GPT-4
	researcherConfig := swarmgo.WorkerConfig{
		Name:  "Researcher",
		Role:  "research",
		Model: "gpt-4",
		Capabilities: []swarmgo.WorkerCapability{
			{
				Name:     "research",
				Keywords: []string{"research", "analyze", "investigate"},
				Priority: 1,
			},
		},
		Functions: []swarmgo.AgentFunction{
			{
				Name:        "research",
				Description: "Research a given topic thoroughly",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"topic": map[string]interface{}{
							"type":        "string",
							"description": "The topic to research",
						},
					},
					"required": []string{"topic"},
				},
				Function: researchFunction,
			},
		},
		Instructions: "You are a research specialist focused on gathering and analyzing information.",
	}
	researchAgent := swarmgo.NewWorkerAgent(researcherConfig)

	// Create a coder using GPT-3.5-turbo
	coderConfig := swarmgo.WorkerConfig{
		Name:  "Coder",
		Role:  "coding",
		Model: "gpt-3.5-turbo",
		Capabilities: []swarmgo.WorkerCapability{
			{
				Name:     "coding",
				Keywords: []string{"code", "program", "function", "implement"},
				Priority: 1,
			},
		},
		Functions: []swarmgo.AgentFunction{
			{
				Name:        "code",
				Description: "Write and execute code",
				Parameters: map[string]interface{}{
					"type": "object",
					"properties": map[string]interface{}{
						"code": map[string]interface{}{
							"type":        "string",
							"description": "The code to execute",
						},
					},
					"required": []string{"code"},
				},
				Function: codeFunction,
			},
		},
		Instructions: "You are a coding specialist focused on writing and implementing code.",
	}
	coderAgent := swarmgo.NewWorkerAgent(coderConfig)

	supervisor.AddWorker(researchAgent)
	supervisor.AddWorker(coderAgent)

	// Example tasks
	tasks := []string{
		"Implement a sorting algorithm in Go",
	}

	// Process tasks
	for _, taskDesc := range tasks {
		fmt.Printf("\nProcessing task: %s\n", taskDesc)

		result, err := supervisor.Coordinate(context.Background(), client, taskDesc)
		if err != nil {
			log.Printf("Error processing task: %v", err)
			continue
		}

		fmt.Printf("Task completed. Result: %s\n", result)
		time.Sleep(time.Second) // Small delay between tasks
	}

	// Print task history with more details
	fmt.Println("\nDetailed Task History:")
	for _, task := range supervisor.TaskHistory {
		fmt.Printf("Task ID: %s\n", task.ID)
		fmt.Printf("Description: %s\n", task.Description)
		fmt.Printf("Type: %s\n", task.Type)
		fmt.Printf("Assigned To: %s\n", task.AssignedTo)
		fmt.Printf("Status: %s\n", task.Status)
		fmt.Printf("Attempts: %d\n", task.Attempts)
		fmt.Printf("Created: %s\n", task.CreatedAt)
		if task.CompletedAt != (time.Time{}) {
			fmt.Printf("Completed: %s\n", task.CompletedAt)
			fmt.Printf("Duration: %s\n", task.CompletedAt.Sub(task.CreatedAt))
		}
		fmt.Println("---")
	}

	// Print worker statistics
	fmt.Println("\nWorker Statistics:")
	for name, worker := range supervisor.Workers {
		fmt.Printf("\nWorker: %s\n", name)
		fmt.Printf("Role: %s\n", worker.Role)
		fmt.Printf("Tasks Completed: %d\n", worker.Performance.TasksCompleted)
		fmt.Printf("Tasks Failed: %d\n", worker.Performance.TasksFailed)
		fmt.Printf("Success Rate: %.2f%%\n", worker.Performance.SuccessRate*100)
		fmt.Printf("Average Time: %s\n", worker.Performance.AverageTime)
	}
}
