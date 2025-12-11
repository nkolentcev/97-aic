package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nnk/97-aic/backend/gigachat"
	"github.com/nnk/97-aic/backend/logger"
	"github.com/nnk/97-aic/backend/storage"
)

// ChatHandler обрабатывает запросы к /api/chat
type ChatHandler struct {
	GigaChatClient *gigachat.Client
	Storage        *storage.Storage
}

// ChatRequest представляет входящий запрос
type ChatRequest struct {
	Message      string                `json:"message"`
	SessionID    string                `json:"session_id,omitempty"`
	UseHistory   bool                  `json:"use_history,omitempty"` // Использовать историю сессии
	ResponseJSON *gigachat.JSONConfig  `json:"response_json,omitempty"`
	Options      *gigachat.ChatOptions `json:"options,omitempty"` // Расширенные параметры
}

// NewChatHandler создает новый обработчик чата
func NewChatHandler(client *gigachat.Client, store *storage.Storage) *ChatHandler {
	return &ChatHandler{
		GigaChatClient: client,
		Storage:        store,
	}
}

// ServeHTTP обрабатывает HTTP запросы
func (h *ChatHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()

	if r.Method != http.MethodPost {
		logger.Warn("неверный метод", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	logger.Debug("получен запрос",
		"content_type", r.Header.Get("Content-Type"),
		"content_length", r.Header.Get("Content-Length"),
	)

	// Читаем тело запроса (уже ограничено middleware)
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("ошибка чтения тела запроса", "error", err)
		http.Error(w, "Ошибка чтения запроса", http.StatusBadRequest)
		return
	}

	var req ChatRequest
	decoder := json.NewDecoder(bytes.NewBuffer(bodyBytes))
	decoder.DisallowUnknownFields()
	if err = decoder.Decode(&req); err != nil {
		logger.Warn("ошибка парсинга JSON", "error", err)
		http.Error(w, fmt.Sprintf("Ошибка парсинга запроса: %v", err), http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		logger.Warn("пустое сообщение")
		http.Error(w, "Поле message обязательно", http.StatusBadRequest)
		return
	}

	logger.Info("получено сообщение",
		"message_length", len(req.Message),
		"session_id", req.SessionID,
		"has_json_config", req.ResponseJSON != nil,
		"use_history", req.UseHistory,
	)

	// Генерируем session_id если не передан
	if req.SessionID == "" {
		req.SessionID = fmt.Sprintf("session_%d", time.Now().UnixNano())
	}

	// Загружаем историю сообщений если нужно
	var history []gigachat.Message
	if req.UseHistory && h.Storage != nil {
		messages, err := h.Storage.GetMessages(req.SessionID, 100)
		if err != nil {
			logger.Warn("ошибка загрузки истории", "error", err, "session_id", req.SessionID)
		} else {
			for _, msg := range messages {
				history = append(history, gigachat.Message{
					Role:    msg.Role,
					Content: msg.Content,
				})
			}
			logger.Debug("загружена история", "count", len(history), "session_id", req.SessionID)
		}
	}

	// Сохраняем сообщение пользователя
	if h.Storage != nil {
		if _, err := h.Storage.SaveMessage(req.SessionID, "user", req.Message); err != nil {
			logger.Error("ошибка сохранения сообщения пользователя", "error", err)
		}
	}

	// Настройка для streaming ответа
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "Streaming не поддерживается", http.StatusInternalServerError)
		return
	}

	// Собираем полный ответ для сохранения
	var fullResponse string

	// Подготавливаем опции для запроса
	opts := req.Options
	if opts == nil {
		opts = &gigachat.ChatOptions{}
	}
	// Устанавливаем JSONConfig если передан в старом формате
	if req.ResponseJSON != nil {
		opts.JSONConfig = req.ResponseJSON
	}
	// Добавляем историю
	opts.History = history

	// Отправка сообщений через streaming
	err = h.GigaChatClient.ChatWithOptions(ctx, req.Message, opts, func(chunk string) error {
		fullResponse += chunk

		data := map[string]string{"content": chunk}
		jsonData, marshalErr := json.Marshal(data)
		if marshalErr != nil {
			return marshalErr
		}

		if _, writeErr := fmt.Fprintf(w, "data: %s\n\n", jsonData); writeErr != nil {
			return writeErr
		}
		flusher.Flush()
		return nil
	})

	durationMs := time.Since(startTime).Milliseconds()
	statusCode := http.StatusOK

	if err != nil {
		logger.Error("ошибка при обработке запроса",
			"error", err,
			"duration_ms", durationMs,
			"session_id", req.SessionID,
		)
		statusCode = http.StatusInternalServerError
		errorData := map[string]string{"error": err.Error()}
		jsonData, _ := json.Marshal(errorData)
		fmt.Fprintf(w, "data: %s\n\n", jsonData)
		flusher.Flush()
	} else {
		// Сохраняем ответ ассистента
		if h.Storage != nil && fullResponse != "" {
			if _, err := h.Storage.SaveMessage(req.SessionID, "assistant", fullResponse); err != nil {
				logger.Error("ошибка сохранения ответа ассистента", "error", err)
			}
		}
	}

	// Логируем полный запрос/ответ в БД
	if h.Storage != nil {
		requestJSON, _ := json.Marshal(map[string]interface{}{
			"message":       req.Message,
			"session_id":    req.SessionID,
			"response_json": req.ResponseJSON,
		})
		responseJSON, _ := json.Marshal(map[string]interface{}{
			"content": fullResponse,
			"status":  statusCode,
		})
		if _, err := h.Storage.SaveRequestLog(req.SessionID, string(requestJSON), string(responseJSON), statusCode, durationMs, nil, nil, nil, nil); err != nil {
			logger.Error("ошибка сохранения лога запроса", "error", err)
		}
	}

	logger.Info("запрос обработан",
		"session_id", req.SessionID,
		"duration_ms", durationMs,
		"response_length", len(fullResponse),
		"status", statusCode,
	)

	// Отправка сигнала завершения
	fmt.Fprintf(w, "data: [DONE]\n\n")
	flusher.Flush()
}
