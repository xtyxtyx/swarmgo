package swarmgo

import (
	"context"
	"encoding/json"
	"fmt"
	openai "github.com/sashabaranov/go-openai"
	"log"
)

// OpenAIClient defines the methods used from the OpenAI client
type OpenAIClient interface {
	CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error)
	CreateChatCompletionStream(ctx context.Context, req openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error)
}

// Swarm represents the main structure
type Swarm struct {
	client OpenAIClient
}

// NewSwarm initializes a new Swarm instance with an OpenAI client
func NewSwarm(apiKey string) *Swarm {
	client := openai.NewClient(apiKey)
	return &Swarm{
		client: client,
	}
}

func NewSwarmWithConfig(config ClientConfig) *Swarm {
	openaiConfig := openai.DefaultConfig(config.AuthToken)
	openaiConfig.BaseURL = config.BaseURL
	client := openai.NewClientWithConfig(openaiConfig)
	return &Swarm{
		client: client,
	}
}

// getChatCompletion requests a chat completion from the OpenAI API
func (s *Swarm) getChatCompletion(
	ctx context.Context,
	agent *Agent,
	history []openai.ChatCompletionMessage,
	contextVariables map[string]interface{},
	modelOverride string,
	stream bool,
	debug bool,
) (openai.ChatCompletionResponse, error) {

	// Prepare the initial system message with agent instructions
	instructions := agent.Instructions
	if agent.InstructionsFunc != nil {
		instructions = agent.InstructionsFunc(contextVariables)
	}
	messages := append([]openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: instructions,
		},
	}, history...)

	// Build tool definitions from agent's functions
	var toolDefs []openai.FunctionDefinition
	var tools []openai.Tool
	for _, af := range agent.Functions {
		def := FunctionToDefinition(af)
		toolDefs = append(toolDefs, def)
		tools = append(tools, openai.Tool{
			Type:     "function",
			Function: &def,
		})
	}

	// Prepare the chat completion request
	model := agent.Model
	if modelOverride != "" {
		model = modelOverride
	}
	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
	}

	if debug {
		log.Printf("Getting chat completion for: %+v\n", messages)
	}

	// Call the OpenAI API to get a chat completion
	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}

	return resp, nil
}

// handleToolCall processes a tool call from the chat completion
func (s *Swarm) handleToolCall(
	ctx context.Context,
	toolCall *openai.ToolCall,
	agent *Agent,
	contextVariables map[string]interface{},
	debug bool,
) (Response, error) {
	toolName := toolCall.Function.Name
	argsJSON := toolCall.Function.Arguments
	toolCallID := toolCall.ID // Extract the tool_call_id

	// Parse the tool call arguments
	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Response{}, err
	}

	if debug {
		log.Printf("Processing tool call: %s with arguments %v\n", toolName, args)
	}

	// Find the corresponding function in the agent's functions
	var functionFound *AgentFunction
	for _, af := range agent.Functions {
		if af.Name == toolName {
			functionFound = &af
			break
		}
	}

	// Handle case where function is not found
	if functionFound == nil {
		errorMessage := fmt.Sprintf("Error: Tool %s not found.", toolName)
		if debug {
			log.Println(errorMessage)
		}
		return Response{
			Messages: []openai.ChatCompletionMessage{
				{
					Role:       "tool",
					Name:       toolName,
					Content:    errorMessage,
					ToolCallID: toolCallID, // Include the tool_call_id here
				},
			},
		}, nil
	}

	// Execute the function and update context variables
	result := functionFound.Function(args, contextVariables)
	for k, v := range result.ContextVariables {
		contextVariables[k] = v
	}

	// Create a message with the tool result
	toolResultMessage := openai.ChatCompletionMessage{
		Role:       openai.ChatMessageRoleTool,
		Name:       toolName,
		Content:    result.Value,
		ToolCallID: toolCallID, // Include the tool_call_id here
	}

	// Return the partial response with the tool result
	partialResponse := Response{
		Messages:         []openai.ChatCompletionMessage{toolResultMessage},
		Agent:            result.Agent,
		ContextVariables: result.ContextVariables,
	}

	return partialResponse, nil
}

// Run executes the chat interaction loop with the agent
func (s *Swarm) Run(
	ctx context.Context,
	agent *Agent,
	messages []openai.ChatCompletionMessage,
	contextVariables map[string]interface{},
	modelOverride string,
	stream bool,
	debug bool,
	maxTurns int,
	executeTools bool,
) (Response, error) {
	activeAgent := agent
	history := make([]openai.ChatCompletionMessage, len(messages))
	copy(history, messages)
	if contextVariables == nil {
		contextVariables = make(map[string]interface{})
	}

	initLen := len(messages)
	turns := 0

	// Main loop for chat interaction
	for turns < maxTurns && activeAgent != nil {
		turns++

		// Get a chat completion from the API
		resp, err := s.getChatCompletion(
			ctx,
			activeAgent,
			history,
			contextVariables,
			modelOverride,
			stream,
			debug,
		)
		if err != nil {
			return Response{}, err
		}

		if len(resp.Choices) == 0 {
			return Response{}, fmt.Errorf("no choices in response")
		}

		choice := resp.Choices[0]
		message := choice.Message

		if debug {
			log.Printf("Received completion: %+v\n", message)
		}

		// Update message role and name
		message.Role = openai.ChatMessageRoleAssistant
		message.Name = activeAgent.Name

		history = append(history, message)

		// Handle tool calls if any
		// Inside the main loop in Run()
		for {
			if len(message.ToolCalls) > 0 && executeTools {
				for _, toolCall := range message.ToolCalls {
					// Process each tool call
					partialResponse, err := s.handleToolCall(
						ctx,
						&toolCall,
						activeAgent,
						contextVariables,
						debug,
					)
					if err != nil {
						return Response{}, err
					}

					history = append(history, partialResponse.Messages...) // Include the tool_call_id here
					for k, v := range partialResponse.ContextVariables {
						contextVariables[k] = v
					}
					if partialResponse.Agent != nil {
						activeAgent = partialResponse.Agent
					}
				}

				// Get the assistant's response after all tool results
				resp, err := s.getChatCompletion(
					ctx,
					activeAgent,
					history,
					contextVariables,
					modelOverride,
					stream,
					debug,
				)
				if err != nil {
					return Response{}, err
				}

				if len(resp.Choices) == 0 {
					return Response{}, fmt.Errorf("no choices in response")
				}

				choice = resp.Choices[0]
				message = choice.Message

				if debug {
					log.Printf("Received completion: %+v\n", message)
				}

				message.Role = openai.ChatMessageRoleAssistant
				message.Name = activeAgent.Name

				history = append(history, message)
			} else {
				// Exit the loop if no more tool calls
				break
			}
		}

		// Break the outer loop if the assistant didn't make a tool call
		if message.FunctionCall == nil || !executeTools {
			if debug {
				log.Println("Ending turn.")
			}
			break
		}
	}

	// Return the final response
	return Response{
		Messages:         history[initLen:],
		Agent:            activeAgent,
		ContextVariables: contextVariables,
	}, nil
}
