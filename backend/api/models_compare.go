package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/nnk/97-aic/backend/logger"
	"github.com/nnk/97-aic/backend/provider"
)

// ModelsCompareHandler обрабатывает запросы для сравнения моделей
type ModelsCompareHandler struct {
	ProviderManager *provider.Manager
}

// CompareRequest запрос на сравнение моделей
type CompareRequest struct {
	Message string   `json:"message"` // Единый запрос для всех моделей
	Models  []string `json:"models"`  // Список моделей для сравнения (формат: "provider:model")
}

// ModelResult результат выполнения запроса на одной модели
type ModelResult struct {
	Provider      string  `json:"provider"`
	Model         string  `json:"model"`
	Response      string  `json:"response"`
	DurationMs    int64   `json:"duration_ms"`
	TokensInput   int     `json:"tokens_input,omitempty"`
	TokensOutput  int     `json:"tokens_output,omitempty"`
	TokensTotal   int     `json:"tokens_total,omitempty"`
	Cost          float64 `json:"cost,omitempty"` // Стоимость в USD
	Error         string  `json:"error,omitempty"`
	ResponseTime  float64 `json:"response_time"` // Время ответа в секундах
	TokensPerSec  float64 `json:"tokens_per_sec,omitempty"` // Скорость генерации
}

// CompareResponse ответ с результатами сравнения
type CompareResponse struct {
	Message      string       `json:"message"`
	Results      []ModelResult `json:"results"`
	Summary      Summary      `json:"summary"`
	Comparison   Comparison   `json:"comparison"`
}

// Summary сводка по результатам
type Summary struct {
	TotalModels   int     `json:"total_models"`
	SuccessCount  int     `json:"success_count"`
	ErrorCount    int     `json:"error_count"`
	AvgDurationMs int64   `json:"avg_duration_ms"`
	FastestModel  string  `json:"fastest_model"`
	SlowestModel  string  `json:"slowest_model"`
	TotalCost     float64 `json:"total_cost"`
}

// Comparison сравнение качества ответов
type Comparison struct {
	BestResponse    string `json:"best_response,omitempty"` // Модель с лучшим ответом (по длине/качеству)
	LongestResponse string `json:"longest_response,omitempty"`
	ShortestResponse string `json:"shortest_response,omitempty"`
}

// NewModelsCompareHandler создает новый обработчик сравнения моделей
func NewModelsCompareHandler(pm *provider.Manager) *ModelsCompareHandler {
	return &ModelsCompareHandler{
		ProviderManager: pm,
	}
}

// ServeHTTP обрабатывает HTTP запросы
func (h *ModelsCompareHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения запроса", http.StatusBadRequest)
		return
	}

	var req CompareRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка парсинга запроса: %v", err), http.StatusBadRequest)
		return
	}

	if req.Message == "" {
		http.Error(w, "Поле message обязательно", http.StatusBadRequest)
		return
	}

	// Если модели не указаны, используем список по умолчанию
	if len(req.Models) == 0 {
		req.Models = provider.GetHuggingFaceModelsForComparison()
		logger.Info("используются модели по умолчанию", "models", req.Models)
	}

	// Проверяем доступность провайдера Ollama
	if _, err := h.ProviderManager.Get("ollama"); err != nil {
		logger.Warn("Ollama провайдер не зарегистрирован", "error", err, "available", h.ProviderManager.List())
		http.Error(w, fmt.Sprintf("Ollama провайдер не настроен. Доступные провайдеры: %v", h.ProviderManager.List()), http.StatusBadRequest)
		return
	}

	logger.Info("начато сравнение моделей",
		"message_length", len(req.Message),
		"models_count", len(req.Models),
		"models", req.Models,
	)

	// Выполняем запросы ко всем моделям
	results := h.compareModels(r.Context(), req.Message, req.Models)

	// Формируем сводку
	summary := h.buildSummary(results)
	comparison := h.buildComparison(results)

	response := CompareResponse{
		Message:    req.Message,
		Results:    results,
		Summary:    summary,
		Comparison: comparison,
	}

	logger.Info("сравнение моделей завершено",
		"success_count", summary.SuccessCount,
		"error_count", summary.ErrorCount,
	)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("ошибка кодирования ответа", "error", err)
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

// compareModels выполняет запросы ко всем моделям параллельно
func (h *ModelsCompareHandler) compareModels(ctx context.Context, message string, models []string) []ModelResult {
	results := make([]ModelResult, 0, len(models))

	for _, modelSpec := range models {
		result := h.testModel(ctx, message, modelSpec)
		results = append(results, result)
	}

	return results
}

