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

// OllamaProvider провайдер для локального Ollama
type OllamaProvider struct {
	httpClient *http.Client
	apiURL     string
	model      string
}

// OllamaConfig конфигурация Ollama провайдера
type OllamaConfig struct {
	APIURL string // по умолчанию http://localhost:11434
	Model  string // по умолчанию llama3.2:3b
}

// Рекомендуемые модели для разного железа
var OllamaModels = map[string][]string{
	// Для слабых машин (8GB RAM, без GPU)
	"low": {
		"qwen2.5:0.5b", // 0.5B - очень быстрая, минимальные требования
		"qwen2.5:1.5b", // 1.5B - хороший баланс
		"llama3.2:1b",  // 1B - компактная Llama
		"gemma2:2b",    // 2B - Google Gemma
	},
	// Для средних машин (16GB RAM, GTX 1060-1080)
	"medium": {
		"llama3.2:3b", // 3B - рекомендуемая по умолчанию
		"qwen2.5:3b",  // 3B - хорошее качество
		"phi3:mini",   // 3.8B - Microsoft Phi-3
		"mistral:7b",  // 7B - отличное качество
	},
	// Для мощных машин (32GB+ RAM, RTX 3070+)
	"high": {
		"llama3.1:8b",         // 8B - высокое качество
		"qwen2.5:7b",          // 7B - отличный для кода
		"codellama:7b",        // 7B - специализированная для кода
		"deepseek-coder:6.7b", // 6.7B - лучшая для программирования
	},
}

// AllOllamaModels все доступные модели
func AllOllamaModels() []string {
	all := []string{}
	for _, models := range OllamaModels {
		all = append(all, models...)
	}
	return all
}

// HuggingFaceModels список моделей HuggingFace для сравнения
// Модели из начала, середины и конца списка по размеру параметров
var HuggingFaceModels = []string{
	// Начало списка - маленькие модели (0.5B - 1.5B)
	"qwen2.5:0.5b", // 0.5B параметров
	"qwen2.5:1.5b", // 1.5B параметров

	// Середина списка - средние модели (3B - 7B)
	"llama3.2:3b", // 3B параметров
	"mistral:7b",  // 7B параметров

	// Конец списка - большие модели (8B+)
	"llama3.1:8b", // 8B параметров
	"qwen2.5:7b",  // 7B параметров (близко к концу для вашей системы)
}

// GetHuggingFaceModelsForComparison возвращает список моделей для сравнения
// Формат: "provider:model" для использования в API
func GetHuggingFaceModelsForComparison() []string {
	models := make([]string, len(HuggingFaceModels))
	for i, model := range HuggingFaceModels {
		models[i] = "ollama:" + model
	}
	return models
}

// NewOllamaProvider создает новый Ollama провайдер
func NewOllamaProvider(cfg OllamaConfig) *OllamaProvider {
	apiURL := cfg.APIURL
	if apiURL == "" {
		apiURL = "http://localhost:11434"
	}

	model := cfg.Model
	if model == "" {
		model = "llama3.2:3b" // Оптимальный по умолчанию
	}

	return &OllamaProvider{
		httpClient: &http.Client{
			Timeout: 300 * time.Second, // Локальные модели могут быть медленными
		},
		apiURL: apiURL,
		model:  model,
	}
}

// Name возвращает имя провайдера
func (p *OllamaProvider) Name() string {
	return "ollama"
}

// Models возвращает список доступных моделей
func (p *OllamaProvider) Models() []string {
	return AllOllamaModels()
}

// SetModel устанавливает модель
func (p *OllamaProvider) SetModel(model string) {
	p.model = model
}

// GetModel возвращает текущую модель
func (p *OllamaProvider) GetModel() string {
	return p.model
}

// GetMaxTokens возвращает максимальный лимит токенов для текущей модели
func (p *OllamaProvider) GetMaxTokens() int {
	// Лимиты зависят от модели, используем консервативные значения
	// Большинство локальных моделей имеют лимит 2048-4096
	// Для больших моделей может быть больше
	if strings.Contains(p.model, "8b") || strings.Contains(p.model, "7b") {
		return 4096
	}
	if strings.Contains(p.model, "3b") || strings.Contains(p.model, "2b") {
		return 2048
	}
	// Для очень маленьких моделей
	return 2048
}

// CalculateCost вычисляет стоимость запроса в USD
// Ollama - локальная модель, стоимость = 0
func (p *OllamaProvider) CalculateCost(inputTokens, outputTokens int) float64 {
	return 0.0
}

