# Руководство для разработчиков

## Настройка окружения разработки

### Backend

1. Установите Go 1.21 или выше
2. Перейдите в директорию backend:
```bash
cd backend
```

3. Установите зависимости:
```bash
go mod download
```

4. Создайте конфигурационный файл:
```bash
cp config.example.yaml config.yaml
# Отредактируйте config.yaml
```

5. Запустите сервер:
```bash
go run .
```

### Frontend

1. Установите Node.js 20 или выше
2. Перейдите в директорию frontend:
```bash
cd frontend
```

3. Установите зависимости:
```bash
npm install
```

4. Запустите dev-сервер:
```bash
npm run dev
```

Dev-сервер будет доступен на `http://localhost:5173` и автоматически проксирует запросы к `/api` на backend.

## Структура кода

### Backend

- `main.go` - точка входа, настройка HTTP сервера
- `config/config.go` - загрузка и валидация конфигурации
- `api/chat.go` - HTTP обработчик для `/api/chat`
- `gigachat/client.go` - клиент для работы с GigaChat API

### Frontend

- `src/App.svelte` - главный компонент приложения
- `src/lib/ChatMessage.svelte` - компонент отображения сообщения
- `src/lib/ChatInput.svelte` - компонент ввода сообщения
- `src/lib/api.ts` - клиент для работы с backend API

## Тестирование

### Backend

```bash
cd backend
go test ./...
```

### Frontend

```bash
cd frontend
npm run check  # Проверка типов TypeScript
```

## Стиль кода

### Go

- Используйте `gofmt` для форматирования
- Следуйте [Effective Go](https://go.dev/doc/effective_go)
- Обрабатывайте все ошибки явно
- Комментарии к экспортируемым функциям обязательны

### TypeScript/Svelte

- Используйте строгий режим TypeScript (`strict: true`)
- Компоненты в PascalCase, файлы в kebab-case
- Форматирование через Prettier (опционально)

## Отладка

### Backend

Логи выводятся в stdout/stderr. Для production используйте systemd journal:
```bash
sudo journalctl -u gigachat-chat -f
```

### Frontend

Используйте DevTools браузера. В dev-режиме доступен hot-reload.

## Добавление новых функций

1. Создайте feature ветку:
```bash
git checkout -b feature/new-feature
```

2. Внесите изменения и протестируйте локально

3. Создайте Pull Request в Gitea

4. После одобрения изменения будут автоматически задеплоены через CI/CD

## Работа с API GigaChat

Клиент находится в `backend/gigachat/client.go`. Для изменения параметров запроса (модель, температура и т.д.) отредактируйте структуру `ChatRequest` в этом файле.

Для получения access token зарегистрируйтесь на [портале разработчиков Сбера](https://developers.sber.ru/docs/ru/gigachat/api/overview) и создайте проект.

## Проблемы и решения

### Backend не запускается

- Проверьте наличие `config.yaml`
- Убедитесь, что порт не занят другим процессом
- Проверьте валидность access token GigaChat

### Frontend не подключается к backend

- Убедитесь, что backend запущен
- Проверьте настройки proxy в `vite.config.js`
- Проверьте CORS настройки в backend (если нужно)

### Streaming не работает

- Проверьте поддержку SSE в браузере
- Убедитесь, что backend правильно отправляет события
- Проверьте логи backend на наличие ошибок

