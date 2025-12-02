package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config представляет конфигурацию приложения
type Config struct {
	// GigaChat API
	GigaChatAccessToken string `yaml:"gigachat_access_token"`
	GigaChatAuthKey     string `yaml:"gigachat_auth_key"`
	GigaChatAPIURL      string `yaml:"gigachat_api_url"`
	GigaChatAuthURL     string `yaml:"gigachat_auth_url"`

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
	if c.GigaChatAccessToken == "" && c.GigaChatAuthKey == "" {
		return fmt.Errorf("необходимо указать либо gigachat_access_token, либо gigachat_auth_key в конфиге")
	}
	return nil
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
