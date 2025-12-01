# GigaChat Chat

Full-stack приложение для работы с GigaChat API от Сбера в формате чата. Backend на Go со встроенным Svelte frontend.

## Описание

Приложение предоставляет веб-интерфейс для общения с GigaChat AI через API. Интерфейс выполнен в стиле Google AI с поддержкой streaming ответов.

## Технологии

- **Backend**: Go 1.21+
- **Frontend**: Svelte + TypeScript + Vite
- **API**: [GigaChat API](https://developers.sber.ru/docs/ru/gigachat/api/overview)
- **CI/CD**: Gitea Actions

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
# Отредактируйте config.yaml и укажите:
# - либо gigachat_access_token (если токен уже получен)
# - либо gigachat_auth_key (Authorization Key для автоматического получения токена)
# Для получения ключей зарегистрируйтесь на https://developers.sber.ru
```

3. Установите зависимости frontend:
```bash
cd ../frontend
npm install
```

### Запуск для разработки

1. Запустите backend:
```bash
cd backend
go run .
```

2. В другом терминале запустите frontend:
```bash
cd frontend
npm run dev
```

3. Откройте браузер на `http://localhost:5173`

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

Сервер будет доступен на порту, указанном в `config.yaml` (по умолчанию 8080).

## Структура проекта

```
/backend     - Go API сервер
/frontend    - Svelte приложение
/deploy      - скрипты для деплоя
/.gitea      - CI/CD workflows
/docs        - документация
```

## Конфигурация

Создайте файл `backend/config.yaml` на основе `backend/config.example.yaml`:

```yaml
# Вариант 1: Использовать готовый Access Token
gigachat_access_token: "your-access-token-here"

# Вариант 2: Использовать Authorization Key (токен получится автоматически)
# gigachat_auth_key: "MDE5YWQ5YzYtZDM3Ni03NTI0LTk1NGItNDljYTdjMThiMzQ5OjQ2YzQzZmYwLWY0MzEtNDE0OS1iZWVkLWM1NjdmYjg3MTU2ZA=="

port: "8080"
gigachat_api_url: "https://gigachat.devices.sberbank.ru/api/v1"
gigachat_auth_url: "https://ngw.devices.sberbank.ru:9443/api/v2/oauth"
```

**Получение Authorization Key:**
1. Зарегистрируйтесь на [портале разработчиков Сбера](https://developers.sber.ru)
2. Создайте проект и получите Client ID и Client Secret
3. Закодируйте пару `Client ID:Client Secret` в base64 или используйте готовый Authorization Key из личного кабинета

## Документация

- [API документация](API.md)
- [Руководство по развертыванию](DEPLOYMENT.md)
- [Руководство для разработчиков](DEVELOPMENT.md)
- [Архитектура проекта](ARCHITECTURE.md)
- [Настройка TLS](TLS_SETUP.md)

## Настройка TLS

GigaChat API требует сертификат НУЦ Минцифры. Подробные инструкции см. в [TLS_SETUP.md](TLS_SETUP.md)

**Быстрое решение для тестирования:**
```bash
export GIGACHAT_SKIP_TLS_VERIFY=true
cd backend
go run .
```

## Тестирование

Подробные инструкции по тестированию см. в [TESTING.md](../TESTING.md)

Быстрая проверка готовности:
```bash
./check-setup.sh
```

Тестирование API:
```bash
./test-api.sh "Привет!"
```

## Лицензия

MIT