// testModel тестирует одну модель
func (h *ModelsCompareHandler) testModel(ctx context.Context, message string, modelSpec string) ModelResult {
	startTime := time.Now()

	// Парсим формат "provider:model" или просто "model" (используем провайдер по умолчанию)
	parts := strings.SplitN(modelSpec, ":", 2)
	var providerName, modelName string

	if len(parts) == 2 {
		providerName = parts[0]
		modelName = parts[1]
	} else {
		// Если провайдер не указан, используем ollama по умолчанию для HuggingFace моделей
		providerName = "ollama"
		modelName = parts[0]
	}

	result := ModelResult{
		Provider: providerName,
		Model:    modelName,
	}

	// Получаем провайдер
	p, err := h.ProviderManager.Get(providerName)
	if err != nil {
		result.Error = fmt.Sprintf("Провайдер не найден: %v. Доступные провайдеры: %v", err, h.ProviderManager.List())
		result.DurationMs = time.Since(startTime).Milliseconds()
		logger.Error("провайдер не найден",
			"provider", providerName,
			"available", h.ProviderManager.List(),
			"error", err,
		)
		return result
	}

	// Устанавливаем модель
	p.SetModel(modelName)

	// Выполняем запрос
	var fullResponse strings.Builder
	var tokenCount int

	opts := &provider.ChatOptions{
		Temperature: 0.7,
		MaxTokens:   1000,
	}

	err = p.Chat(ctx, message, opts, func(chunk string) error {
		fullResponse.WriteString(chunk)
		// Простой подсчет токенов (приблизительный: ~4 символа на токен)
		tokenCount += len(chunk) / 4
		return nil
	})

	duration := time.Since(startTime)
	result.DurationMs = duration.Milliseconds()
	result.ResponseTime = duration.Seconds()
	result.Response = fullResponse.String()
	result.TokensOutput = tokenCount
	result.TokensTotal = tokenCount

	// Вычисляем скорость генерации
	if result.ResponseTime > 0 {
		result.TokensPerSec = float64(result.TokensOutput) / result.ResponseTime
	}

	if err != nil {
		result.Error = err.Error()
		logger.Error("ошибка при тестировании модели",
			"provider", providerName,
			"model", modelName,
			"error", err,
			"duration_ms", result.DurationMs,
		)
		return result
	}

	logger.Debug("модель успешно протестирована",
		"provider", providerName,
		"model", modelName,
		"duration_ms", result.DurationMs,
		"tokens", result.TokensTotal,
	)

	// Для платных моделей вычисляем стоимость (заглушка, нужно реализовать для каждого провайдера)
	result.Cost = h.calculateCost(providerName, modelName, result.TokensTotal)

	logger.Info("модель протестирована",
		"provider", providerName,
		"model", modelName,
		"duration_ms", result.DurationMs,
		"tokens", result.TokensTotal,
	)

	return result
}

// calculateCost вычисляет стоимость запроса (заглушка)
func (h *ModelsCompareHandler) calculateCost(provider, model string, tokens int) float64 {
	// Для локальных моделей (Ollama) стоимость = 0
	if provider == "ollama" {
		return 0.0
	}

	// Для платных провайдеров можно добавить реальные цены
	// Например, для Groq или GigaChat
	// Пока возвращаем 0 для всех
	return 0.0
}

// buildSummary строит сводку по результатам
func (h *ModelsCompareHandler) buildSummary(results []ModelResult) Summary {
	summary := Summary{
		TotalModels: len(results),
	}

	var totalDuration int64
	var fastestTime int64 = -1
	var slowestTime int64 = -1
	var fastestModel, slowestModel string

	for _, r := range results {
		if r.Error == "" {
			summary.SuccessCount++
			totalDuration += r.DurationMs

			if fastestTime == -1 || r.DurationMs < fastestTime {
				fastestTime = r.DurationMs
				fastestModel = fmt.Sprintf("%s:%s", r.Provider, r.Model)
			}

			if r.DurationMs > slowestTime {
				slowestTime = r.DurationMs
				slowestModel = fmt.Sprintf("%s:%s", r.Provider, r.Model)
			}

			summary.TotalCost += r.Cost
		} else {
			summary.ErrorCount++
		}
	}

	if summary.SuccessCount > 0 {
		summary.AvgDurationMs = totalDuration / int64(summary.SuccessCount)
	}

	summary.FastestModel = fastestModel
	summary.SlowestModel = slowestModel

	return summary
}

// buildComparison строит сравнение качества ответов
func (h *ModelsCompareHandler) buildComparison(results []ModelResult) Comparison {
	comp := Comparison{}

	var longestLen, shortestLen int = -1, -1
	var longestModel, shortestModel string

	for _, r := range results {
		if r.Error == "" {
			length := len(r.Response)

			if longestLen == -1 || length > longestLen {
				longestLen = length
				longestModel = fmt.Sprintf("%s:%s", r.Provider, r.Model)
			}

			if shortestLen == -1 || length < shortestLen {
				shortestLen = length
				shortestModel = fmt.Sprintf("%s:%s", r.Provider, r.Model)
			}
		}
	}

	comp.LongestResponse = longestModel
	comp.ShortestResponse = shortestModel
	comp.BestResponse = longestModel // Упрощенно: лучший = самый длинный

	return comp
}
