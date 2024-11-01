package swarmgo

import (
	"bytes"
	"context"
	"errors"
	"io"
	"log"
	"os"
	"testing"

	openai "github.com/sashabaranov/go-openai"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockOpenAIClient is a mock of the OpenAI client
type MockOpenAIClient struct {
	mock.Mock
}

func (m *MockOpenAIClient) CreateChatCompletion(ctx context.Context, req openai.ChatCompletionRequest) (openai.ChatCompletionResponse, error) {
	args := m.Called(ctx, req)
	return args.Get(0).(openai.ChatCompletionResponse), args.Error(1)
}

// NewMockSwarm initializes a new Swarm instance with a mock OpenAI client
func NewMockSwarm(mockClient *MockOpenAIClient) *Swarm {
	return &Swarm{
		client: mockClient,
	}
}

// TestNewSwarm tests the NewSwarm function
func TestNewSwarm(t *testing.T) {
	apiKey := "test-api-key"
	sw := NewSwarm(apiKey)
	assert.NotNil(t, sw)
	assert.NotNil(t, sw.client)
}

// TestFunctionToDefinition tests the FunctionToDefinition function
func TestFunctionToDefinition(t *testing.T) {
	af := AgentFunction{
		Name:        "testFunction",
		Description: "A test function",
		Parameters: map[string]interface{}{
			"type": "object",
			"properties": map[string]interface{}{
				"arg1": map[string]interface{}{
					"type":        "string",
					"description": "Argument 1",
				},
			},
			"required": []interface{}{"arg1"},
		},
	}

	def := FunctionToDefinition(af)

	assert.Equal(t, af.Name, def.Name)
	assert.Equal(t, af.Description, def.Description)
	assert.Equal(t, af.Parameters, def.Parameters)
}

// TestHandleFunctionCall tests the handleFunctionCall method
func TestHandleFunctionCall(t *testing.T) {
	sw := NewSwarm("test-api-key")
	ctx := context.Background()

	toolCall := openai.ToolCall{
		ID:      "testFunction",
		Function: openai.FunctionCall{
			Name:      "testFunction",
			Arguments: `{"arg1": "value1"}`,
		},
	}

	agentFunction := AgentFunction{
		Name:        "testFunction",
		Description: "A test function",
		Function: func(args map[string]interface{}, contextVariables map[string]interface{}) Result {
			return Result{
				Value: "Function executed successfully",
			}
		},
	}

	agent := &Agent{
		Name:      "TestAgent",
		Functions: []AgentFunction{agentFunction},
	}

	contextVariables := map[string]interface{}{}

	response, err := sw.handleToolCall(ctx, &toolCall, agent, contextVariables, false)

	assert.NoError(t, err)
	assert.Len(t, response.Messages, 1)
	assert.Equal(t, "tool", response.Messages[0].Role)
	assert.Equal(t, "testFunction", response.Messages[0].Name)
	assert.Equal(t, "Function executed successfully", response.Messages[0].Content)
}

// TestHandleFunctionCallFunctionNotFound tests handleFunctionCall when function is not found
func TestHandleFunctionCallFunctionNotFound(t *testing.T) {
	sw := NewSwarm("test-api-key")
	ctx := context.Background()

	toolCall := openai.ToolCall{
		ID:      "nonExistentFunction",
		Function: openai.FunctionCall{
			Name:      "nonExistentFunction",
			Arguments: `{}`,
		},
	}

	agent := &Agent{
		Name:      "TestAgent",
		Functions: []AgentFunction{},
	}

	contextVariables := map[string]interface{}{}

	response, err := sw.handleToolCall(ctx, &toolCall, agent, contextVariables, false)

	assert.NoError(t, err)
	assert.Len(t, response.Messages, 1)
	assert.Equal(t, "tool", response.Messages[0].Role)
	assert.Equal(t, "nonExistentFunction", response.Messages[0].Name)
	assert.Contains(t, response.Messages[0].Content, "Error: Tool nonExistentFunction not found.")
}

// TestRun tests the Run method
func TestRun(t *testing.T) {
	mockClient := new(MockOpenAIClient)
	sw := NewMockSwarm(mockClient)
	ctx := context.Background()

	agentFunction := AgentFunction{
		Name:        "testFunction",
		Description: "A test function",
		Function: func(args map[string]interface{}, contextVariables map[string]interface{}) Result {
			return Result{
				Value: "Function executed successfully",
			}
		},
	}

	agent := &Agent{
		Name:      "TestAgent",
		Functions: []AgentFunction{agentFunction},
		Model:     "test-model",
	}

	messages := []openai.ChatCompletionMessage{
		{Role: "user", Content: "Hello"},
	}

	// Mock the OpenAI API response
	mockResponse1 := openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "",
					FunctionCall: &openai.FunctionCall{
						Name:      "testFunction",
						Arguments: `{"arg1": "value1"}`,
					},
				},
			},
		},
	}

	mockResponse2 := openai.ChatCompletionResponse{
		Choices: []openai.ChatCompletionChoice{
			{
				Message: openai.ChatCompletionMessage{
					Role:    openai.ChatMessageRoleAssistant,
					Content: "Here is the result of the function.",
				},
			},
		},
	}

	mockClient.On("CreateChatCompletion", mock.Anything, mock.Anything).Return(mockResponse1, nil).Once()
	mockClient.On("CreateChatCompletion", mock.Anything, mock.Anything).Return(mockResponse2, nil).Once()

	response, err := sw.Run(ctx, agent, messages, nil, "", false, false, 5, true)

	assert.NoError(t, err)
	assert.Len(t, response.Messages, 3)
	assert.Equal(t, "TestAgent", response.Agent.Name)
	assert.Equal(t, "Here is the result of the function.", response.Messages[2].Content)
}

// TestRunFunctionCallError tests the Run method when function call returns an error
func TestRunFunctionCallError(t *testing.T) {
	mockClient := new(MockOpenAIClient)
	sw := NewMockSwarm(mockClient)
	ctx := context.Background()

	agentFunction := AgentFunction{
		Name:        "testFunction",
		Description: "A test function",
		Function: func(args map[string]interface{}, contextVariables map[string]interface{}) Result {
			return Result{
				Value: "Function executed successfully",
			}
		},
	}

	agent := &Agent{
		Name:      "TestAgent",
		Functions: []AgentFunction{agentFunction},
		Model:     "test-model",
	}

	messages := []openai.ChatCompletionMessage{
		{Role: "user", Content: "Hello"},
	}

	// Mock the OpenAI API to return an error
	mockClient.On("CreateChatCompletion", mock.Anything, mock.Anything).Return(openai.ChatCompletionResponse{}, errors.New("API error"))

	response, err := sw.Run(ctx, agent, messages, nil, "", false, false, 5, true)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "API error")
	assert.Len(t, response.Messages, 0)
}

// TestProcessAndPrintResponse tests the ProcessAndPrintResponse function
func TestProcessAndPrintResponse(t *testing.T) {
	response := Response{
		Messages: []openai.ChatCompletionMessage{
			{
				Role:    "assistant",
				Name:    "TestAgent",
				Content: "Hello, how can I assist you?",
			},
			{
				Role:    "function",
				Name:    "testFunction",
				Content: "Function output",
			},
		},
	}

	// Capture the output
	var buf bytes.Buffer
	writer := io.MultiWriter(os.Stdout, &buf)
	log.SetOutput(writer)

	ProcessAndPrintResponse(response)

	output := buf.String()
	assert.NotNil(t, output)
}
