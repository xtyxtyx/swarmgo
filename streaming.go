package swarmgo

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/prathyushnallamothu/swarmgo/llm"
)

// StreamHandler represents a handler for streaming responses
type StreamHandler interface {
	OnStart()
	OnToken(token string)
	OnToolCall(toolCall llm.ToolCall)
	OnComplete(message llm.Message)
	OnError(err error)
}

// DefaultStreamHandler provides a basic implementation of StreamHandler
type DefaultStreamHandler struct{}

func (h *DefaultStreamHandler) OnStart()                              {}
func (h *DefaultStreamHandler) OnToken(token string)                  {}
func (h *DefaultStreamHandler) OnToolCall(toolCall llm.ToolCall)      {}
func (h *DefaultStreamHandler) OnComplete(message llm.Message)        {}
func (h *DefaultStreamHandler) OnError(err error)                     {}

// StreamingResponse handles streaming chat completions
func (s *Swarm) StreamingResponse(
	ctx context.Context,
	agent *Agent,
	messages []llm.Message,
	contextVariables map[string]interface{},
	modelOverride string,
	handler StreamHandler,
	debug bool,
) error {
	if handler == nil {
		handler = &DefaultStreamHandler{}
	}

	if contextVariables == nil {
		contextVariables = make(map[string]interface{})
	}

	// Prepare the initial system message with agent instructions
	instructions := agent.Instructions
	if agent.InstructionsFunc != nil {
		instructions = agent.InstructionsFunc(contextVariables)
	}
	allMessages := append([]llm.Message{
		{
			Role:    llm.RoleSystem,
			Content: instructions,
		},
	}, messages...)

	// Build tool definitions
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

	// Prepare the streaming request
	model := agent.Model
	if modelOverride != "" {
		model = modelOverride
	}

	req := llm.ChatCompletionRequest{
		Model:    model,
		Messages: allMessages,
		Tools:    tools,
		Stream:   true,
	}

	stream, err := s.client.CreateChatCompletionStream(ctx, req)
	if err != nil {
		handler.OnError(fmt.Errorf("failed to create chat completion stream: %v", err))
		return err
	}
	defer stream.Close()

	handler.OnStart()

	var currentMessage llm.Message
	currentMessage.Role = llm.RoleAssistant
	currentMessage.Name = agent.Name

	// Track tool calls being built
	toolCallsInProgress := make(map[string]*llm.ToolCall)
	processedToolCalls := make(map[string]bool)

	for {
		response, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				handler.OnComplete(currentMessage)
				return nil
			}
			handler.OnError(fmt.Errorf("error receiving from stream: %v", err))
			return err
		}

		if len(response.Choices) == 0 {
			continue
		}

		choice := response.Choices[0]

		// Handle content streaming
		if choice.Message.Content != "" {
			currentMessage.Content += choice.Message.Content
			handler.OnToken(choice.Message.Content)
		}

		// Handle tool calls
		if len(choice.Message.ToolCalls) > 0 {
			for _, toolCall := range choice.Message.ToolCalls {
				// Skip if we've already processed this tool call
				if processedToolCalls[toolCall.ID] {
					continue
				}

				// Get or create the in-progress tool call
				inProgress, exists := toolCallsInProgress[toolCall.ID]
				if !exists {
					inProgress = &llm.ToolCall{
						ID:   toolCall.ID,
						Type: toolCall.Type,
						Function: llm.ToolCallFunction{
							Name:      toolCall.Function.Name,
							Arguments: "",
						},
					}
					toolCallsInProgress[toolCall.ID] = inProgress
				}

				// Accumulate function arguments
				if toolCall.Function.Arguments != "" {
					inProgress.Function.Arguments += toolCall.Function.Arguments
				}

				// If we have a complete tool call (has both name and arguments)
				if inProgress.Function.Name != "" && inProgress.Function.Arguments != "" {
					// Try to parse the arguments to verify it's complete JSON
					var args map[string]interface{}
					if err := json.Unmarshal([]byte(inProgress.Function.Arguments), &args); err != nil {
						continue // Wait for more chunks
					}

					// Mark as processed
					processedToolCalls[toolCall.ID] = true

					// Add to current message
					currentMessage.ToolCalls = append(currentMessage.ToolCalls, *inProgress)
					handler.OnToolCall(*inProgress)

					// Find the corresponding function
					var fn *AgentFunction
					for _, f := range agent.Functions {
						if f.Name == inProgress.Function.Name {
							fn = &f
							break
						}
					}

					if fn == nil {
						handler.OnError(fmt.Errorf("unknown function: %s", inProgress.Function.Name))
						continue
					}

					// Execute the function
					result := fn.Function(args, contextVariables)

					// Create function response message
					var resultContent string
					if result.Error != nil {
						resultContent = fmt.Sprintf("Error: %v", result.Error)
					} else {
						resultContent = fmt.Sprintf("%v", result.Data)
					}

					// Add function response to messages
					functionMessage := llm.Message{
						Role:    llm.Role(inProgress.Function.Name),
						Content: resultContent,
						Name:    inProgress.Function.Name,
					}

					// Add the current message and function result to messages
					allMessages = append(allMessages, currentMessage)
					allMessages = append(allMessages, functionMessage)

					// Create a new request with updated messages
					req.Messages = allMessages

					// Close current stream and start a new one
					stream.Close()
					stream, err = s.client.CreateChatCompletionStream(ctx, req)
					if err != nil {
						handler.OnError(fmt.Errorf("failed to create new stream after tool call: %v", err))
						return err
					}

					// Reset current message for new response
					currentMessage = llm.Message{
						Role: llm.RoleAssistant,
						Name: agent.Name,
					}

					// Clean up the completed tool call
					delete(toolCallsInProgress, toolCall.ID)
				}
			}
		}
	}
}
