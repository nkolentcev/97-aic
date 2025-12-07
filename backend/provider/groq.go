package provider

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// GroqProvider провайдер для Groq API (OpenAI-compatible)
type GroqProvider struct {
	httpClient *http.Client
	apiKey     string
	apiURL     string
	model      string
}

// GroqConfig конфигурация Groq провайдера
type GroqConfig struct {
	APIKey string
	APIURL string // по умолчанию https://api.groq.com/openai/v1
	Model  string // по умолчанию llama-3.3-70b-versatile
}

// Доступные модели Groq (бесплатные)
var GroqModels = []string{
	"llama-3.3-70b-versatile",    // Лучшая для большинства задач
	"llama-3.1-8b-instant",       // Быстрая, легкая
	"llama-3.2-3b-preview",       // Очень быстрая, минимальная
	"mixtral-8x7b-32768",         // Хороша для длинного контекста
	"gemma2-9b-it",               // Google Gemma 2
}

// NewGroqProvider создает новый Groq провайдер
func NewGroqProvider(cfg GroqConfig) *GroqProvider {
	apiURL := cfg.APIURL
	if apiURL == "" {
		apiURL = "https://api.groq.com/openai/v1"
	}

	model := cfg.Model
	if model == "" {
		model = "llama-3.3-70b-versatile"
	}

	return &GroqProvider{
		httpClient: &http.Client{
			Timeout: 120 * time.Second,
		},
		apiKey: cfg.APIKey,
		apiURL: apiURL,
		model:  model,
	}
}

// Name возвращает имя провайдера
func (p *GroqProvider) Name() string {
	return "groq"
}

// Models возвращает список доступных моделей
func (p *GroqProvider) Models() []string {
	return GroqModels
}

// SetModel устанавливает модель
func (p *GroqProvider) SetModel(model string) {
	p.model = model
}

// GetModel возвращает текущую модель
func (p *GroqProvider) GetModel() string {
	return p.model
}

// groqChatRequest запрос к Groq API
type groqChatRequest struct {
	Model       string         `json:"model"`
	Messages    []groqMessage  `json:"messages"`
	Stream      bool           `json:"stream"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Temperature float64        `json:"temperature,omitempty"`
}

type groqMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type groqChatResponse struct {
	ID      string       `json:"id"`
	Choices []groqChoice `json:"choices"`
}

type groqChoice struct {
	Index        int         `json:"index"`
	Delta        *groqDelta  `json:"delta,omitempty"`
	Message      *groqDelta  `json:"message,omitempty"`
	FinishReason string      `json:"finish_reason"`
}

type groqDelta struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Chat отправляет сообщение через Groq API
func (p *GroqProvider) Chat(ctx context.Context, message string, opts *ChatOptions, onChunk func(string) error) error {
	messages := []groqMessage{}

	// System prompt
	var systemPrompt string
	if opts != nil {
		if opts.JSONFormat && opts.JSONSchemaText != "" {
			systemPrompt = BuildJSONPrompt(opts.JSONSchemaText)
		} else if opts.ReasoningMode != "" {
			systemPrompt = BuildReasoningPrompt(opts.ReasoningMode, opts.SystemPrompt)
		} else if opts.SystemPrompt != "" {
			systemPrompt = opts.SystemPrompt
		}
	}

	if systemPrompt != "" {
		messages = append(messages, groqMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// История
	if opts != nil && len(opts.History) > 0 {
		for _, msg := range opts.History {
			messages = append(messages, groqMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	// Текущее сообщение
	messages = append(messages, groqMessage{
		Role:    "user",
		Content: message,
	})

	reqBody := groqChatRequest{
		Model:    p.model,
		Messages: messages,
		Stream:   true,
	}

	if opts != nil {
		if opts.MaxTokens > 0 {
			reqBody.MaxTokens = opts.MaxTokens
		}
		if opts.Temperature > 0 {
			reqBody.Temperature = opts.Temperature
		}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга запроса: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ошибка Groq API: %d - %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" || !strings.HasPrefix(line, "data: ") {
			continue
		}

		data := strings.TrimPrefix(line, "data: ")
		if data == "[DONE]" {
			break
		}

		var chatResp groqChatResponse
		if err := json.Unmarshal([]byte(data), &chatResp); err != nil {
			continue
		}

		if len(chatResp.Choices) > 0 {
			choice := chatResp.Choices[0]
			var content string
			if choice.Delta != nil {
				content = choice.Delta.Content
			} else if choice.Message != nil {
				content = choice.Message.Content
			}

			if content != "" {
				if err := onChunk(content); err != nil {
					return err
				}
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения потока: %w", err)
	}

	return nil
}
