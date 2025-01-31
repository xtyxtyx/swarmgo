package swarmgo

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/prathyushnallamothu/swarmgo/llm"
)

// Swarm represents the main structure
type Swarm struct {
	client llm.LLM
}

// NewSwarm initializes a new Swarm instance with an LLM client
func NewSwarm(apiKey string, provider llm.LLMProvider) *Swarm {
	if provider == llm.OpenAI {
		client := llm.NewOpenAILLM(apiKey)
		return &Swarm{
			client: client,
		}
	}
	if provider == llm.Gemini {
		client, err := llm.NewGeminiLLM(apiKey)
		if err != nil {
			log.Fatalf("Failed to create Gemini client: %v", err)
		}
		return &Swarm{
			client: client,
		}
	}
	if provider == llm.Claude {
		client := llm.NewClaudeLLM(apiKey)

		return &Swarm{
			client: client,
		}
	}
	if provider == llm.Ollama {
		client, err := llm.NewOllamaLLM()
		if err != nil {
			log.Fatalf("Failed to create Ollama client: %v", err)
		}
		return &Swarm{
			client: client,
		}
	}
	if provider == llm.DeepSeek {
		client := llm.NewDeepSeekLLM(apiKey)
		return &Swarm{
			client: client,
		}
	}
	return nil
}

func NewSwarmWithHost(apiKey, host string, provider llm.LLMProvider) *Swarm {
	if provider == llm.OpenAI {
		client := llm.NewOpenAILLMWithHost(apiKey, host)
		return &Swarm{
			client: client,
		}
	}
	return nil
}

// getChatCompletion requests a chat completion from the LLM
func (s *Swarm) getChatCompletion(
	ctx context.Context,
	agent *Agent,
	history []llm.Message,
	contextVariables map[string]interface{},
	modelOverride string,
	stream bool,
	debug bool,
) (llm.ChatCompletionResponse, error) {
	// Prepare the initial system message with agent instructions
	instructions := agent.Instructions
	if agent.InstructionsFunc != nil {
		instructions = agent.InstructionsFunc(contextVariables)
	}
	messages := append([]llm.Message{
		{
			Role:    llm.RoleSystem,
			Content: instructions,
		},
	}, history...)

	// Build tool definitions from agent's functions
	var tools []llm.Tool
	for _, af := range agent.Functions {
		def := FunctionToDefinition(af)
		tools = append(tools, llm.Tool{
			Type: "function",
			Function: &llm.Function{
				Name:        def.Name,
				Description: def.Description,
				Parameters:  def.Parameters,
			},
		})
	}

	// Prepare the chat completion request
	model := agent.Model
	if modelOverride != "" {
		model = modelOverride
	}

	req := llm.ChatCompletionRequest{
		Model:    model,
		Messages: messages,
		Tools:    tools,
	}

	if debug {
		log.Printf("Getting chat completion for: %+v\n", messages)
	}

	// Call the LLM to get a chat completion
	resp, err := s.client.CreateChatCompletion(ctx, req)
	if err != nil {
		return llm.ChatCompletionResponse{}, err
	}

	return resp, nil
}

// handleToolCall processes a tool call from the chat completion
func (s *Swarm) handleToolCall(
	ctx context.Context,
	toolCall *llm.ToolCall,
	agent *Agent,
	contextVariables map[string]interface{},
	debug bool,
) (Response, error) {
	toolName := toolCall.Function.Name
	argsJSON := toolCall.Function.Arguments

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
			Messages: []llm.Message{
				{
					Role:    llm.RoleAssistant,
					Content: errorMessage,
				},
			},
		}, nil
	}

	// Execute the function
	result := functionFound.Function(args, contextVariables)

	// Create a message with the tool result
	toolResultMessage := llm.Message{
		Role:    llm.RoleAssistant,
		Content: fmt.Sprintf("%v", result.Data),
	}

	// Return the partial response with the tool result and any agent transfer
	partialResponse := Response{
		Messages:         []llm.Message{toolResultMessage},
		Agent:            result.Agent, // Use the agent from the result if provided
		ContextVariables: contextVariables,
	}

	return partialResponse, nil
}

