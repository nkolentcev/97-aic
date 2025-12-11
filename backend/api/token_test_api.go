package api

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/nnk/97-aic/backend/logger"
	"github.com/nnk/97-aic/backend/provider"
)

// TokenTestHandler обрабатывает запросы для тестирования токенов
type TokenTestHandler struct {
	ProviderManager *provider.Manager
}

// TokenTestRequest запрос на тестирование токенов
type TokenTestRequest struct {
	Provider string `json:"provider"` // Провайдер
	Model    string `json:"model"`    // Модель
	TestType string `json:"test_type"` // short, long, exceed_limit, all
}

// TokenTestResult результат одного теста
type TokenTestResult struct {
	TestType     string  `json:"test_type"`      // short, long, exceed_limit
	Message      string  `json:"message"`        // Текст запроса
	Response     string  `json:"response"`       // Ответ модели
	TokensInput  int     `json:"tokens_input"`   // Токены запроса
	TokensOutput int     `json:"tokens_output"`  // Токены ответа
	TokensTotal  int     `json:"tokens_total"`   // Всего токенов
	Cost         float64 `json:"cost"`           // Стоимость
	DurationMs   int64   `json:"duration_ms"`    // Время выполнения
	Success      bool    `json:"success"`        // Успех/ошибка
	Error        string  `json:"error,omitempty"` // Ошибка (если есть)
	MaxTokens    int     `json:"max_tokens"`     // Максимальный лимит модели
}

// TokenTestResponse ответ с результатами тестирования
type TokenTestResponse struct {
	Provider string           `json:"provider"`
	Model    string           `json:"model"`
	Results  []TokenTestResult `json:"results"`
	Summary  TokenTestSummary  `json:"summary"`
}

// TokenTestSummary сводка по тестированию
type TokenTestSummary struct {
	TotalTests    int     `json:"total_tests"`
	SuccessCount  int     `json:"success_count"`
	ErrorCount    int     `json:"error_count"`
	TotalTokens   int     `json:"total_tokens"`
	TotalCost     float64 `json:"total_cost"`
	AvgDurationMs int64   `json:"avg_duration_ms"`
}

// NewTokenTestHandler создает новый обработчик тестирования токенов
func NewTokenTestHandler(pm *provider.Manager) *TokenTestHandler {
	return &TokenTestHandler{
		ProviderManager: pm,
	}
}

// ServeHTTP обрабатывает HTTP запросы
func (h *TokenTestHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Метод не разрешен", http.StatusMethodNotAllowed)
		return
	}

	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Ошибка чтения запроса", http.StatusBadRequest)
		return
	}

	var req TokenTestRequest
	if err := json.Unmarshal(bodyBytes, &req); err != nil {
		http.Error(w, fmt.Sprintf("Ошибка парсинга запроса: %v", err), http.StatusBadRequest)
		return
	}

	// Получаем провайдер
	p, err := h.ProviderManager.Get(req.Provider)
	if err != nil {
		http.Error(w, fmt.Sprintf("Ошибка провайдера: %v", err), http.StatusBadRequest)
		return
	}

	// Устанавливаем модель
	if req.Model != "" {
		p.SetModel(req.Model)
	}

	maxTokens := p.GetMaxTokens()

	// Определяем какие тесты нужно выполнить
	testTypes := []string{}
	if req.TestType == "all" {
		testTypes = []string{"short", "long", "exceed_limit"}
	} else {
		testTypes = []string{req.TestType}
	}

	// Выполняем тесты
	var results []TokenTestResult
	for _, testType := range testTypes {
		result := h.runTest(r.Context(), p, testType, maxTokens)
		results = append(results, result)
	}

	// Формируем сводку
	summary := h.buildSummary(results)

	response := TokenTestResponse{
		Provider: p.Name(),
		Model:    p.GetModel(),
		Results:  results,
		Summary:  summary,
	}

	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(response); err != nil {
		logger.Error("ошибка кодирования ответа", "error", err)
		http.Error(w, "Ошибка формирования ответа", http.StatusInternalServerError)
		return
	}
}

// runTest выполняет один тест
func (h *TokenTestHandler) runTest(ctx context.Context, p provider.Provider, testType string, maxTokens int) TokenTestResult {
	result := TokenTestResult{
		TestType:  testType,
		MaxTokens: maxTokens,
	}

	// Генерируем запрос в зависимости от типа теста
	var message string
	var targetTokens int

	switch testType {
	case "short":
		targetTokens = 50
		message = provider.GenerateTextForTokens(targetTokens, "Это короткий тестовый запрос. ")
	case "long":
		// Генерируем запрос близкий к лимиту, но не превышающий его
		targetTokens = maxTokens - 500 // Оставляем запас
		if targetTokens < 1000 {
			targetTokens = 1000 // Минимум 1000 токенов для длинного запроса
		}
		message = provider.GenerateTextForTokens(targetTokens, "Это длинный тестовый запрос для проверки обработки больших объемов текста. ")
	case "exceed_limit":
		// Генерируем запрос, который превышает лимит
		targetTokens = maxTokens + 1000
		message = provider.GenerateTextForTokens(targetTokens, "Это запрос, который превышает максимальный лимит токенов модели. ")
	default:
		result.Error = fmt.Sprintf("Неизвестный тип теста: %s", testType)
		result.Success = false
		return result
	}

	result.Message = message
	result.TokensInput = provider.CountTokens(message)

	startTime := time.Now()

	// Выполняем запрос
	var fullResponse string
	err := p.Chat(ctx, message, &provider.ChatOptions{
		MaxTokens: maxTokens,
	}, func(chunk string) error {
		fullResponse += chunk
		return nil
	})

	durationMs := time.Since(startTime).Milliseconds()
	result.DurationMs = durationMs
	result.Response = fullResponse
	result.TokensOutput = provider.CountTokens(fullResponse)
	result.TokensTotal = result.TokensInput + result.TokensOutput
	result.Cost = p.CalculateCost(result.TokensInput, result.TokensOutput)

	if err != nil {
		result.Error = err.Error()
		result.Success = false
		logger.Error("ошибка при тестировании токенов",
			"test_type", testType,
			"provider", p.Name(),
			"model", p.GetModel(),
			"error", err,
		)
	} else {
		result.Success = true
		logger.Info("тест токенов выполнен",
			"test_type", testType,
			"provider", p.Name(),
			"model", p.GetModel(),
			"tokens_input", result.TokensInput,
			"tokens_output", result.TokensOutput,
			"tokens_total", result.TokensTotal,
		)
	}

	return result
}

// buildSummary строит сводку по результатам тестирования
func (h *TokenTestHandler) buildSummary(results []TokenTestResult) TokenTestSummary {
	summary := TokenTestSummary{
		TotalTests: len(results),
	}

	var totalDuration int64
	for _, r := range results {
		if r.Success {
			summary.SuccessCount++
			totalDuration += r.DurationMs
			summary.TotalTokens += r.TokensTotal
			summary.TotalCost += r.Cost
		} else {
			summary.ErrorCount++
		}
	}

	if summary.SuccessCount > 0 {
		summary.AvgDurationMs = totalDuration / int64(summary.SuccessCount)
	}

	return summary
}
