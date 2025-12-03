package gigachat

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Client представляет клиент для работы с GigaChat API
type Client struct {
	httpClient *http.Client
	apiURL     string
	authURL    string
	authKey    string

	mu          sync.RWMutex
	accessToken string
	expiresAt   time.Time
}

// TokenResponse представляет ответ на запрос токена
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
}

// JSONConfig настройки для JSON-ответа
type JSONConfig struct {
	Enabled    bool   `json:"enabled"`
	SchemaText string `json:"schema_text,omitempty"` // Текст структуры из текстового поля
}

// CollectConfig настройки для режима сбора требований
type CollectConfig struct {
	Enabled           bool     `json:"enabled"`
	Role              string   `json:"role,omitempty"`               // Роль модели (например, "технический аналитик")
	Goal              string   `json:"goal,omitempty"`               // Цель сбора (например, "ТЗ на мобильное приложение")
	RequiredQuestions []string `json:"required_questions,omitempty"` // Список обязательных вопросов
	OutputFormat      string   `json:"output_format,omitempty"`      // Формат финального результата
}

// ChatOptions расширенные параметры запроса к API
type ChatOptions struct {
	SystemPrompt  string         `json:"system_prompt,omitempty"`
	History       []Message      `json:"history,omitempty"`
	JSONConfig    *JSONConfig    `json:"json_config,omitempty"`
	CollectConfig *CollectConfig `json:"collect_config,omitempty"`
	MaxTokens     int            `json:"max_tokens,omitempty"`
	Temperature   float64        `json:"temperature,omitempty"`
}

// NewClient создает новый клиент GigaChat
func NewClient(authKey, apiURL, authURL string) *Client {
	return &Client{
		httpClient: createHTTPClient(),
		apiURL:     apiURL,
		authURL:    authURL,
		authKey:    authKey,
	}
}

// NewClientWithToken создает клиент с готовым токеном (для тестов)
func NewClientWithToken(accessToken, apiURL string) *Client {
	return &Client{
		httpClient:  createHTTPClient(),
		apiURL:      apiURL,
		accessToken: accessToken,
		expiresAt:   time.Now().Add(30 * time.Minute),
	}
}

// getToken возвращает актуальный токен, обновляя при необходимости
func (c *Client) getToken(ctx context.Context) (string, error) {
	c.mu.RLock()
	token := c.accessToken
	expires := c.expiresAt
	c.mu.RUnlock()

	// Токен валиден еще минимум 1 минуту
	if token != "" && time.Now().Add(time.Minute).Before(expires) {
		return token, nil
	}

	// Нужно обновить токен
	return c.refreshToken(ctx)
}

// refreshToken получает новый токен
func (c *Client) refreshToken(ctx context.Context) (string, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Проверяем еще раз под блокировкой
	if c.accessToken != "" && time.Now().Add(time.Minute).Before(c.expiresAt) {
		return c.accessToken, nil
	}

	if c.authKey == "" {
		return "", fmt.Errorf("auth_key не задан, невозможно обновить токен")
	}

	authURL := c.authURL
	if authURL == "" {
		authURL = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	}

	body := strings.NewReader("scope=GIGACHAT_API_PERS")
	req, err := http.NewRequestWithContext(ctx, "POST", authURL, body)
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Authorization", "Basic "+c.authKey)
	req.Header.Set("RqUID", uuid.New().String())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ошибка получения токена: %d - %s", resp.StatusCode, string(respBody))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("токен не получен в ответе")
	}

	c.accessToken = tokenResp.AccessToken
	if tokenResp.ExpiresAt > 0 {
		c.expiresAt = time.UnixMilli(tokenResp.ExpiresAt)
	} else {
		c.expiresAt = time.Now().Add(30 * time.Minute)
	}

	return c.accessToken, nil
}

