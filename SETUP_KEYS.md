# Настройка ключей API

## Быстрая проверка ключей

### Способ 1: Прямая проверка (без запуска сервера)

```bash
./check-keys-direct.sh backend/config.yaml
```

Скрипт проверит ключи напрямую через API провайдеров.

### Способ 2: Через запущенный сервер

```bash
# 1. Запустите сервер
cd backend && go run .

# 2. В другом терминале проверьте ключи
./check-keys-simple.sh
```

## Настройка ключей

### 1. GigaChat

1. Зарегистрируйтесь на https://developers.sber.ru/
2. Создайте приложение и получите:
   - Client ID
   - Client Secret
3. Создайте Base64 строку: `echo -n "ClientID:ClientSecret" | base64`
4. Добавьте в `backend/config.yaml`:
   ```yaml
   gigachat_auth_key: "ваш_base64_ключ"
   ```

Или используйте готовый токен:
```yaml
gigachat_access_token: "ваш_токен"
```

### 2. Groq

1. Зарегистрируйтесь на https://console.groq.com/
2. Создайте API ключ: https://console.groq.com/keys
3. Добавьте в `backend/config.yaml`:
   ```yaml
   providers:
     groq:
       enabled: true
       api_key: "gsk_ваш_ключ"
   ```

## Проверка конфигурации

После настройки ключей:

```bash
# Проверка без запуска сервера
./check-keys-direct.sh backend/config.yaml

# Или запустите сервер и проверьте
cd backend
go run .
```

В другом терминале:
```bash
curl http://localhost:8080/api/v2/providers
```

## Структура config.yaml

Файл должен находиться в директории `backend/`:

```
97-aic/
  backend/
    config.yaml          ← здесь
    config.example.yaml
    main.go
```

## Пример минимальной конфигурации

```yaml
default_provider: "groq"

# GigaChat (опционально)
gigachat_auth_key: "ваш_ключ"

# Groq
providers:
  groq:
    enabled: true
    api_key: "gsk_ваш_ключ"
    model: "llama-3.3-70b-versatile"

# Ollama (опционально, локально)
providers:
  ollama:
    enabled: true
    api_url: "http://localhost:11434"
    model: "llama3.2:3b"

port: "8080"
log_level: "info"
database_path: "data.db"
```

