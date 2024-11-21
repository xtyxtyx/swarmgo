package swarmgo

import (
	"context"
	"encoding/json"
	"fmt"

	openai "github.com/sashabaranov/go-openai"
)

// StreamHandler represents a handler for streaming responses
type StreamHandler interface {
	OnStart()
	OnToken(token string)
	OnToolCall(toolCall openai.ToolCall)
	OnComplete(message openai.ChatCompletionMessage)
	OnError(err error)
}

// DefaultStreamHandler provides a basic implementation of StreamHandler
type DefaultStreamHandler struct{}

func (h *DefaultStreamHandler) OnStart()                                        {}
func (h *DefaultStreamHandler) OnToken(token string)                            {}
func (h *DefaultStreamHandler) OnToolCall(toolCall openai.ToolCall)             {}
func (h *DefaultStreamHandler) OnComplete(message openai.ChatCompletionMessage) {}
func (h *DefaultStreamHandler) OnError(err error)                               {}

// StreamingResponse handles streaming chat completions
func (s *Swarm) StreamingResponse(
	ctx context.Context,
	agent *Agent,
	messages []openai.ChatCompletionMessage,
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
	allMessages := append([]openai.ChatCompletionMessage{
		{
			Role:    openai.ChatMessageRoleSystem,
			Content: instructions,
		},
	}, messages...)

	// Build tool definitions
	var tools []openai.Tool
	for _, af := range agent.Functions {
		def := FunctionToDefinition(af)
		tools = append(tools, openai.Tool{
			Type:     "function",
			Function: &def,
		})
	}

	// Prepare the streaming request
	model := agent.Model
	if modelOverride != "" {
		model = modelOverride
	}

	req := openai.ChatCompletionRequest{
		Model:    model,
		Messages: allMessages,
		Tools:    tools,
		Stream:   true,
	}

	stream, err := s.client.(interface {
		CreateChatCompletionStream(context.Context, openai.ChatCompletionRequest) (*openai.ChatCompletionStream, error)
	}).CreateChatCompletionStream(ctx, req)

	if err != nil {
		handler.OnError(fmt.Errorf("failed to create chat completion stream: %v", err))
		return err
	}
	defer stream.Close()

	handler.OnStart()

	var currentMessage openai.ChatCompletionMessage
	currentMessage.Role = openai.ChatMessageRoleAssistant
	currentMessage.Name = agent.Name

	for {
		response, err := stream.Recv()
		if err != nil {
			if err.Error() == "EOF" {
				// Handle any pending tool calls before completing
				if len(currentMessage.ToolCalls) > 0 {
					var toolResults []openai.ChatCompletionMessage
					for _, toolCall := range currentMessage.ToolCalls {
						// Find the corresponding function
						var fn *AgentFunction
						for _, f := range agent.Functions {
							if f.Name == toolCall.Function.Name {
								fn = &f
								break
							}
						}

						if fn == nil {
							handler.OnError(fmt.Errorf("unknown function: %s", toolCall.Function.Name))
							continue
						}

						// Parse arguments
						var args map[string]interface{}
						if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
							handler.OnError(fmt.Errorf("failed to parse arguments: %v", err))
							continue
						}

						// Execute the function
						result := fn.Function(args, contextVariables)

						// Add function call and result to messages
						toolResults = append(toolResults, openai.ChatCompletionMessage{
							Role:       openai.ChatMessageRoleTool,
							Content:    fmt.Sprintf("%v", result.Value),
							ToolCallID: toolCall.ID,
							Name:       agent.Name,
						})

						// Update context variables
						for k, v := range result.ContextVariables {
							contextVariables[k] = v
						}
					}

					// Add tool results to messages and get final response
					allMessages = append(allMessages, currentMessage)
					allMessages = append(allMessages, toolResults...)

					// Make one final non-streaming call to get the conclusion
					finalReq := openai.ChatCompletionRequest{
						Model:    model,
						Messages: allMessages,
						Stream:   false,
					}

					finalResp, err := s.client.CreateChatCompletion(ctx, finalReq)
					if err != nil {
						handler.OnError(fmt.Errorf("failed to get final response: %v", err))
						return err
					}

					if len(finalResp.Choices) > 0 {
						finalMessage := finalResp.Choices[0].Message
						handler.OnToken(finalMessage.Content)
						currentMessage.Content = finalMessage.Content
					}
				}
				handler.OnComplete(currentMessage)
				return nil
			}
			handler.OnError(fmt.Errorf("error receiving from stream: %v", err))
			return err
		}

		if len(response.Choices) > 0 {
			delta := response.Choices[0].Delta

			// Handle content streaming
			if delta.Content != "" {
				currentMessage.Content += delta.Content
				handler.OnToken(delta.Content)
			}

			// Handle tool calls
			if delta.ToolCalls != nil {
				for _, toolCall := range delta.ToolCalls {
					if len(currentMessage.ToolCalls) == 0 ||
						len(currentMessage.ToolCalls) <= *toolCall.Index {
						currentMessage.ToolCalls = append(currentMessage.ToolCalls, openai.ToolCall{
							ID:       toolCall.ID,
							Type:     toolCall.Type,
							Function: openai.FunctionCall{},
						})
					}

					if (toolCall.Function != openai.FunctionCall{}) {
						if toolCall.Function.Name != "" {
							currentMessage.ToolCalls[*toolCall.Index].Function.Name = toolCall.Function.Name
						}
						if toolCall.Function.Arguments != "" {
							currentMessage.ToolCalls[*toolCall.Index].Function.Arguments += toolCall.Function.Arguments
						}
					}

					handler.OnToolCall(currentMessage.ToolCalls[*toolCall.Index])
				}
			}
		}
	}
}
