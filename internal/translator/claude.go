package translator

import (
	"context"
	"fmt"
	"unicode"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Message represents a conversation message
type Message struct {
	Role    string
	Content string
}

// Translator handles translation using Claude API
type Translator struct {
	client  anthropic.Client
	model   string
	history []Message
}

// New creates a new Translator
func New(apiKey, model string) (*Translator, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	client := anthropic.NewClient(option.WithAPIKey(apiKey))

	return &Translator{
		client:  client,
		model:   model,
		history: make([]Message, 0),
	}, nil
}

// Translate translates the input text, maintaining conversation history
func (t *Translator) Translate(ctx context.Context, input string) (string, error) {
	// Add user message to history
	t.history = append(t.history, Message{Role: "user", Content: input})

	// Build messages for API call
	messages := make([]anthropic.MessageParam, 0, len(t.history))
	for _, msg := range t.history {
		if msg.Role == "user" {
			messages = append(messages, anthropic.NewUserMessage(anthropic.NewTextBlock(msg.Content)))
		} else {
			messages = append(messages, anthropic.NewAssistantMessage(anthropic.NewTextBlock(msg.Content)))
		}
	}

	// Detect language direction
	systemPrompt := buildSystemPrompt(input)

	// Call Claude API
	resp, err := t.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.Model(t.model),
		MaxTokens: int64(1024),
		System: []anthropic.TextBlockParam{
			{Text: systemPrompt},
		},
		Messages: messages,
	})
	if err != nil {
		// Remove the failed message from history
		t.history = t.history[:len(t.history)-1]
		return "", fmt.Errorf("API error: %w", err)
	}

	// Extract response text
	var result string
	for _, block := range resp.Content {
		if block.Type == "text" {
			result += block.Text
		}
	}

	// Add assistant response to history
	t.history = append(t.history, Message{Role: "assistant", Content: result})

	return result, nil
}

// Reset clears conversation history
func (t *Translator) Reset() {
	t.history = make([]Message, 0)
}

// HistoryLength returns the number of messages in history
func (t *Translator) HistoryLength() int {
	return len(t.history)
}

func buildSystemPrompt(input string) string {
	// Detect if input is primarily Chinese
	isChinese := containsChinese(input)

	if isChinese {
		return `You are a translator. The FIRST message is the text to translate from Chinese to English.
ALL subsequent messages are modification requests to refine that translation (e.g., "more casual", "shorter", "formal").

Translation style:
- Use casual, conversational tone (like texting or instant messaging)
- Keep vocabulary and grammar simple
- Avoid overly formal or complex expressions
- Unless the user explicitly asks for formal/academic style

Only output the translation, no explanations.`
	}

	return `You are a translator. The FIRST message is the text to translate from English to Chinese.
ALL subsequent messages are modification requests to refine that translation (e.g., "more casual", "shorter", "formal").

Translation style:
- Use casual, conversational tone (like texting or instant messaging)
- Keep vocabulary and grammar simple
- Avoid overly formal or complex expressions
- Unless the user explicitly asks for formal/academic style

Only output the translation, no explanations.`
}

func containsChinese(s string) bool {
	for _, r := range s {
		if unicode.Is(unicode.Han, r) {
			return true
		}
	}
	return false
}
