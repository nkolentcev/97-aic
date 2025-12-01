# Правила репозитория

## Описание проекта

Учебный full-stack проект: backend на Go, frontend на Svelte.
CI/CD: Gitea Actions.

## Структура проекта

```
/backend     - Go API сервер
/frontend    - Svelte приложение
/deploy      - конфигурации для деплоя
/.gitea      - CI/CD workflows
```

## Команды

### Backend (Go)

```bash
cd backend
go run .              # запуск
go test ./...         # тесты
go build -o server .  # сборка
```

### Frontend (Svelte)

```bash
cd frontend
npm install           # установка зависимостей
npm run dev           # dev-сервер
npm run build         # production сборка
npm run check         # проверка типов
```

## Стиль кода

### Go

- Использовать `gofmt` для форматирования
- Следовать [Effective Go](https://go.dev/doc/effective_go)
- Обрабатывать все ошибки явно (никаких `_` для ошибок)
- Комментарии к экспортируемым функциям обязательны

### Svelte/TypeScript

- Использовать TypeScript строго (`strict: true`)
- Форматирование через Prettier
- Компоненты в PascalCase, файлы в kebab-case

## Git

- Коммиты на русском языке
- Формат: `тип: краткое описание`
  - `feat:` — новая функциональность
  - `fix:` — исправление бага
  - `docs:` — документация
  - `refactor:` — рефакторинг
  - `test:` — тесты
  - `ci:` — CI/CD
- Ветки: `feature/название`, `fix/название`

## CI/CD (Gitea Actions)

Пайплайн запускается на каждый push:
1. Линтинг и тесты backend
2. Сборка и проверка frontend
3. Деплой на staging (ветка main)

## Важно

- Не коммитить секреты и .env файлы
- Перед коммитом запускать тесты локально
- PR требует минимум 1 approve
