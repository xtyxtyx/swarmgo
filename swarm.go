package swarmgo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	openai "github.com/sashabaranov/go-openai"
)

type Swarm struct {
	client *openai.Client
}

func NewSwarm(apiKey string) *Swarm {
	client := openai.NewClient(apiKey)
	return &Swarm{
		client: client,
	}
}

func (s *Swarm) getChatCompletion(
	ctx context.Context,
	agent *Agent,
	history []openai.ChatCompletionMessage,
	contextVariables map[string]interface{},
	modelOverride string,
	stream bool,
	debug bool,
) (openai.ChatCompletionResponse, error) {

	// Prepare the messages
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

	// Build function definitions
	var functionDefs []openai.FunctionDefinition
	for _, af := range agent.Functions {
		def := FunctionToDefinition(af)
		functionDefs = append(functionDefs, def)
	}

	// Prepare the request
	model := agent.Model
	if modelOverride != "" {
		model = modelOverride
	}
	req := openai.ChatCompletionRequest{
		Model:     model,
		Messages:  messages,
		Functions: functionDefs,
	}

	if debug {
		log.Printf("Getting chat completion for: %+v\n", messages)
	}

	// Call the API
	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return openai.ChatCompletionResponse{}, err
	}

	return resp, nil
}

func (s *Swarm) handleFunctionCall(
	ctx context.Context,
	functionCall *openai.FunctionCall,
	agent *Agent,
	contextVariables map[string]interface{},
	debug bool,
) (Response, error) {
	functionName := functionCall.Name
	argsJSON := functionCall.Arguments

	var args map[string]interface{}
	if err := json.Unmarshal([]byte(argsJSON), &args); err != nil {
		return Response{}, err
	}

	if debug {
		log.Printf("Processing function call: %s with arguments %v\n", functionName, args)
	}

	// Find the function
	var functionFound *AgentFunction
	for _, af := range agent.Functions {
		if af.Name == functionName {
			functionFound = &af
			break
		}
	}

	if functionFound == nil {
		errorMessage := fmt.Sprintf("Error: Tool %s not found.", functionName)
		if debug {
			log.Println(errorMessage)
		}
		return Response{
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    "tool",
					Name:    functionName,
					Content: errorMessage,
				},
			},
		}, nil
	}

	// Call the function
	result := functionFound.Function(args, contextVariables)

	// Update context variables
	for k, v := range result.ContextVariables {
		contextVariables[k] = v
	}

	// Create function result message
	functionResultMessage := openai.ChatCompletionMessage{
		Role:    openai.ChatMessageRoleFunction,
		Name:    functionName,
		Content: result.Value,
	}

	// Collect the partial response
	partialResponse := Response{
		Messages:         []openai.ChatCompletionMessage{functionResultMessage},
		Agent:            result.Agent,
		ContextVariables: result.ContextVariables,
	}

	return partialResponse, nil
}

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

	for turns < maxTurns && activeAgent != nil {
		turns++

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

		message.Role = openai.ChatMessageRoleAssistant
		message.Name = activeAgent.Name

		history = append(history, message)

		// Keep handling function calls
		for {
			if message.FunctionCall != nil && executeTools {
				// Handle function call
				partialResponse, err := s.handleFunctionCall(
					ctx,
					message.FunctionCall,
					activeAgent,
					contextVariables,
					debug,
				)
				if err != nil {
					return Response{}, err
				}

				history = append(history, partialResponse.Messages...)
				for k, v := range partialResponse.ContextVariables {
					contextVariables[k] = v
				}
				if partialResponse.Agent != nil {
					activeAgent = partialResponse.Agent
				}

				// Get the assistant's response after function result
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
				// No more function calls
				break
			}
		}

		// Break the outer loop if the assistant didn't make a function call
		if message.FunctionCall == nil || !executeTools {
			if debug {
				log.Println("Ending turn.")
			}
			break
		}
	}

	return Response{
		Messages:         history[initLen:],
		Agent:            activeAgent,
		ContextVariables: contextVariables,
	}, nil
}
