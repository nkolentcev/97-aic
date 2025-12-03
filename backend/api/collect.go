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

// CollectHandler обрабатывает запросы к /api/chat/collect
type CollectHandler struct {
	GigaChatClient *gigachat.Client
	Storage        *storage.Storage
}

// CollectRequest представляет входящий запрос для режима сбора требований
type CollectRequest struct {
	Message           string                  `json:"message"`
	SessionID         string                  `json:"session_id,omitempty"`
	CollectConfig     *gigachat.CollectConfig `json:"collect_config,omitempty"`
	StartNewSession   bool                    `json:"start_new_session,omitempty"` // Начать новую сессию сбора
}

// CollectResponse представляет ответ режима сбора
type CollectResponse struct {
	SessionID string                 `json:"session_id"`
	Status    string                 `json:"status"`              // "collecting", "ready", "error"
	Question  string                 `json:"question,omitempty"`  // Следующий вопрос
	Collected []string               `json:"collected,omitempty"` // Собранные данные
	Result    string                 `json:"result,omitempty"`    // Финальный результат
	Error     string                 `json:"error,omitempty"`     // Ошибка
	RawResponse string               `json:"raw_response,omitempty"` // Сырой ответ модели
}

// NewCollectHandler создает новый обработчик режима сбора требований
func NewCollectHandler(client *gigachat.Client, store *storage.Storage) *CollectHandler {
	return &CollectHandler{
		GigaChatClient: client,
		Storage:        store,
	}
}

// ServeHTTP обрабатывает HTTP запросы
func (h *CollectHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	startTime := time.Now()
	ctx := r.Context()

	if r.Method != http.MethodPost {
		logger.Warn("неверный метод", "method", r.Method, "path", r.URL.Path)
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	// Читаем тело запроса
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		logger.Error("ошибка чтения тела запроса", "error", err)
		sendCollectError(w, "", "Ошибка чтения запроса", http.StatusBadRequest)
		return
	}

	var req CollectRequest
	decoder := json.NewDecoder(bytes.NewBuffer(bodyBytes))
	if err = decoder.Decode(&req); err != nil {
		logger.Warn("ошибка парсинга JSON", "error", err)
		sendCollectError(w, "", fmt.Sprintf("Ошибка парсинга запроса: %v", err), http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		logger.Warn("пустое сообщение")
		sendCollectError(w, "", "Поле message обязательно", http.StatusBadRequest)
		return
	}

	// Генерируем session_id если не передан или начинаем новую сессию
	if req.SessionID == "" || req.StartNewSession {
		req.SessionID = fmt.Sprintf("collect_%d", time.Now().UnixNano())
	}

	logger.Info("получен запрос на сбор требований",
		"message_length", len(req.Message),
		"session_id", req.SessionID,
		"has_collect_config", req.CollectConfig != nil,
	)

	// Загружаем историю сообщений
	var history []gigachat.Message
	if h.Storage != nil && !req.StartNewSession {
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

	// Подготавливаем конфигурацию сбора требований
	collectConfig := req.CollectConfig
	if collectConfig == nil {
		collectConfig = &gigachat.CollectConfig{
			Enabled: true,
			Role:    "технический аналитик",
			Goal:    "техническое задание",
		}
	}
	collectConfig.Enabled = true

	// Подготавливаем опции для запроса
	opts := &gigachat.ChatOptions{
		CollectConfig: collectConfig,
		History:       history,
	}

	// Собираем ответ
	var fullResponse string
	err = h.GigaChatClient.ChatWithOptions(ctx, req.Message, opts, func(chunk string) error {
		fullResponse += chunk
		return nil
	})

	durationMs := time.Since(startTime).Milliseconds()

	if err != nil {
		logger.Error("ошибка при обработке запроса",
			"error", err,
			"duration_ms", durationMs,
			"session_id", req.SessionID,
		)
		sendCollectError(w, req.SessionID, fmt.Sprintf("Ошибка API: %v", err), http.StatusInternalServerError)
		return
	}

	// Сохраняем ответ ассистента
	if h.Storage != nil && fullResponse != "" {
		if _, err := h.Storage.SaveMessage(req.SessionID, "assistant", fullResponse); err != nil {
			logger.Error("ошибка сохранения ответа ассистента", "error", err)
		}
	}

	// Парсим ответ для определения статуса
	response := CollectResponse{
		SessionID:   req.SessionID,
		RawResponse: fullResponse,
	}

	status, parseErr := gigachat.ParseCollectResponse(fullResponse)
	if parseErr != nil {
		logger.Warn("не удалось распарсить JSON-ответ, возвращаем сырой ответ",
			"error", parseErr,
			"response_length", len(fullResponse),
		)
		response.Status = "raw"
		response.Result = fullResponse
	} else {
		response.Status = status.Status
		response.Question = status.Question
		response.Collected = status.Collected
		response.Result = status.Result
	}

	// Логируем запрос/ответ в БД
	if h.Storage != nil {
		requestJSON, _ := json.Marshal(req)
		responseJSON, _ := json.Marshal(response)
		if _, err := h.Storage.SaveRequestLog(req.SessionID, string(requestJSON), string(responseJSON), http.StatusOK, durationMs); err != nil {
			logger.Error("ошибка сохранения лога запроса", "error", err)
		}
	}

	logger.Info("запрос на сбор требований обработан",
		"session_id", req.SessionID,
		"duration_ms", durationMs,
		"status", response.Status,
	)

	// Отправляем ответ
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// sendCollectError отправляет ошибку в формате CollectResponse
func sendCollectError(w http.ResponseWriter, sessionID, errorMsg string, statusCode int) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(CollectResponse{
		SessionID: sessionID,
		Status:    "error",
		Error:     errorMsg,
	})
}

