package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// ProviderConfig конфигурация AI-провайдера
type ProviderConfig struct {
	Enabled bool   `yaml:"enabled"`
	APIKey  string `yaml:"api_key,omitempty"`
	APIURL  string `yaml:"api_url,omitempty"`
	AuthURL string `yaml:"auth_url,omitempty"`
	Model   string `yaml:"model,omitempty"`
}

// Config представляет конфигурацию приложения
type Config struct {
	// Активный провайдер по умолчанию
	DefaultProvider string `yaml:"default_provider"` // gigachat, groq, ollama

	// GigaChat API (legacy + новый формат)
	GigaChatAccessToken string `yaml:"gigachat_access_token"`
	GigaChatAuthKey     string `yaml:"gigachat_auth_key"`
	GigaChatAPIURL      string `yaml:"gigachat_api_url"`
	GigaChatAuthURL     string `yaml:"gigachat_auth_url"`

	// Провайдеры (новый формат)
	Providers struct {
		GigaChat ProviderConfig `yaml:"gigachat"`
		Groq     ProviderConfig `yaml:"groq"`
		Ollama   ProviderConfig `yaml:"ollama"`
	} `yaml:"providers"`

	// Сервер
	Port string `yaml:"port"`

	// Логирование
	LogLevel  string `yaml:"log_level"`  // debug, info, warn, error
	LogFormat string `yaml:"log_format"` // text, json

	// Хранилище
	DatabasePath string `yaml:"database_path"`

	// Лимиты
	MaxRequestBodySize int `yaml:"max_request_body_size"` // в байтах
	MaxQueryLimit      int `yaml:"max_query_limit"`       // максимальный limit для запросов
	DefaultQueryLimit  int `yaml:"default_query_limit"`   // дефолтный limit

	// CORS
	CORSAllowedOrigins []string `yaml:"cors_allowed_origins"`
}

// Константы по умолчанию
const (
	DefaultPort               = "8080"
	DefaultLogLevel           = "info"
	DefaultLogFormat          = "text"
	DefaultDatabasePath       = "data.db"
	DefaultMaxRequestBodySize = 1 << 20 // 1 MB
	DefaultMaxQueryLimit      = 1000
	DefaultQueryLimit         = 100
	DefaultGigaChatAPIURL     = "https://gigachat.devices.sberbank.ru/api/v1"
	DefaultGigaChatAuthURL    = "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
)

// Load загружает конфигурацию из файла
func Load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("ошибка чтения конфига: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("ошибка парсинга конфига: %w", err)
	}

	cfg.applyDefaults()

	if err := cfg.validate(); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// applyDefaults применяет значения по умолчанию
func (c *Config) applyDefaults() {
	if c.Port == "" {
		c.Port = DefaultPort
	}
	if c.GigaChatAPIURL == "" {
		c.GigaChatAPIURL = DefaultGigaChatAPIURL
	}
	if c.GigaChatAuthURL == "" {
		c.GigaChatAuthURL = DefaultGigaChatAuthURL
	}
	if c.LogLevel == "" {
		c.LogLevel = DefaultLogLevel
	}
	if c.LogFormat == "" {
		c.LogFormat = DefaultLogFormat
	}
	if c.DatabasePath == "" {
		c.DatabasePath = DefaultDatabasePath
	}
	if c.MaxRequestBodySize <= 0 {
		c.MaxRequestBodySize = DefaultMaxRequestBodySize
	}
	if c.MaxQueryLimit <= 0 {
		c.MaxQueryLimit = DefaultMaxQueryLimit
	}
	if c.DefaultQueryLimit <= 0 {
		c.DefaultQueryLimit = DefaultQueryLimit
	}
	if len(c.CORSAllowedOrigins) == 0 {
		c.CORSAllowedOrigins = []string{"*"}
	}
}

// validate проверяет конфигурацию
func (c *Config) validate() error {
	// Проверяем, что хотя бы один провайдер настроен
	hasProvider := false

	// Legacy GigaChat config
	if c.GigaChatAccessToken != "" || c.GigaChatAuthKey != "" {
		hasProvider = true
	}

	// New providers config
	if c.Providers.GigaChat.Enabled && c.Providers.GigaChat.APIKey != "" {
		hasProvider = true
	}
	if c.Providers.Groq.Enabled && c.Providers.Groq.APIKey != "" {
		hasProvider = true
	}
	if c.Providers.Ollama.Enabled {
		hasProvider = true // Ollama не требует API key
	}

	if !hasProvider {
		return fmt.Errorf("необходимо настроить хотя бы один AI-провайдер в конфиге")
	}
	return nil
}

// GetDefaultProvider возвращает провайдер по умолчанию
func (c *Config) GetDefaultProvider() string {
	if c.DefaultProvider != "" {
		return c.DefaultProvider
	}
	// Автоопределение
	if c.Providers.Groq.Enabled && c.Providers.Groq.APIKey != "" {
		return "groq"
	}
	if c.Providers.Ollama.Enabled {
		return "ollama"
	}
	if c.GigaChatAccessToken != "" || c.GigaChatAuthKey != "" || c.Providers.GigaChat.Enabled {
		return "gigachat"
	}
	return "gigachat"
}

// GetEnabledProviders возвращает список включенных провайдеров
func (c *Config) GetEnabledProviders() []string {
	providers := []string{}
	if c.GigaChatAccessToken != "" || c.GigaChatAuthKey != "" || c.Providers.GigaChat.Enabled {
		providers = append(providers, "gigachat")
	}
	if c.Providers.Groq.Enabled && c.Providers.Groq.APIKey != "" {
		providers = append(providers, "groq")
	}
	if c.Providers.Ollama.Enabled {
		providers = append(providers, "ollama")
	}
	return providers
}

// IsCORSAllowed проверяет, разрешен ли origin
func (c *Config) IsCORSAllowed(origin string) bool {
	for _, allowed := range c.CORSAllowedOrigins {
		if allowed == "*" || allowed == origin {
			return true
		}
	}
	return false
}