// createHTTPClient создает HTTP клиент с настройкой TLS
func createHTTPClient() *http.Client {
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		caCertPool = x509.NewCertPool()
	}

	certPaths := []string{
		"/etc/ssl/certs/russian_trusted_root_ca.pem",
		"/etc/ssl/certs/ca-certificates.crt",
		"/usr/local/share/ca-certificates/russian_trusted_root_ca.crt",
		os.Getenv("HOME") + "/.local/share/ca-certificates/russian_trusted_root_ca.crt",
	}

	for _, certPath := range certPaths {
		if certData, err := os.ReadFile(certPath); err == nil {
			if caCertPool.AppendCertsFromPEM(certData) {
				break
			}
		}
	}

	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	if os.Getenv("GIGACHAT_SKIP_TLS_VERIFY") == "true" {
		tlsConfig.InsecureSkipVerify = true
	}

	transport := &http.Transport{
		TLSClientConfig:     tlsConfig,
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}
}

// ChatRequest представляет запрос к API
type ChatRequest struct {
	Model       string    `json:"model"`
	Messages    []Message `json:"messages"`
	Stream      bool      `json:"stream"`
	MaxTokens   int       `json:"max_tokens,omitempty"`
	Temperature float64   `json:"temperature,omitempty"`
}

// Message представляет сообщение в чате
type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ChatResponse представляет ответ от API
type ChatResponse struct {
	ID      string   `json:"id"`
	Object  string   `json:"object"`
	Created int64    `json:"created"`
	Model   string   `json:"model"`
	Choices []Choice `json:"choices"`
}

// Choice представляет выбор в ответе
type Choice struct {
	Index        int      `json:"index"`
	Delta        *Delta   `json:"delta,omitempty"`
	Message      *Message `json:"message,omitempty"`
	FinishReason string   `json:"finish_reason"`
}

// Delta представляет инкрементальное обновление
type Delta struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// buildJSONSystemPrompt создает system prompt для форсирования JSON-формата
func buildJSONSystemPrompt(config *JSONConfig) string {
	prompt := "Ваш ответ должен быть строго в формате JSON.\n"

	if config.SchemaText != "" {
		prompt += fmt.Sprintf("Структура ответа должна соответствовать следующей схеме:\n%s\n", config.SchemaText)
	}

	prompt += "\nВАЖНО: Отвечай ТОЛЬКО валидным JSON без дополнительных пояснений, комментариев или markdown-разметки!"

	return prompt
}

// buildCollectSystemPrompt создает system prompt для режима сбора требований
func buildCollectSystemPrompt(config *CollectConfig) string {
	var sb strings.Builder

	// Роль модели
	role := config.Role
	if role == "" {
		role = "профессиональный аналитик"
	}
	sb.WriteString(fmt.Sprintf("Ты — %s.\n\n", role))

	// Цель сбора
	goal := config.Goal
	if goal == "" {
		goal = "техническое задание"
	}
	sb.WriteString(fmt.Sprintf("Твоя задача — через диалог с пользователем собрать всю необходимую информацию для составления: %s.\n\n", goal))

	// Инструкции по работе
	sb.WriteString("ПРАВИЛА РАБОТЫ:\n")
	sb.WriteString("1. Задавай вопросы ПО ОДНОМУ, жди ответа пользователя перед следующим вопросом.\n")
	sb.WriteString("2. Уточняй детали, если ответ пользователя неполный или неясный.\n")
	sb.WriteString("3. Не переходи к следующему вопросу, пока не получишь достаточно информации по текущему.\n\n")

	// Обязательные вопросы
	if len(config.RequiredQuestions) > 0 {
		sb.WriteString("ОБЯЗАТЕЛЬНЫЕ ВОПРОСЫ (задай все по очереди):\n")
		for i, q := range config.RequiredQuestions {
			sb.WriteString(fmt.Sprintf("%d. %s\n", i+1, q))
		}
		sb.WriteString("\n")
	}

	// Формат результата
	sb.WriteString("ФОРМАТ ОТВЕТА:\n")
	sb.WriteString("Пока собираешь информацию, отвечай в формате JSON:\n")
	sb.WriteString(`{"status": "collecting", "question": "Твой следующий вопрос", "collected": ["список уже полученных данных"]}`)
	sb.WriteString("\n\n")

	sb.WriteString("Когда ВСЕ необходимые вопросы заданы и получены ответы, выдай финальный результат:\n")

	outputFormat := config.OutputFormat
	if outputFormat == "" {
		outputFormat = "структурированный документ с разделами"
	}
	sb.WriteString(fmt.Sprintf(`{"status": "ready", "result": "%s в виде текста"}`, outputFormat))
	sb.WriteString("\n\n")

	sb.WriteString("ВАЖНО: Отвечай ТОЛЬКО валидным JSON без дополнительных пояснений!")

	return sb.String()
}

