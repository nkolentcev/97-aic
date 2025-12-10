# Сравнение моделей HuggingFace

## Описание

API endpoint для сравнения производительности разных моделей HuggingFace на одном и том же запросе. Замеряет время ответа, количество токенов, стоимость и сравнивает качество ответов.

## Установка Ollama

Для использования моделей HuggingFace через Ollama необходимо установить Ollama:

```bash
# Linux
curl -fsSL https://ollama.com/install.sh | sh

# Или через пакетный менеджер
# Ubuntu/Debian
curl -fsSL https://ollama.com/install.sh | sh

# Запуск Ollama сервера
ollama serve

# В другом терминале - скачивание моделей (каждую модель нужно устанавливать отдельно)
ollama pull qwen2.5:0.5b
ollama pull qwen2.5:1.5b
ollama pull llama3.2:3b
ollama pull mistral:7b
ollama pull llama3.1:8b
ollama pull qwen2.5:7b

# Или используйте цикл для автоматической установки:
for model in qwen2.5:0.5b qwen2.5:1.5b llama3.2:3b mistral:7b llama3.1:8b qwen2.5:7b; do
  ollama pull $model
done
```

## API Endpoint

### POST /api/v2/models/compare

Сравнивает производительность нескольких моделей на одном запросе.

#### Запрос

```json
{
  "message": "Объясни, что такое машинное обучение простыми словами",
  "models": [
    "ollama:qwen2.5:0.5b",
    "ollama:qwen2.5:1.5b",
    "ollama:llama3.2:3b",
    "ollama:mistral:7b",
    "ollama:llama3.1:8b"
  ]
}
```

Если `models` не указан, используются модели по умолчанию из начала, середины и конца списка.

#### Ответ

```json
{
  "message": "Объясни, что такое машинное обучение простыми словами",
  "results": [
    {
      "provider": "ollama",
      "model": "qwen2.5:0.5b",
      "response": "Машинное обучение - это...",
      "duration_ms": 1234,
      "tokens_output": 150,
      "tokens_total": 150,
      "cost": 0.0,
      "response_time": 1.234,
      "tokens_per_sec": 121.5
    },
    {
      "provider": "ollama",
      "model": "llama3.2:3b",
      "response": "Машинное обучение представляет собой...",
      "duration_ms": 2345,
      "tokens_output": 200,
      "tokens_total": 200,
      "cost": 0.0,
      "response_time": 2.345,
      "tokens_per_sec": 85.3
    }
  ],
  "summary": {
    "total_models": 5,
    "success_count": 5,
    "error_count": 0,
    "avg_duration_ms": 2000,
    "fastest_model": "ollama:qwen2.5:0.5b",
    "slowest_model": "ollama:llama3.1:8b",
    "total_cost": 0.0
  },
  "comparison": {
    "best_response": "ollama:llama3.1:8b",
    "longest_response": "ollama:llama3.1:8b",
    "shortest_response": "ollama:qwen2.5:0.5b"
  }
}
```

## Метрики

### Время ответа (duration_ms)
Время от отправки запроса до получения полного ответа в миллисекундах.

### Количество токенов
- `tokens_output` - количество токенов в ответе
- `tokens_total` - общее количество токенов (входные + выходные)
- Подсчет приблизительный: ~4 символа на токен

### Скорость генерации (tokens_per_sec)
Количество токенов, генерируемых в секунду.

### Стоимость (cost)
Для локальных моделей (Ollama) стоимость = 0. Для платных провайдеров вычисляется на основе количества токенов.

## Модели по умолчанию

При отсутствии параметра `models` используются следующие модели:

1. **Начало списка** (маленькие модели):
   - `qwen2.5:0.5b` - 0.5B параметров
   - `qwen2.5:1.5b` - 1.5B параметров

2. **Середина списка** (средние модели):
   - `llama3.2:3b` - 3B параметров
   - `mistral:7b` - 7B параметров

3. **Конец списка** (большие модели):
   - `llama3.1:8b` - 8B параметров
   - `qwen2.5:7b` - 7B параметров

## Пример использования

### cURL

```bash
curl -X POST http://localhost:8080/api/v2/models/compare \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Напиши короткое стихотворение о программировании"
  }'
```

### JavaScript

```javascript
const response = await fetch('http://localhost:8080/api/v2/models/compare', {
  method: 'POST',
  headers: {
    'Content-Type': 'application/json',
  },
  body: JSON.stringify({
    message: 'Объясни концепцию рекурсии',
    models: [
      'ollama:qwen2.5:0.5b',
      'ollama:llama3.2:3b',
      'ollama:llama3.1:8b'
    ]
  })
});

const data = await response.json();
console.log('Самый быстрый:', data.summary.fastest_model);
console.log('Самый медленный:', data.summary.slowest_model);
```

## Конфигурация

Убедитесь, что Ollama провайдер включен в `config.yaml`:

```yaml
providers:
  ollama:
    enabled: true
    api_url: "http://localhost:11434"
    model: "llama3.2:3b"
```

## Ограничения

1. Подсчет токенов приблизительный (4 символа на токен)
2. Для точного подсчета токенов требуется интеграция с токенизаторами
3. Сравнение качества ответов упрощенное (по длине ответа)
4. Все модели тестируются последовательно (не параллельно)

## Улучшения в будущем

- Параллельное выполнение запросов к моделям
- Точный подсчет токенов через токенизаторы
- Метрики качества ответов (BLEU, ROUGE, семантическое сходство)
- Кэширование результатов для одинаковых запросов
