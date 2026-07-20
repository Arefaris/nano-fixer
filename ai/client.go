package ai

import (
	"context"
	"errors"
	"strings"

	openai "github.com/sashabaranov/go-openai"
)

type Client struct {
	client    *openai.Client
	modelName string
}

func NewClient(apiKey, apiBaseURL, modelName string) *Client {
	config := openai.DefaultConfig(apiKey)
	if apiBaseURL != "" {
		config.BaseURL = apiBaseURL
	}
	
	return &Client{
		client:    openai.NewClientWithConfig(config),
		modelName: modelName,
	}
}

func (c *Client) CorrectText(ctx context.Context, text string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", errors.New("empty text provided")
	}

	systemPrompt := `You are a background grammar correction assistant. 
Your task is to fix grammar, spelling, and missing articles in the provided text.
CRITICAL RULES:
1. Do NOT change the meaning of the text.
2. Do NOT change the style, tone, or register. If the text is casual (e.g., Discord slang, gaming terms, fast typing), keep it casual and natural.
3. Return ONLY the corrected text. Do not add explanations, quotes, or markdown formatting.`

	resp, err := c.client.CreateChatCompletion(
		ctx,
		openai.ChatCompletionRequest{
			Model: c.modelName,
			Messages: []openai.ChatCompletionMessage{
				{
					Role:    openai.ChatMessageRoleSystem,
					Content: systemPrompt,
				},
				{
					Role:    openai.ChatMessageRoleUser,
					Content: text,
				},
			},
			Temperature: 0.3, // Low temperature for consistent corrections
		},
	)

	if err != nil {
		return "", err
	}

	if len(resp.Choices) > 0 {
		return strings.TrimSpace(resp.Choices[0].Message.Content), nil
	}

	return "", errors.New("no completion choices returned")
}
