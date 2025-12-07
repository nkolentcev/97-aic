package provider

import (
	"fmt"
	"sync"
)

// Manager управляет провайдерами AI
type Manager struct {
	mu              sync.RWMutex
	providers       map[string]Provider
	defaultProvider string
}

// NewManager создает новый менеджер провайдеров
func NewManager() *Manager {
	return &Manager{
		providers: make(map[string]Provider),
	}
}

// Register регистрирует провайдера
func (m *Manager) Register(name string, p Provider) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.providers[name] = p
}

// SetDefault устанавливает провайдера по умолчанию
func (m *Manager) SetDefault(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	if _, ok := m.providers[name]; !ok {
		return fmt.Errorf("провайдер %s не зарегистрирован", name)
	}
	m.defaultProvider = name
	return nil
}

// Get возвращает провайдера по имени
func (m *Manager) Get(name string) (Provider, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if name == "" {
		name = m.defaultProvider
	}

	p, ok := m.providers[name]
	if !ok {
		return nil, fmt.Errorf("провайдер %s не найден", name)
	}
	return p, nil
}

// GetDefault возвращает провайдера по умолчанию
func (m *Manager) GetDefault() (Provider, error) {
	return m.Get(m.defaultProvider)
}

// List возвращает список зарегистрированных провайдеров
func (m *Manager) List() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	names := make([]string, 0, len(m.providers))
	for name := range m.providers {
		names = append(names, name)
	}
	return names
}

// GetDefaultName возвращает имя провайдера по умолчанию
func (m *Manager) GetDefaultName() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.defaultProvider
}

// ProviderInfo информация о провайдере
type ProviderInfo struct {
	Name         string   `json:"name"`
	Models       []string `json:"models"`
	CurrentModel string   `json:"current_model"`
	IsDefault    bool     `json:"is_default"`
}

// ListInfo возвращает информацию о всех провайдерах
func (m *Manager) ListInfo() []ProviderInfo {
	m.mu.RLock()
	defer m.mu.RUnlock()

	infos := make([]ProviderInfo, 0, len(m.providers))
	for name, p := range m.providers {
		infos = append(infos, ProviderInfo{
			Name:         name,
			Models:       p.Models(),
			CurrentModel: p.GetModel(),
			IsDefault:    name == m.defaultProvider,
		})
	}
	return infos
}
