package config

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// Config представляет конфигурацию приложения
type Config struct {
	GigaChatAccessToken string `yaml:"gigachat_access_token"`
	GigaChatAuthKey     string `yaml:"gigachat_auth_key"`
	Port                string `yaml:"port"`
	GigaChatAPIURL      string `yaml:"gigachat_api_url"`
	GigaChatAuthURL     string `yaml:"gigachat_auth_url"`
}

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

	// Значения по умолчанию
	if cfg.Port == "" {
		cfg.Port = "8080"
	}
	if cfg.GigaChatAPIURL == "" {
		cfg.GigaChatAPIURL = "https://gigachat.devices.sberbank.ru/api/v1"
	}

	if cfg.GigaChatAccessToken == "" && cfg.GigaChatAuthKey == "" {
		return nil, fmt.Errorf("необходимо указать либо gigachat_access_token, либо gigachat_auth_key в конфиге")
	}

	return &cfg, nil
}

