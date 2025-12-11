package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nnk/97-aic/backend/logger"
	"github.com/nnk/97-aic/backend/provider"
	"github.com/nnk/97-aic/backend/storage"
)

// ChatHandlerV2 обрабатывает запросы к /api/v2/chat с поддержкой провайдеров
type ChatHandlerV2 struct {
	ProviderManager *provider.Manager
	Storage         *storage.Storage
}

// ChatRequestV2 запрос к API v2
type ChatRequestV2 struct {
	Message    string `json:"message"`
	SessionID  string `json:"session_id,omitempty"`
	UseHistory bool   `json:"use_history,omitempty"`

	// Провайдер и модель
	Provider string `json:"provider,omitempty"` // gigachat, groq, ollama
	Model    string `json:"model,omitempty"`    // конкретная модель

	// System Prompt (День 5)
	SystemPrompt string `json:"system_prompt,omitempty"`

	// Режим рассуждения (День 4)
	ReasoningMode string `json:"reasoning_mode,omitempty"` // direct, step_by_step, experts

	// JSON формат
	JSONFormat bool   `json:"json_format,omitempty"`
	JSONSchema string `json:"json_schema,omitempty"`

	// Параметры генерации
	MaxTokens   int     `json:"max_tokens,omitempty"`
	Temperature float64 `json:"temperature,omitempty"`
}

// NewChatHandlerV2 создает новый обработчик
func NewChatHandlerV2(pm *provider.Manager, store *storage.Storage) *ChatHandlerV2 {
	return &ChatHandlerV2{
		ProviderManager: pm,
		Storage:         store,
	}
}

// ServeHTTP обрабатывает HTTP запросы
func (h *ChatHandlerV2) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()

	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения запроса", http.StatusBadRequest)
		return
	}

	var req ChatRequestV2
	decoder := json.NewDecoder(bytes.NewBuffer(bodyBytes))
	if err = decoder.Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка парсинга запроса: %v", err), http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Поле message обязательно", http.StatusBadRequest)
		return
	}

	// Получаем провайдер
	p, err := h.ProviderManager.Get(req.Provider)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка провайдера: %v", err), http.StatusBadRequest)
		return
	}

	// Устанавливаем модель если указана
	if req.Model != "" {
		p.SetModel(req.Model)
	}

	logger.Info("v2 запрос",
		"provider", p.Name(),
		"model", p.GetModel(),
		"reasoning_mode", req.ReasoningMode,
		"system_prompt_length", len(req.SystemPrompt),
		"message_length", len(req.Message),
	)

	// Debug: выводим system prompt
	if req.SystemPrompt != "" {
		fmt.Printf("[DEBUG] System Prompt: %s\n", req.SystemPrompt)
	}

	// Генерируем session_id
	if req.SessionID == "" {
		req.SessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
	}

	// Загружаем историю
	var history []provider.Message
	if req.UseHistory && h.Storage != nil {
		messages, err := h.Storage.GetMessages(req.SessionID, 100)
		if err != nil {
			logger.Warn("ошибка загрузки истории", "error", err)
		} else {
			for _, msg := range messages {
				history = append(history, provider.Message{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
		}
	}

	// Сохраняем сообщение пользователя
	if h.Storage != nil {
		h.Storage.SaveMessage(req.SessionID, "user", req.Message)
	}

	// Настройка streaming
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming не поддерживается", http.StatusInternalServerError)
		return
	}

	var fullResponse string

	// Подготавливаем опции
	opts := &provider.ChatOptions{
		SystemPrompt:   req.SystemPrompt,
		History:        history,
		MaxTokens:      req.MaxTokens,
		Temperature:    req.Temperature,
		ReasoningMode:  req.ReasoningMode,
		JSONFormat:     req.JSONFormat,
		JSONSchemaText: req.JSONSchema,
	}

	// Подсчитываем токены запроса перед отправкой
	tokensInput := provider.CountTokensForMessages(req.SystemPrompt, history, req.Message)

	// Отправляем запрос
	err = p.Chat(ctx, req.Message, opts, func(chunk string) error {
		fullResponse += chunk

		data := map[string]string{"content": chunk}
		jsonData, _ := json.Marshal(data)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
		return nil
	})

	durationMs := time.Since(startTime).Milliseconds()
	statusCode := http.StatusOK

	// Подсчитываем токены ответа (приблизительно)
	tokensOutput := provider.CountTokens(fullResponse)
	tokensTotal := tokensInput + tokensOutput

	// Вычисляем стоимость
	cost := p.CalculateCost(tokensInput, tokensOutput)

	if err != nil {
		logger.Error("ошибка при обработке запроса", "error", err, "duration_ms", durationMs)
		statusCode = http.StatusInternalServerError
		errorData := map[string]string{"error": err.Error()}
		jsonData, _ := json.Marshal(errorData)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	} else {
		// Сохраняем ответ
		if h.Storage != nil && fullResponse != "" {
			h.Storage.SaveMessage(req.SessionID, "assistant", fullResponse)
		}
	}

	// Логируем запрос
	if h.Storage != nil {
		requestJSON, _ := json.Marshal(map[string]interface{}{
			"message":        req.Message,
			"session_id":     req.SessionID,
			"provider":       p.Name(),
			"model":          p.GetModel(),
			"reasoning_mode": req.ReasoningMode,
			"system_prompt":  req.SystemPrompt,
			"tokens_input":   tokensInput,
		})

		// Формируем response JSON с учетом ошибок
		responseData := map[string]interface{}{
			"content":       fullResponse,
			"status":        statusCode,
			"tokens_input":  tokensInput,
			"tokens_output": tokensOutput,
			"tokens_total":  tokensTotal,
			"cost":          cost,
		}
		if err != nil {
			responseData["error"] = err.Error()
		}

		responseJSON, _ := json.Marshal(responseData)

		// Сохраняем логи с токенами и стоимостью
		h.Storage.SaveRequestLog(
			req.SessionID,
			string(requestJSON),
			string(responseJSON),
			statusCode,
			durationMs,
			&tokensInput,
			&tokensOutput,
			&tokensTotal,
			&cost,
		)
	}

	logger.Info("v2 запрос обработан",
		"session_id", req.SessionID,
		"provider", p.Name(),
		"duration_ms", durationMs,
		"response_length", len(fullResponse),
		"tokens_input", tokensInput,
		"tokens_output", tokensOutput,
		"tokens_total", tokensTotal,
		"cost", cost,
	)

	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}

// ProvidersHandler возвращает список доступных провайдеров
type ProvidersHandler struct {
	ProviderManager *provider.Manager
}

// NewProvidersHandler создает обработчик
func NewProvidersHandler(pm *provider.Manager) *ProvidersHandler {
	return &ProvidersHandler{ProviderManager: pm}
}

// ServeHTTP обрабатывает запрос списка провайдеров
func (h *ProvidersHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	infos := h.ProviderManager.ListInfo()

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"providers":        infos,
		"default_provider": h.ProviderManager.GetDefaultName(),
		"reasoning_modes": []map[string]string{
			{"id": "direct", "name": "Прямой ответ", "description": "Краткий ответ без рассуждений"},
			{"id": "step_by_step", "name": "Пошаговое решение", "description": "Разбивает задачу на шаги"},
			{"id": "experts", "name": "Группа экспертов", "description": "Несколько экспертов дают мнения"},
		},
	})
}
