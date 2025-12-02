package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/nnk/97-aic/backend/config"
	"github.com/nnk/97-aic/backend/logger"
	"github.com/nnk/97-aic/backend/storage"
)

// HistoryHandler обрабатывает запросы к /api/history
type HistoryHandler struct {
	Storage *storage.Storage
	Config  *config.Config
}

// NewHistoryHandler создает новый обработчик истории
func NewHistoryHandler(store *storage.Storage, cfg *config.Config) *HistoryHandler {
	return &HistoryHandler{
		Storage: store,
		Config:  cfg,
	}
}

// ServeHTTP обрабатывает HTTP запросы для получения истории сообщений
func (h *HistoryHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	limitStr := r.URL.Query().Get("limit")

	limit := h.Config.DefaultQueryLimit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Ограничиваем максимальный limit
	if limit > h.Config.MaxQueryLimit {
		limit = h.Config.MaxQueryLimit
	}

	messages, err := h.Storage.GetMessages(sessionID, limit)
	if err != nil {
		logger.Error("ошибка получения истории", "error", err, "session_id", sessionID)
		http.Error(w, "Ошибка получения истории", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(messages); err != nil {
		logger.Error("ошибка кодирования ответа", "error", err)
	}
}

// LogsHandler обрабатывает запросы к /api/logs
type LogsHandler struct {
	Storage *storage.Storage
	Config  *config.Config
}

// NewLogsHandler создает новый обработчик логов
func NewLogsHandler(store *storage.Storage, cfg *config.Config) *LogsHandler {
	return &LogsHandler{
		Storage: store,
		Config:  cfg,
	}
}

// ServeHTTP обрабатывает HTTP запросы для получения логов запросов
func (h *LogsHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("session_id")
	limitStr := r.URL.Query().Get("limit")

	limit := h.Config.DefaultQueryLimit
	if limitStr != "" {
		if l, err := strconv.Atoi(limitStr); err == nil && l > 0 {
			limit = l
		}
	}

	// Ограничиваем максимальный limit
	if limit > h.Config.MaxQueryLimit {
		limit = h.Config.MaxQueryLimit
	}

	logs, err := h.Storage.GetRequestLogs(sessionID, limit)
	if err != nil {
		logger.Error("ошибка получения логов", "error", err, "session_id", sessionID)
		http.Error(w, "Ошибка получения логов", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(logs); err != nil {
		logger.Error("ошибка кодирования ответа", "error", err)
	}
}

// HealthHandler обрабатывает запросы к /health
type HealthHandler struct {
	Storage *storage.Storage
}

// NewHealthHandler создает новый обработчик health check
func NewHealthHandler(store *storage.Storage) *HealthHandler {
	return &HealthHandler{Storage: store}
}

// ServeHTTP обрабатывает HTTP запросы для health check
func (h *HealthHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	status := map[string]string{
		"status": "ok",
	}

	// Проверяем подключение к БД
	if h.Storage != nil {
		if _, err := h.Storage.GetMessages("_health_check_", 1); err != nil {
			status["status"] = "degraded"
			status["database"] = "error"
		} else {
			status["database"] = "ok"
		}
	}

	w.Header().Set("Content-Type", "application/json")
	if status["status"] != "ok" {
		w.WriteHeader(http.StatusServiceUnavailable)
	}
	json.NewEncoder(w).Encode(status)
}