// Run executes the chat interaction loop with the agent
func (s *Swarm) Run(
	ctx context.Context,
	agent *Agent,
	messages []llm.Message,
	contextVariables map[string]interface{},
	modelOverride string,
	stream bool,
	debug bool,
	maxTurns int,
	executeTools bool,
) (Response, error) {
	// Use a cloned copy of messages for history
	history := cloneMessages(messages)

	if contextVariables == nil {
		contextVariables = make(map[string]interface{})
	}

	if agent.Memory == nil {
		agent.Memory = NewMemoryStore(100)
	}

	if last := lastMessage(messages); last != nil && last.Role == llm.RoleUser {
		agent.Memory.AddMemory(Memory{
			Content:   last.Content,
			Timestamp: time.Now(),
		})
	}

	// Get chat completion from LLM
	resp, err := s.getChatCompletion(ctx, agent, history, contextVariables, modelOverride, stream, debug)
	if err != nil {
		return Response{}, err
	}

	if len(resp.Choices) == 0 {
		return Response{}, fmt.Errorf("no choices in response")
	}

	choice := resp.Choices[0]
	history = append(history, choice.Message)

	if len(choice.Message.ToolCalls) > 0 && executeTools {
		// Handle tool calls in a separate helper function
		toolResults, updatedHistory, updatedAgent, err := s.handleToolCalls(ctx, choice.Message.ToolCalls, history, agent, contextVariables, modelOverride, stream, debug)
		if err != nil {
			return Response{}, err
		}
		history = updatedHistory
		agent = updatedAgent

		// Get follow-up response from LLM (avoid recursive tool calls)
		followUpResp, err := s.getChatCompletion(ctx, agent, history, contextVariables, modelOverride, stream, debug)
		if err != nil {
			return Response{}, err
		}
		followUpChoice := followUpResp.Choices[0]
		// Remove any tool calls to avoid loops
		followUpChoice.Message.ToolCalls = nil
		// Always append the follow-up message, even if empty
		history = append(history, followUpChoice.Message)

		return Response{
			Messages:         history[len(messages):],
			Agent:            agent,
			ContextVariables: contextVariables,
			ToolResults:      toolResults,
		}, nil
	}

	// No tool calls executed
	return Response{
		Messages:         history[len(messages):],
		Agent:            agent,
		ContextVariables: contextVariables,
		ToolResults:      nil,
	}, nil
}

// Helper function to clone a slice of messages
func cloneMessages(msgs []llm.Message) []llm.Message {
	cloned := make([]llm.Message, len(msgs))
	copy(cloned, msgs)
	return cloned
}

// Helper function to return the last message in a slice
func lastMessage(msgs []llm.Message) *llm.Message {
	if len(msgs) == 0 {
		return nil
	}
	return &msgs[len(msgs)-1]
}

// Helper method to handle tool calls within the chat loop
func (s *Swarm) handleToolCalls(
	ctx context.Context,
	toolCalls []llm.ToolCall,
	history []llm.Message,
	agent *Agent,
	contextVariables map[string]interface{},
	modelOverride string,
	stream bool,
	debug bool,
) ([]ToolResult, []llm.Message, *Agent, error) {
	var toolResults []ToolResult
	for _, toolCall := range toolCalls {
		toolResp, err := s.handleToolCall(ctx, &toolCall, agent, contextVariables, debug)
		if err != nil {
			return nil, history, agent, err
		}

		var args interface{}
		_ = json.Unmarshal([]byte(toolCall.Function.Arguments), &args)
		toolResults = append(toolResults, ToolResult{
			ToolName: toolCall.Function.Name,
			Args:     args,
			Result: Result{
				Success: true,
				Data:    toolResp.Messages[0].Content,
				Error:   nil,
				Agent:   toolResp.Agent,
			},
		})

		history = append(history, llm.Message{
			Role:    llm.RoleFunction,
			Content: toolResp.Messages[0].Content,
			Name:    toolCall.Function.Name,
		})
		if toolResp.Agent != nil {
			agent = toolResp.Agent
		}
	}
	return toolResults, history, agent, nil
}
