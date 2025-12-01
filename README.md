# GigaChat Chat Application

Full-stack приложение для работы с GigaChat API от Сбера в формате чата. Backend на Go со встроенным Svelte frontend.

## Описание проекта

Приложение предоставляет веб-интерфейс для общения с GigaChat AI через API. Интерфейс выполнен в стиле Google AI с поддержкой streaming ответов.

**Идея проекта:** Реализовать бэкенд на языке Go со встроенным фронтендом на Svelte, для работы с API GigaChat от Сбера в формате запрос/ответ. Пользователь в формате чата пишет вопрос и читает ответ - визуально как Google AI. Проект автоматизированно собирается и разворачивается на удаленном сервере.

## Технологический стек

- **Backend/API:** Go 1.21+
- **Frontend (Web):** Svelte + TypeScript + Vite
- **API:** [GigaChat API](https://developers.sber.ru/docs/ru/gigachat/api/overview)
- **DevOps/CI/CD:** Gitea Actions
- **Документация:** Doc-as-code в формате Markdown

## Архитектура агентов

Проект организован с использованием системы координации специализированных ИИ-агентов:

- **`<agent_go>`** - Go Backend Engineer: разработка API, бизнес-логика, интеграция с GigaChat API
- **`<agent_svelte>`** - Svelte Frontend Engineer: разработка веб-интерфейса
- **`<agent_devops>`** - DevOps/Gitea CI Expert: настройка CI/CD, деплой

## Структура проекта

```
/backend     - Go API сервер
/frontend    - Svelte приложение
/deploy      - конфигурации для деплоя
/.gitea      - CI/CD workflows
/docs        - документация (Doc-as-code)
```

## Быстрый старт

### Требования

- Go 1.21 или выше
- Node.js 20 или выше
- npm или yarn

### Установка

1. Клонируйте репозиторий:
```bash
git clone <repository-url>
cd 97-aic
```

2. Настройте backend:
```bash
cd backend
cp config.example.yaml config.yaml
# Отредактируйте config.yaml и укажите Authorization Key из giga-ai.md
```

3. Установите зависимости frontend:
```bash
cd ../frontend
npm install
```

### Запуск для разработки

**Backend:**
```bash
cd backend
./run-dev.sh  # или GIGACHAT_SKIP_TLS_VERIFY=true go run .
```

**Frontend:**
```bash
cd frontend
npm run dev
```

Откройте http://localhost:5173 в браузере

### Production сборка

1. Соберите frontend:
```bash
cd frontend
npm run build
```

2. Скопируйте собранный frontend в backend:
```bash
mkdir -p backend/static
cp -r frontend/dist/* backend/static/
```

3. Соберите backend:
```bash
cd backend
go build -o server .
```

4. Запустите сервер:
```bash
./server
```

## Документация

- [README](docs/README.md) - общее описание проекта, установка, запуск
- [API документация](docs/API.md) - описание API endpoints
- [Архитектура](docs/ARCHITECTURE.md) - архитектура проекта и поток данных
- [Руководство по развертыванию](docs/DEPLOYMENT.md) - инструкции по развертыванию
- [Руководство для разработчиков](docs/DEVELOPMENT.md) - руководство для разработчиков
- [Настройка TLS](docs/TLS_SETUP.md) - настройка сертификатов для GigaChat API
- [Соло-разработка с агентами](docs/SOLO_DEVELOPMENT.md) - современный подход к разработке с Claude Code и Cursor

## Конфигурация

Создайте файл `backend/config.yaml` на основе `backend/config.example.yaml`:

```yaml
gigachat_auth_key: "your-authorization-key-here"
port: "8080"
gigachat_api_url: "https://gigachat.devices.sberbank.ru/api/v1"
gigachat_auth_url: "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
```

## Команды разработки

### Backend (Go)

```bash
cd backend
go run .              # запуск
go test ./...         # тесты
go build -o server .  # сборка
./run-dev.sh          # запуск с отключенной проверкой TLS (для разработки)
```

### Frontend (Svelte)

```bash
cd frontend
npm install           # установка зависимостей
npm run dev           # dev-сервер
npm run build         # production сборка
npm run check         # проверка типов
```

## CI/CD

Пайплайн Gitea Actions автоматически:
1. Собирает frontend
2. Копирует в backend/static
3. Собирает Go backend
4. Деплоит на VPS сервер

Подробнее см. [DEPLOYMENT.md](docs/DEPLOYMENT.md)

## Стиль кода

### Go
- Использовать `gofmt` для форматирования
- Следовать [Effective Go](https://go.dev/doc/effective_go)
- Обрабатывать все ошибки явно
- Комментарии к экспортируемым функциям обязательны

### Svelte/TypeScript
- Использовать TypeScript строго (`strict: true`)
- Компоненты в PascalCase, файлы в kebab-case

## Git workflow

- Коммиты на русском языке
- Формат: `тип: краткое описание`
  - `feat:` — новая функциональность
  - `fix:` — исправление бага
  - `docs:` — документация
  - `refactor:` — рефакторинг
  - `test:` — тесты
  - `ci:` — CI/CD
- Ветки: `feature/название`, `fix/название`

## Лицензия

MIT

