# SwarmGo

SwarmGo is a Go package that allows you to create AI agents capable of interacting, coordinating, and executing tasks. Inspired by OpenAI's Swarm framework, SwarmGo focuses on making agent coordination and execution lightweight, highly controllable, and easily testable.

It achieves this through two primitive abstractions: Agents and handoffs. An Agent encompasses instructions and tools (functions it can execute), and can at any point choose to hand off a conversation to another Agent.

These primitives are powerful enough to express rich dynamics between tools and networks of agents, allowing you to build scalable, real-world solutions while avoiding a steep learning curve.

## Table of Contents

- Why SwarmGo
- Installation
- Quick Start
- Usage
- Creating an Agent
- Running the Agent
- Adding Functions (Tools)
- Using Context Variables
- Agent Handoff
- Examples
- Contributing
- License

## Why SwarmGo

SwarmGo explores patterns that are lightweight, scalable, and highly customizable by design. It's best suited for situations dealing with a large number of independent capabilities and instructions that are difficult to encode into a single prompt.

SwarmGo runs (almost) entirely on the client and, much like the Chat Completions API, does not store state between calls.

## Installation

```bash
go get github.com/prathyushnallamothu/swarmgo
```

## Quick Start
Here's a simple example to get you started:

```go
package main

import (
	"context"
	"fmt"
	"log"

	swarmgo "github.com/prathyushnallamothu/swarmgo"
	openai "github.com/sashabaranov/go-openai"
)

func main() {
	client := swarmgo.NewSwarm("YOUR_OPENAI_API_KEY")

	agent := &swarmgo.Agent{
		Name:         "Agent",
		Instructions: "You are a helpful assistant.",
		Model:        "gpt-3.5-turbo",
	}

	messages := []openai.ChatCompletionMessage{
		{Role: "user", Content: "Hello!"},
	}

	ctx := context.Background()
	response, err := client.Run(ctx, agent, messages, nil, "", false, false, 5, true)
	if err != nil {
		log.Fatalf("Error: %v", err)
	}

	fmt.Println(response.Messages[len(response.Messages)-1].Content)
}
```

## Usage
Creating an Agent
An Agent represents an AI assistant with specific instructions and functions (tools) it can use.

```go
agent := &swarmgo.Agent{
	Name:         "Agent",
	Instructions: "You are a helpful assistant.",
	Model:        "gpt-4o",
}
```

Name: Identifier for the agent (no spaces or special characters).
Instructions: The system prompt or instructions for the agent.
Model: The OpenAI model to use (e.g., "gpt-4o").

## Running the Agent

To interact with the agent, use the Run method:

```go
messages := []openai.ChatCompletionMessage{
	{Role: "user", Content: "Hello!"},
}

ctx := context.Background()
response, err := client.Run(ctx, agent, messages, nil, "", false, false, 5, true)
if err != nil {
	log.Fatalf("Error: %v", err)
}

fmt.Println(response.Messages[len(response.Messages)-1].Content)
```

## Adding Functions (Tools)

Agents can use functions to perform specific tasks. Functions are defined and then added to an agent.

### Defining a Function

```go
func getWeather(args map[string]interface{}, contextVariables map[string]interface{}) swarmgo.Result {
	location := args["location"].(string)
	// Simulate fetching weather data
	return swarmgo.Result{
		Value: fmt.Sprintf(`{"temp": 67, "unit": "F", "location": "%s"}`, location),
	}
}
```

### Adding the Function to an Agent

```go
agent.Functions = []swarmgo.AgentFunction{
	{
		Name:        "getWeather",
		Description: "Get the current weather in a given location.",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"location": map[string]interface{}{
					"type":        "string",
					"description": "The city to get the weather for.",
				},
			},
			"required": []interface{}{"location"},
		},
		Function: getWeather,
	},
}
```
## Using Context Variables

Context variables allow you to pass information between function calls and agents.

### Using Context Variables in Instructions

```go
func instructions(contextVariables map[string]interface{}) string {
	name, ok := contextVariables["name"].(string)
	if !ok {
		name = "User"
	}
	return fmt.Sprintf("You are a helpful assistant. Greet the user by name (%s).", name)
}

agent.InstructionsFunc = instructions
```

## Agent Handoff

Agents can hand off conversations to other agents. This is useful for delegating tasks or escalating when an agent is unable to handle a request.

```go
func transferToAnotherAgent(args map[string]interface{}, contextVariables map[string]interface{}) swarmgo.Result {
	anotherAgent := &swarmgo.Agent{
		Name:         "AnotherAgent",
		Instructions: "You are another agent.",
		Model:        "gpt-3.5-turbo",
	}
	return swarmgo.Result{
		Agent: anotherAgent,
		Value: "Transferring to AnotherAgent.",
	}
}
```

### Adding Handoff Functions to Agents

```go
agent.Functions = append(agent.Functions, swarmgo.AgentFunction{
	Name:        "transferToAnotherAgent",
	Description: "Transfer the conversation to AnotherAgent.",
	Parameters: map[string]interface{}{
		"type":       "object",
		"properties": map[string]interface{}{},
	},
	Function: transferToAnotherAgent,
})
```

## Examples

For more examples, see the [examples](examples) directory.

## Contributing
Contributions are welcome! Please follow these steps:

1. Fork the repository.
2. Create a new branch (git checkout -b feature/YourFeature).
3. Commit your changes (git commit -am 'Add a new feature').
4. Push to the branch (git push origin feature/YourFeature).
5. Open a Pull Request.

## License

This project is licensed under the MIT License. See the [LICENSE](LICENSE) file for details.

## Acknowledgments

- Thanks to OpenAI for the inspiration and the [Swarm framework](https://github.com/openai/swarm).
- Thanks to [Sashabaranov](https://github.com/sashabaranov) for the [go-openai](https://github.com/sashabaranov/go-openai) package.
