package gigachat

import (
	"bufio"
	"bytes"
	"crypto/tls"
	"crypto/x509"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
)

// Client представляет клиент для работы с GigaChat API
type Client struct {
	AccessToken string
	APIURL      string
}

// TokenResponse представляет ответ на запрос токена
type TokenResponse struct {
	AccessToken string `json:"access_token"`
	ExpiresAt   int64  `json:"expires_at,omitempty"`
	TokenType   string `json:"token_type,omitempty"`
}

// NewClient создает новый клиент GigaChat
func NewClient(accessToken, apiURL string) *Client {
	return &Client{
		AccessToken: accessToken,
		APIURL:      apiURL,
	}
}

// GetAccessToken получает Access Token используя Authorization Key
func GetAccessToken(authKey, authURL string) (string, error) {
	if authURL == "" {
		authURL = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
	}

	// Тело запроса с scope
	body := strings.NewReader("scope=GIGACHAT_API_PERS")
	
	// URL должен быть без /token в конце
	req, err := http.NewRequest("POST", authURL, body)
	if err != nil {
		return "", fmt.Errorf("ошибка создания запроса: %w", err)
	}

	// Используем Basic auth с Authorization Key (который уже в base64)
	req.Header.Set("Authorization", "Basic "+authKey)
	req.Header.Set("RqUID", generateRqUID())
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	req.Header.Set("Accept", "application/json")

	client := createHTTPClient()
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("ошибка выполнения запроса: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("ошибка получения токена: %d - %s", resp.StatusCode, string(body))
	}

	var tokenResp TokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&tokenResp); err != nil {
		return "", fmt.Errorf("ошибка парсинга ответа: %w", err)
	}

	if tokenResp.AccessToken == "" {
		return "", fmt.Errorf("токен не получен в ответе")
	}

	return tokenResp.AccessToken, nil
}

// generateRqUID генерирует уникальный идентификатор запроса в формате UUID4
func generateRqUID() string {
	return uuid.New().String()
}

// createHTTPClient создает HTTP клиент с настройкой TLS
func createHTTPClient() *http.Client {
	// Пытаемся загрузить системные сертификаты
	caCertPool, err := x509.SystemCertPool()
	if err != nil {
		// Если не удалось загрузить системные сертификаты, создаем пустой пул
		caCertPool = x509.NewCertPool()
	}

	// Пытаемся загрузить сертификат НУЦ Минцифры из стандартных мест
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

	// Настройка TLS конфигурации
	tlsConfig := &tls.Config{
		RootCAs: caCertPool,
	}

	// Если переменная окружения установлена, отключаем проверку сертификата (только для тестирования!)
	if os.Getenv("GIGACHAT_SKIP_TLS_VERIFY") == "true" {
		tlsConfig.InsecureSkipVerify = true
	}

	transport := &http.Transport{
		TLSClientConfig: tlsConfig,
	}

	return &http.Client{
		Transport: transport,
		Timeout:   60 * time.Second,
	}
}

// ChatRequest представляет запрос к API
type ChatRequest struct {
	Model    string    `json:"model"`
	Messages []Message `json:"messages"`
	Stream   bool      `json:"stream"`
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
	Index        int     `json:"index"`
	Delta        *Delta  `json:"delta,omitempty"`
	Message      *Message `json:"message,omitempty"`
	FinishReason string  `json:"finish_reason"`
}

// Delta представляет инкрементальное обновление
type Delta struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// Chat отправляет сообщение в GigaChat API и возвращает streaming ответ
func (c *Client) Chat(message string, onChunk func(string) error) error {
	reqBody := ChatRequest{
		Model: "GigaChat",
		Messages: []Message{
			{
				Role:    "user",
				Content: message,
			},
		},
		Stream: true,
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("ошибка маршалинга запроса: %w", err)
	}

	req, err := http.NewRequest("POST", c.APIURL+"/chat/completions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("ошибка создания запроса: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.AccessToken)

	// Создаем HTTP клиент с настройкой TLS для работы с сертификатом НУЦ Минцифры
	client := createHTTPClient()
	resp, err := client.Do(req)
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

