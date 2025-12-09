# Тестирование температуры

## Доступные провайдеры

Проект поддерживает три провайдера:

1. **gigachat** - GigaChat API (Сбер)
   - Модели: `GigaChat`, `GigaChat-Plus`, `GigaChat-Pro`
   - Требует: `gigachat_auth_key` или `gigachat_access_token` в конфиге

2. **groq** - Groq API (бесплатно)
   - Модели: зависят от конфигурации
   - Требует: `providers.groq.api_key` в конфиге

3. **ollama** - Ollama (локально)
   - Модели: зависят от установленных локально
   - Требует: запущенный Ollama сервер

## API Endpoint

**POST** `/api/v2/chat`

## Параметры запроса

```json
{
  "message": "текст запроса",
  "provider": "gigachat",           // опционально, по умолчанию из конфига
  "model": "GigaChat",              // опционально
  "system_prompt": "...",           // опционально
  "temperature": 0.7,               // опционально, 0-2
  "reasoning_mode": "direct",        // опционально: direct, step_by_step, experts
  "json_format": false,              // опционально
  "json_schema": "...",              // опционально
  "max_tokens": 1000,                // опционально
  "session_id": "...",               // опционально
  "use_history": false               // опционально
}
```

## Примеры запросов для тестирования температуры

### 1. GigaChat с температурой 0 (точность)

```bash
curl -X POST http://localhost:8080/api/v2/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Расскажи про квантовые компьютеры в 3 предложениях",
    "provider": "gigachat",
    "model": "GigaChat",
    "temperature": 0
  }'
```

### 2. GigaChat с температурой 0.7 (баланс)

```bash
curl -X POST http://localhost:8080/api/v2/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Расскажи про квантовые компьютеры в 3 предложениях",
    "provider": "gigachat",
    "model": "GigaChat",
    "temperature": 0.7
  }'
```

### 3. GigaChat с температурой 1.2 (креативность)

```bash
curl -X POST http://localhost:8080/api/v2/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Расскажи про квантовые компьютеры в 3 предложениях",
    "provider": "gigachat",
    "model": "GigaChat",
    "temperature": 1.2
  }'
```

### 4. Groq с разными температурами

```bash
# Температура 0
curl -X POST http://localhost:8080/api/v2/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Придумай название для стартапа в сфере AI",
    "provider": "groq",
    "temperature": 0
  }'

# Температура 0.7
curl -X POST http://localhost:8080/api/v2/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Придумай название для стартапа в сфере AI",
    "provider": "groq",
    "temperature": 0.7
  }'

# Температура 1.2
curl -X POST http://localhost:8080/api/v2/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Придумай название для стартапа в сфере AI",
    "provider": "groq",
    "temperature": 1.2
  }'
```

### 5. Ollama с температурой

```bash
curl -X POST http://localhost:8080/api/v2/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Объясни что такое машинное обучение",
    "provider": "ollama",
    "model": "llama2",
    "temperature": 0.7
  }'
```

### 6. С system prompt и температурой

```bash
curl -X POST http://localhost:8080/api/v2/chat \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Что такое Python?",
    "provider": "gigachat",
    "system_prompt": "Ты — эксперт по программированию. Отвечай кратко и с примерами кода.",
    "temperature": 0.5
  }'
```

## Получение списка провайдеров

```bash
curl http://localhost:8080/api/v2/providers
```

Ответ:
```json
{
  "providers": [
    {
      "name": "gigachat",
      "models": ["GigaChat", "GigaChat-Plus", "GigaChat-Pro"],
      "current_model": "GigaChat",
      "is_default": true
    },
    {
      "name": "groq",
      "models": ["llama-3.1-70b-versatile", "mixtral-8x7b-32768"],
      "current_model": "llama-3.1-70b-versatile",
      "is_default": false
    }
  ],
  "default_provider": "gigachat",
  "reasoning_modes": [
    {
      "id": "direct",
      "name": "Прямой ответ",
      "description": "Краткий ответ без рассуждений"
    },
    {
      "id": "step_by_step",
      "name": "Пошаговое решение",
      "description": "Разбивает задачу на шаги"
    },
    {
      "id": "experts",
      "name": "Группа экспертов",
      "description": "Несколько экспертов дают мнения"
    }
  ]
}
```

## Формат ответа

API возвращает streaming ответ в формате Server-Sent Events (SSE):

```
data: {"content":"часть ответа"}
data: {"content":" следующая часть"}
data: [DONE]
```

## Рекомендации по температуре

- **0-0.3** (низкая): Точные, детерминированные ответы. Подходит для:
  - Фактологических вопросов
  - Математических вычислений
  - Технических объяснений
  - Структурированных данных

- **0.7** (средняя): Баланс точности и креативности. Подходит для:
  - Общих вопросов
  - Объяснений концепций
  - Большинства задач

- **1.2-2.0** (высокая): Креативные и разнообразные ответы. Подходит для:
  - Генерации идей
  - Творческих задач
  - Вариативных ответов
  - Художественных текстов

## Тестирование через UI

1. Откройте приложение в браузере
2. Перейдите на вкладку **"Тест температуры"**
3. Введите запрос
4. Нажмите **"Запустить тест"**
5. Сравните результаты для температур 0, 0.7 и 1.2

## Примеры запросов для сравнения

### Тест 1: Фактологический вопрос
```
"Сколько планет в Солнечной системе?"
```
Ожидаемый результат: при температуре 0 ответ должен быть наиболее точным и стабильным.

### Тест 2: Креативная задача
```
"Придумай 5 названий для кафе в стиле ретро"
```
Ожидаемый результат: при температуре 1.2 ответы должны быть более разнообразными и креативными.

### Тест 3: Объяснение концепции
```
"Объясни что такое блокчейн простыми словами"
```
Ожидаемый результат: при температуре 0.7 баланс между точностью и понятностью.

