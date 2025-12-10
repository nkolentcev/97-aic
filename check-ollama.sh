#!/bin/bash

# Скрипт для проверки установки Ollama и доступных моделей

echo "=== Проверка установки Ollama ==="

# Проверка наличия Ollama
if ! command -v ollama &> /dev/null; then
    echo "❌ Ollama не установлен"
    echo ""
    echo "Установите Ollama:"
    echo "  curl -fsSL https://ollama.com/install.sh | sh"
    exit 1
fi

echo "✅ Ollama установлен"
echo ""

# Проверка запущенного сервера Ollama
if ! curl -s http://localhost:11434/api/tags > /dev/null 2>&1; then
    echo "⚠️  Ollama сервер не запущен"
    echo ""
    echo "Запустите Ollama сервер:"
    echo "  ollama serve"
    echo ""
    echo "Или в фоновом режиме:"
    echo "  ollama serve &"
    exit 1
fi

echo "✅ Ollama сервер запущен"
echo ""

# Получение списка установленных моделей
echo "=== Установленные модели ==="
INSTALLED_MODELS=$(ollama list 2>/dev/null | tail -n +2 | awk '{print $1}')

if [ -z "$INSTALLED_MODELS" ]; then
    echo "⚠️  Модели не установлены"
    echo ""
    echo "Установите модели для сравнения (каждую отдельно):"
    echo "  ollama pull qwen2.5:0.5b"
    echo "  ollama pull qwen2.5:1.5b"
    echo "  ollama pull llama3.2:3b"
    echo "  ollama pull mistral:7b"
    echo "  ollama pull llama3.1:8b"
    echo "  ollama pull qwen2.5:7b"
    echo ""
    echo "Или используйте цикл:"
    echo "  for model in qwen2.5:0.5b qwen2.5:1.5b llama3.2:3b mistral:7b llama3.1:8b qwen2.5:7b; do ollama pull \$model; done"
else
    echo "$INSTALLED_MODELS" | while read -r model; do
        if [ -n "$model" ]; then
            echo "  ✅ $model"
        fi
    done
fi

echo ""
echo "=== Рекомендуемые модели для сравнения ==="
echo "  • qwen2.5:0.5b   (0.5B параметров - начало списка)"
echo "  • qwen2.5:1.5b   (1.5B параметров - начало списка)"
echo "  • llama3.2:3b    (3B параметров - середина списка)"
echo "  • mistral:7b     (7B параметров - середина списка)"
echo "  • llama3.1:8b     (8B параметров - конец списка)"
echo "  • qwen2.5:7b     (7B параметров - конец списка)"
