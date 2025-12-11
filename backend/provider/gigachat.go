package provider

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

// GigaChatProvider провайдер для GigaChat API
type GigaChatProvider struct {
	httpClient *http.Client
	apiURL     string
	authURL    string
	authKey    string

	mu          sync.RWMutex
	accessToken string
	expiresAt   time.Time
	model       string
}

// GigaChatConfig конфигурация GigaChat провайдера
type GigaChatConfig struct {
	AuthKey       string
	AccessToken   string // Готовый токен (для тестов)
	APIURL        string
	AuthURL       string
	Model         string
	SkipTLSVerify bool // Пропускать проверку TLS сертификата (для тестирования)
}

// GigaChatModels доступные модели GigaChat
var GigaChatModels = []string{
	"GigaChat",      // Стандартная модель
	"GigaChat-Plus", // Улучшенная модель
	"GigaChat-Pro",  // Профессиональная модель
}

// NewGigaChatProvider создает новый GigaChat провайдер
func NewGigaChatProvider(cfg GigaChatConfig) *GigaChatProvider {
	apiURL := cfg.APIURL
	if apiURL == "" {
		apiURL = "https://gigachat.devices.sberbank.ru/api/v1"
	}

	authURL := cfg.AuthURL
	if authURL == "" {
		authURL = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	}

	model := cfg.Model
	if model == "" {
		model = "GigaChat"
	}

	p := &GigaChatProvider{
		httpClient: createGigaChatHTTPClient(cfg.SkipTLSVerify),
		apiURL:     apiURL,
		authURL:    authURL,
		authKey:    cfg.AuthKey,
		model:      model,
	}

	// Если передан готовый токен
	if cfg.AccessToken != "" {
		p.accessToken = cfg.AccessToken
		p.expiresAt = time.Now().Add(30 * time.Minute)
	}

	return p
}

// Name возвращает имя провайдера
func (p *GigaChatProvider) Name() string {
	return "gigachat"
}

// Models возвращает список доступных моделей
func (p *GigaChatProvider) Models() []string {
	return GigaChatModels
}

// SetModel устанавливает модель
func (p *GigaChatProvider) SetModel(model string) {
	p.model = model
}

// GetModel возвращает текущую модель
func (p *GigaChatProvider) GetModel() string {
	return p.model
}

// GetMaxTokens возвращает максимальный лимит токенов для текущей модели
func (p *GigaChatProvider) GetMaxTokens() int {
	// Лимиты для разных моделей GigaChat
	switch p.model {
	case "GigaChat-Pro":
		return 32768 // Большой контекст
	case "GigaChat-Plus":
		return 8192
	case "GigaChat":
		fallthrough
	default:
		return 4096 // Стандартный лимит
	}
}

// CalculateCost вычисляет стоимость запроса в USD
// GigaChat имеет разные тарифы, здесь используется приблизительная стоимость
func (p *GigaChatProvider) CalculateCost(inputTokens, outputTokens int) float64 {
	// Приблизительные цены для GigaChat (нужно обновить актуальными тарифами)
	// Используем средние значения
	inputPricePer1k := 0.001  // $0.001 за 1000 входных токенов
	outputPricePer1k := 0.002 // $0.002 за 1000 выходных токенов

	inputCost := float64(inputTokens) / 1000.0 * inputPricePer1k
	outputCost := float64(outputTokens) / 1000.0 * outputPricePer1k

	return inputCost + outputCost
}

// getToken возвращает актуальный токен
func (p *GigaChatProvider) getToken(ctx context.Context) (string, error) {
	p.mu.RLock()
	token := p.accessToken
	expires := p.expiresAt
	p.mu.RUnlock()

	if token != "" && time.Now().Add(time.Minute).Before(expires) {
		return token, nil
	}

	return p.refreshToken(ctx)
}

