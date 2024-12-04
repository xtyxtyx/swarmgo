package llm

import (
	"context"

	"github.com/sashabaranov/go-openai"
)

// OpenAILLM implements the LLM interface for OpenAI
type OpenAILLM struct {
	client *openai.Client
}

// NewOpenAILLM creates a new OpenAI LLM client
func NewOpenAILLM(apiKey string) *OpenAILLM {
	client := openai.NewClient(apiKey)
	return &OpenAILLM{client: client}
}


// convertToOpenAIMessages converts our generic Message type to OpenAI's message type
func convertToOpenAIMessages(messages []Message) []openai.ChatCompletionMessage {
	openAIMessages := make([]openai.ChatCompletionMessage, len(messages))
	for i, msg := range messages {
		openAIMessages[i] = openai.ChatCompletionMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
			Name:    msg.Name,
		}
	}
	return openAIMessages
}

// convertFromOpenAIMessage converts OpenAI's message type to our generic Message type
func convertFromOpenAIMessage(msg openai.ChatCompletionMessage) Message {
	return Message{
		Role:    Role(msg.Role),
		Content: msg.Content,
		Name:    msg.Name,
	}
}

// convertFromOpenAIDelta converts OpenAI's delta message type to our generic Message type
func convertFromOpenAIDelta(delta openai.ChatCompletionStreamChoiceDelta) Message {
	return Message{
		Role:    Role(delta.Role),
		Content: delta.Content,
	}
}

// convertToOpenAITools converts our generic Tool type to OpenAI's tool type
func convertToOpenAITools(tools []Tool) []openai.Tool {
	if len(tools) == 0 {
		return nil
	}
	
	openAITools := make([]openai.Tool, len(tools))
	for i, tool := range tools {
		def := openai.FunctionDefinition{
			Name:        tool.Function.Name,
			Description: tool.Function.Description,
			Parameters:  tool.Function.Parameters,
		}
		openAITools[i] = openai.Tool{
			Type:     openai.ToolTypeFunction,
			Function: &def,
		}
	}
	return openAITools
}

// convertFromOpenAIToolCalls converts OpenAI's tool calls to our generic type
func convertFromOpenAIToolCalls(toolCalls []openai.ToolCall) []ToolCall {
	if len(toolCalls) == 0 {
		return nil
	}

	calls := make([]ToolCall, len(toolCalls))
	for i, call := range toolCalls {
		calls[i] = ToolCall{
			ID:   call.ID,
			Type: string(call.Type),
		}
		calls[i].Function.Name = call.Function.Name
		calls[i].Function.Arguments = call.Function.Arguments
	}
	return calls
}

// CreateChatCompletion implements the LLM interface for OpenAI
func (o *OpenAILLM) CreateChatCompletion(ctx context.Context, req ChatCompletionRequest) (ChatCompletionResponse, error) {
	openAIReq := openai.ChatCompletionRequest{
		Model:            req.Model,
		Messages:         convertToOpenAIMessages(req.Messages),
		Temperature:      float32(req.Temperature),
		TopP:            float32(req.TopP),
		N:               req.N,
		Stop:            req.Stop,
		MaxTokens:       req.MaxTokens,
		PresencePenalty: req.PresencePenalty,
		Tools:           convertToOpenAITools(req.Tools),
	}

	resp, err := o.client.CreateChatCompletion(ctx, openAIReq)
	if err != nil {
		return ChatCompletionResponse{}, err
	}

	choices := make([]Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		msg := convertFromOpenAIMessage(c.Message)
		msg.ToolCalls = convertFromOpenAIToolCalls(c.Message.ToolCalls)
		choices[i] = Choice{
			Index:        c.Index,
			Message:      msg,
			FinishReason: string(c.FinishReason),
		}
	}

	return ChatCompletionResponse{
		ID:      resp.ID,
		Choices: choices,
		Usage: Usage{
			PromptTokens:     resp.Usage.PromptTokens,
			CompletionTokens: resp.Usage.CompletionTokens,
			TotalTokens:      resp.Usage.TotalTokens,
		},
	}, nil
}

// openAIStreamWrapper wraps OpenAI's stream to implement our ChatCompletionStream interface
type openAIStreamWrapper struct {
	stream *openai.ChatCompletionStream
}

func (w *openAIStreamWrapper) Recv() (ChatCompletionResponse, error) {
	resp, err := w.stream.Recv()
	if err != nil {
		return ChatCompletionResponse{}, err
	}

	choices := make([]Choice, len(resp.Choices))
	for i, c := range resp.Choices {
		message := convertFromOpenAIDelta(c.Delta)
		message.Role = Role(c.Delta.Role)
		
		// Handle tool calls in delta
		if len(c.Delta.ToolCalls) > 0 {
			message.ToolCalls = make([]ToolCall, len(c.Delta.ToolCalls))
			for j, tc := range c.Delta.ToolCalls {
				toolCall := ToolCall{
					ID:   tc.ID,
					Type: string(tc.Type),
					Function: ToolCallFunction{
						Name:      tc.Function.Name,
						Arguments: tc.Function.Arguments,
					},
				}
				message.ToolCalls[j] = toolCall
			}
		}
		
		choices[i] = Choice{
			Index:        c.Index,
			Message:      message,
			FinishReason: string(c.FinishReason),
		}
	}

	return ChatCompletionResponse{
		ID:      resp.ID,
		Choices: choices,
	}, nil
}

func (w *openAIStreamWrapper) Close() error {
	w.stream.Close()
	return nil
}

// CreateChatCompletionStream implements the LLM interface for OpenAI streaming
func (o *OpenAILLM) CreateChatCompletionStream(ctx context.Context, req ChatCompletionRequest) (ChatCompletionStream, error) {
	openAIReq := openai.ChatCompletionRequest{
		Model:            req.Model,
		Messages:         convertToOpenAIMessages(req.Messages),
		Temperature:      req.Temperature,
		TopP:            req.TopP,
		N:               req.N,
		Stop:            req.Stop,
		MaxTokens:       req.MaxTokens,
		PresencePenalty: req.PresencePenalty,
		User:            req.User,
		Tools:           convertToOpenAITools(req.Tools),
		Stream:          true,
	}

	stream, err := o.client.CreateChatCompletionStream(ctx, openAIReq)
	if err != nil {
		return nil, err
	}

	return &openAIStreamWrapper{stream: stream}, nil
}
