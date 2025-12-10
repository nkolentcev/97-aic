# Устранение неполадок

## Проблема: Ошибка при отправке запроса к API

### Шаг 1: Проверка запуска сервера

```bash
# Проверьте, запущен ли сервер
curl http://localhost:8080/health

# Если не отвечает, запустите сервер:
cd backend
go run .
```

### Шаг 2: Проверка конфигурации Ollama

Убедитесь, что в `backend/config.yaml` Ollama включен:

```yaml
providers:
  ollama:
    enabled: true
    api_url: "http://localhost:11434"
    model: "llama3.2:3b"
```

### Шаг 3: Проверка запуска Ollama

```bash
# Проверьте, запущен ли Ollama сервер
curl http://localhost:11434/api/tags

# Если не отвечает, запустите:
ollama serve

# Проверьте установленные модели
ollama list
```

### Шаг 4: Установка моделей

Если модели не установлены:

```bash
# Используйте скрипт
./install-models.sh

# Или установите вручную
ollama pull qwen2.5:0.5b
ollama pull qwen2.5:1.5b
ollama pull llama3.2:3b
ollama pull mistral:7b
ollama pull llama3.1:8b
ollama pull qwen2.5:7b
```

### Шаг 5: Диагностика API

```bash
# Запустите диагностический скрипт
./test-api-debug.sh

# Или проверьте вручную
curl http://localhost:8080/api/v2/providers | jq '.'
```

### Шаг 6: Проверка логов сервера

При запуске сервера проверьте логи на наличие ошибок:

```bash
cd backend
go run . 2>&1 | tee server.log
```

Ищите сообщения:
- "Ollama провайдер зарегистрирован" - должно быть при старте
- "начато сравнение моделей" - при запросе
- Ошибки подключения к Ollama

### Шаг 7: Тестирование с минимальным запросом

```bash
# Простой тест одной модели
curl -X POST http://localhost:8080/api/v2/chat \
  -H "Content-Type: application/json" \
  -d '{
    "provider": "ollama",
    "model": "qwen2.5:0.5b",
    "message": "Привет"
  }'
```

## Частые проблемы

### Проблема: "Провайдер не найден"

**Решение**: Убедитесь, что Ollama включен в конфиге и сервер перезапущен.

### Проблема: "Connection refused" к Ollama

**Решение**: Запустите `ollama serve` в отдельном терминале.

### Проблема: "Model not found"

**Решение**: Установите модель через `ollama pull <model_name>`.

### Проблема: Таймаут запроса

**Решение**: Увеличьте таймаут в скрипте или дождитесь завершения (модели могут работать медленно).

## Полная проверка системы

```bash
# 1. Проверка Ollama
./check-ollama.sh

# 2. Проверка API
./test-api-debug.sh

# 3. Тест сравнения моделей
./test-models-compare.sh
```
