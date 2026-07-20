package ai

import (
	"context"
	"errors"
	"fmt"
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

// UpdateConfig updates the client settings dynamically when changed in UI
func (c *Client) UpdateConfig(apiKey, apiBaseURL, modelName string) {
	config := openai.DefaultConfig(apiKey)
	if apiBaseURL != "" {
		config.BaseURL = apiBaseURL
	}
	c.client = openai.NewClientWithConfig(config)
	c.modelName = modelName
}

func (c *Client) CorrectText(ctx context.Context, text string, targetLanguage string) (string, error) {
	if strings.TrimSpace(text) == "" {
		return "", errors.New("empty text provided")
	}

	var systemPrompt string
	if targetLanguage == "" || strings.ToLower(targetLanguage) == "auto" {
		systemPrompt = `You are a background grammar and style correction assistant.
Your task is to fix grammar, spelling, typos, and missing articles in the provided text.
CRITICAL RULES:
1. Detect the language of the input text automatically. Correct it in that SAME language. Do NOT translate it.
2. Do NOT change the meaning of the text.
3. Do NOT change the style, tone, or register. If the text is casual (e.g. Discord slang, gaming terms, shorthand, fast typing), keep it casual and natural.
4. Return ONLY the corrected text. Do not add explanations, quotes, or markdown formatting.`
	} else {
		systemPrompt = fmt.Sprintf(`You are a background grammar and style correction assistant.
Your task is to correct the text and ensure it is in the target language: %s.
CRITICAL RULES:
1. Translate the text to %s if it is in another language.
2. Fix all grammar, spelling, typos, and style errors.
3. Do NOT change the core meaning of the text.
4. Try to preserve the original style, tone, and register (e.g. if the input is casual slang, translate it to equivalent casual slang in %s).
5. Return ONLY the corrected text. Do not add explanations, quotes, or markdown formatting.`, targetLanguage, targetLanguage, targetLanguage)
	}

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
			Temperature: 0.3,
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
