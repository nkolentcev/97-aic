# Архитектура проекта

## Обзор

Проект реализован как full-stack приложение с разделением на backend (Go) и frontend (Svelte), объединенные в единый бинарный файл для упрощения развертывания.

## Архитектура агентов

Проект организован с использованием системы координации специализированных ИИ-агентов согласно `ai-template.md`:

### `<agent_go>` - Go Backend Engineer

**Ответственность:**
- Разработка HTTP API сервера
- Интеграция с GigaChat API
- Обработка streaming ответов
- Бизнес-логика приложения
- Раздача статического контента (встроенный frontend)

**Реализованные компоненты:**
- `main.go` - точка входа, настройка HTTP сервера
- `api/chat.go` - обработчик API endpoint `/api/chat`
- `gigachat/client.go` - клиент для работы с GigaChat API
- `config/config.go` - загрузка и валидация конфигурации

### `<agent_svelte>` - Svelte Frontend Engineer

**Ответственность:**
- Разработка пользовательского интерфейса
- Реализация чата в стиле Google AI
- Обработка streaming ответов на клиенте
- Интеграция с backend API

**Реализованные компоненты:**
- `App.svelte` - главный компонент приложения
- `lib/ChatMessage.svelte` - компонент отображения сообщения
- `lib/ChatInput.svelte` - компонент ввода сообщения
- `lib/api.ts` - клиент для работы с backend API

### `<agent_devops>` - DevOps/Gitea CI Expert

**Ответственность:**
- Настройка CI/CD пайплайнов
- Автоматизация сборки и деплоя
- Конфигурация развертывания на VPS

**Реализованные компоненты:**
- `.gitea/workflows/deploy.yml` - Gitea Actions workflow
- `deploy/deploy.sh` - скрипт ручного деплоя
- `backend/run-dev.sh` - скрипт для разработки

## Структура данных

### Backend

```
backend/
├── main.go                 # Точка входа
├── api/
│   └── chat.go            # API обработчик
├── config/
│   └── config.go          # Конфигурация
├── gigachat/
│   └── client.go          # GigaChat API клиент
├── static/                # Встроенный frontend (заполняется при сборке)
└── config.yaml            # Конфигурационный файл
```

### Frontend

```
frontend/
├── src/
│   ├── App.svelte         # Главный компонент
│   ├── lib/
│   │   ├── api.ts         # API клиент
│   │   ├── ChatMessage.svelte
│   │   └── ChatInput.svelte
│   └── main.ts            # Точка входа
└── dist/                  # Production build
```

## Поток данных

1. **Пользователь отправляет сообщение:**
   - Frontend: `ChatInput` → `App.svelte` → `api.ts`
   - HTTP POST `/api/chat` с `{"message": "текст"}`

2. **Backend обрабатывает запрос:**
   - `api/chat.go` парсит запрос
   - `gigachat/client.go` отправляет запрос в GigaChat API
   - Получает streaming ответ

3. **Streaming ответ:**
   - Backend отправляет SSE события: `data: {"content": "часть ответа"}`
   - Frontend читает stream через `ReadableStream`
   - Обновляет UI в реальном времени

4. **Завершение:**
   - Backend отправляет `data: [DONE]`
   - Frontend завершает обработку

## Безопасность

- Конфигурация с секретами хранится в `config.yaml` (не коммитится)
- TLS настройка для работы с GigaChat API
- CORS настроен для разработки
- Валидация входных данных на backend

## Масштабирование

Текущая архитектура поддерживает:
- Один экземпляр backend с встроенным frontend
- Stateless API (можно масштабировать горизонтально)
- Streaming ответы через SSE

Возможные улучшения:
- Добавление базы данных для истории сообщений
- Кэширование токенов доступа
- Rate limiting
- Load balancing для нескольких экземпляров

