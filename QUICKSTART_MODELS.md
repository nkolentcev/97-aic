# Быстрый старт: Сравнение моделей HuggingFace

## Установка Ollama

```bash
# Установка Ollama
curl -fsSL https://ollama.com/install.sh | sh

# Запуск сервера Ollama
ollama serve

# В другом терминале - установка моделей для сравнения
# Вариант 1: Использовать скрипт (рекомендуется)
./install-models.sh

# Вариант 2: Установить каждую модель отдельно
ollama pull qwen2.5:0.5b
ollama pull qwen2.5:1.5b
ollama pull llama3.2:3b
ollama pull mistral:7b
ollama pull llama3.1:8b
ollama pull qwen2.5:7b
```

## Проверка установки

```bash
./check-ollama.sh
```

## Запуск сервера

```bash
cd backend
go run .
```

## Тестирование API

```bash
# Простой тест
./test-models-compare.sh

# Или через curl
curl -X POST http://localhost:8080/api/v2/models/compare \
  -H "Content-Type: application/json" \
  -d '{
    "message": "Объясни, что такое машинное обучение"
  }'
```

## Результат

API вернет JSON с:
- Временем ответа каждой модели
- Количеством токенов
- Скоростью генерации (токенов/сек)
- Сводкой: самая быстрая/медленная модель
- Сравнением качества ответов

Подробнее: [docs/MODELS_COMPARISON.md](docs/MODELS_COMPARISON.md)