// refreshToken получает новый токен
func (p *GigaChatProvider) refreshToken(ctx context.Context) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.accessToken != "" && time.Now().Add(time.Minute).Before(p.expiresAt) {
		return p.accessToken, nil
	}

	if p.authKey == "" {
		return "", fmt.Errorf("auth_key не задан, невозможно обновить токен")
	}

	body := strings.NewReader("scope=GIGACHAT_API_PERS")
	req, err := http.NewRequestWithContext(ctx, "POST", p.authURL, body)
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Authorization", "Basic "+p.authKey)
	req.Header.Set("RqUID", uuid.New().String())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ошибка получения токена: %d - %s", resp.StatusCode, string(respBody))
	}

	var tokenResp struct {
		AccessToken string `json:"access_token"`
		ExpiresAt   int64  `json:"expires_at,omitempty"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("токен не получен в ответе")
	}

	p.accessToken = tokenResp.AccessToken
	if tokenResp.ExpiresAt > 0 {
		p.expiresAt = time.UnixMilli(tokenResp.ExpiresAt)
	} else {
		p.expiresAt = time.Now().Add(30 * time.Minute)
	}

	return p.accessToken, nil
}

// gigachatChatRequest запрос к GigaChat API
type gigachatChatRequest struct {
	Model       string            `json:"model"`
	Messages    []gigachatMessage `json:"messages"`
	Stream      bool              `json:"stream"`
	MaxTokens   int               `json:"max_tokens,omitempty"`
	Temperature float64           `json:"temperature,omitempty"`
}

type gigachatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type gigachatChatResponse struct {
	ID      string           `json:"id"`
	Choices []gigachatChoice `json:"choices"`
}

type gigachatChoice struct {
	Index        int            `json:"index"`
	Delta        *gigachatDelta `json:"delta,omitempty"`
	Message      *gigachatDelta `json:"message,omitempty"`
	FinishReason string         `json:"finish_reason"`
}

type gigachatDelta struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Chat отправляет сообщение через GigaChat API
func (p *GigaChatProvider) Chat(ctx context.Context, message string, opts *ChatOptions, onChunk func(string) error) error {
	token, err := p.getToken(ctx)
	if err != nil {
		return fmt.Errorf("ошибка получения токена: %w", err)
	}

	messages := []gigachatMessage{}

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
		messages = append(messages, gigachatMessage{
			Role:    "system",
			Content: systemPrompt,
		})
	}

	// История
	if opts != nil && len(opts.History) > 0 {
		for _, msg := range opts.History {
			messages = append(messages, gigachatMessage{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}
	}

	// Текущее сообщение
	messages = append(messages, gigachatMessage{
		Role:    "user",
		Content: message,
	})

	reqBody := gigachatChatRequest{
		Model:    p.model,
		Messages: messages,
		Stream:   true,
	}

	if opts != nil {
		if opts.MaxTokens > 0 {
			reqBody.MaxTokens = opts.MaxTokens
		}
		if opts.Temperature >= 0 {
			reqBody.Temperature = opts.Temperature
		}
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга запроса: %w", err)
	}

	// Debug: логируем запрос
	fmt.Printf("[GigaChat] Request: %s\n", string(jsonData))

	req, err := http.NewRequestWithContext(ctx, "POST", p.apiURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := p.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("ошибка GigaChat API: %d - %s", resp.StatusCode, string(body))
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

		var chatResp gigachatChatResponse
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

// createGigaChatHTTPClient создает HTTP клиент с настройкой TLS для GigaChat
func createGigaChatHTTPClient(skipTLSVerify bool) *http.Client {
	// Проверяем переменную окружения (приоритет) или параметр конфига
	skipVerify := skipTLSVerify || os.Getenv("GIGACHAT_SKIP_TLS_VERIFY") == "true"

	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		caCertPool = x509.NewCertPool()
	}

	// Если не пропускаем проверку, пытаемся загрузить российские сертификаты
	if !skipVerify {
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
	}

	tlsConfig := &tls.Config{
		RootCAs:            caCertPool,
		InsecureSkipVerify: skipVerify,
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