// ollamaChatRequest запрос к Ollama API
type ollamaChatRequest struct {
	Model    string          `json:"model"`
	Messages []ollamaMessage `json:"messages"`
	Stream   bool            `json:"stream"`
	Options  *ollamaOptions  `json:"options,omitempty"`
}

type ollamaMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type ollamaOptions struct {
	NumPredict  int     `json:"num_predict,omitempty"` // max_tokens
	Temperature float64 `json:"temperature,omitempty"`
}

type ollamaChatResponse struct {
	Model      string        `json:"model"`
	Message    ollamaMessage `json:"message"`
	Done       bool          `json:"done"`
	DoneReason string        `json:"done_reason,omitempty"`
}

// Chat отправляет сообщение через Ollama API
func (p *OllamaProvider) Chat(ctx context.Context, message string, opts *ChatOptions, onChunk func(string) error) error {
	messages := []ollamaMessage{}

	// System prompt - объединяем все части
	var systemPrompt string
	if opts != nil {
		// Базовый system prompt от пользователя
		if opts.SystemPrompt != "" {
			systemPrompt = opts.SystemPrompt
		}

		// Добавляем режим рассуждения
		if opts.ReasoningMode != "" && opts.ReasoningMode != "direct" {
			reasoningPrompt := BuildReasoningPrompt(opts.ReasoningMode, "")
			if systemPrompt != "" {
				systemPrompt = systemPrompt + "\n\n" + reasoningPrompt
			} else {
				systemPrompt = reasoningPrompt
			}
		}

		// Добавляем JSON-инструкцию
		if opts.JSONFormat && opts.JSONSchemaText != "" {
			jsonPrompt := BuildJSONPrompt(opts.JSONSchemaText)
			if systemPrompt != "" {
				systemPrompt = systemPrompt + "\n\n" + jsonPrompt
			} else {
				systemPrompt = jsonPrompt
			}
		}
	}

	if systemPrompt != "" {
		messages = append(messages, ollamaMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// История
	if opts != nil && len(opts.History) > 0 {
		for _, msg := range opts.History {
			messages = append(messages, ollamaMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	// Текущее сообщение
	messages = append(messages, ollamaMessage{
		Role:    "user",
		Content: message,
	})

	reqBody := ollamaChatRequest{
		Model:    p.model,
		Messages: messages,
		Stream:   true,
	}

	if opts != nil && (opts.MaxTokens > 0 || opts.Temperature >= 0) {
		reqBody.Options = &ollamaOptions{}
		if opts.MaxTokens > 0 {
			reqBody.Options.NumPredict = opts.MaxTokens
		}
		if opts.Temperature >= 0 {
			reqBody.Options.Temperature = opts.Temperature
		}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга запроса: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiURL+"/api/chat", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса (убедитесь что Ollama запущена: ollama serve): %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ошибка Ollama API: %d - %s", resp.StatusCode, string(body))
	}

	scanner := bufio.NewScanner(resp.Body)
	// Увеличиваем буфер для больших ответов
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var chatResp ollamaChatResponse
		if err := json.Unmarshal([]byte(line), &chatResp); err != nil {
			continue
		}

		if chatResp.Message.Content != "" {
			if err := onChunk(chatResp.Message.Content); err != nil {
				return err
			}
		}

		if chatResp.Done {
			break
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("ошибка чтения потока: %w", err)
	}

	return nil
}

// ListModels получает список установленных моделей из Ollama
func (p *OllamaProvider) ListModels(ctx context.Context) ([]string, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", p.apiURL+"/api/tags", nil)
	if err != nil {
		return nil, err
	}

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("не удалось подключиться к Ollama: %w", err)
	}
	defer resp.Body.Close()

	var result struct {
		Models []struct {
			Name string `json:"name"`
		} `json:"models"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	models := make([]string, len(result.Models))
	for i, m := range result.Models {
		models[i] = m.Name
	}

	return models, nil
}

// PullModel скачивает модель (для справки, вызывается через CLI)
func (p *OllamaProvider) PullModel(ctx context.Context, model string, onProgress func(string) error) error {
	reqBody := map[string]interface{}{
		"name":   model,
		"stream": true,
	}

	jsonData, _ := json.Marshal(reqBody)
	req, err := http.NewRequestWithContext(ctx, "POST", p.apiURL+"/api/pull", bytes.NewBuffer(jsonData))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		var status struct {
			Status string `json:"status"`
		}
		if json.Unmarshal(scanner.Bytes(), &status) == nil && status.Status != "" {
			if onProgress != nil {
				if err := onProgress(status.Status); err != nil {
					return err
				}
			}
		}
	}

	return scanner.Err()
}