// Chat отправляет сообщение в GigaChat API и возвращает streaming ответ
func (c *Client) Chat(ctx context.Context, message string, onChunk func(string) error) error {
	return c.ChatWithJSON(ctx, message, nil, onChunk)
}

// ChatWithJSON отправляет сообщение с поддержкой JSON-формата ответа
func (c *Client) ChatWithJSON(ctx context.Context, message string, jsonConfig *JSONConfig, onChunk func(string) error) error {
	opts := &ChatOptions{
		JSONConfig: jsonConfig,
	}
	return c.ChatWithOptions(ctx, message, opts, onChunk)
}

// ChatWithHistory отправляет сообщение с историей предыдущих сообщений
func (c *Client) ChatWithHistory(ctx context.Context, message string, history []Message, opts *ChatOptions, onChunk func(string) error) error {
	if opts == nil {
		opts = &ChatOptions{}
	}
	opts.History = history
	return c.ChatWithOptions(ctx, message, opts, onChunk)
}

// ChatWithOptions отправляет сообщение с расширенными параметрами
func (c *Client) ChatWithOptions(ctx context.Context, message string, opts *ChatOptions, onChunk func(string) error) error {
	token, err := c.getToken(ctx)
	if err != nil {
		return fmt.Errorf("ошибка получения токена: %w", err)
	}

	messages := []Message{}

	// Добавляем system prompt
	var systemPrompt string
	if opts != nil {
		// Приоритет: CollectConfig > JSONConfig > SystemPrompt
		if opts.CollectConfig != nil && opts.CollectConfig.Enabled {
			systemPrompt = buildCollectSystemPrompt(opts.CollectConfig)
		} else if opts.JSONConfig != nil && opts.JSONConfig.Enabled {
			systemPrompt = buildJSONSystemPrompt(opts.JSONConfig)
		} else if opts.SystemPrompt != "" {
			systemPrompt = opts.SystemPrompt
		}
	}

	if systemPrompt != "" {
		messages = append(messages, Message{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// Добавляем историю сообщений
	if opts != nil && len(opts.History) > 0 {
		messages = append(messages, opts.History...)
	}

	// Добавляем текущее сообщение пользователя
	messages = append(messages, Message{
		Role:    "user",
		Content: message,
	})

	reqBody := ChatRequest{
		Model:    "GigaChat",
		Messages: messages,
		Stream:   true,
	}

	// Добавляем опциональные параметры
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

	req, err := http.NewRequestWithContext(ctx, "POST", c.apiURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ошибка API: %d - %s", resp.StatusCode, string(body))
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

		var chatResp ChatResponse
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

// CollectStatus представляет статус сбора требований
type CollectStatus struct {
	Status    string   `json:"status"`              // "collecting" или "ready"
	Question  string   `json:"question,omitempty"`  // Следующий вопрос (если collecting)
	Collected []string `json:"collected,omitempty"` // Собранные данные
	Result    string   `json:"result,omitempty"`    // Финальный результат (если ready)
}

// ParseCollectResponse парсит JSON-ответ режима сбора требований
func ParseCollectResponse(response string) (*CollectStatus, error) {
	// Пытаемся найти JSON в ответе
	response = strings.TrimSpace(response)

	// Убираем возможные markdown-обертки
	if strings.HasPrefix(response, "```json") {
		response = strings.TrimPrefix(response, "```json")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	} else if strings.HasPrefix(response, "```") {
		response = strings.TrimPrefix(response, "```")
		response = strings.TrimSuffix(response, "```")
		response = strings.TrimSpace(response)
	}

	var status CollectStatus
	if err := json.Unmarshal([]byte(response), &status); err != nil {
		return nil, fmt.Errorf("ошибка парсинга JSON: %w", err)
	}

	return &status, nil
}
