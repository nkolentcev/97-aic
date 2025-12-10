#!/bin/bash

# Скрипт для установки всех моделей для сравнения

echo "=== Установка моделей HuggingFace для сравнения ==="
echo ""

MODELS=(
    "qwen2.5:0.5b"
    "qwen2.5:1.5b"
    "llama3.2:3b"
    "mistral:7b"
    "llama3.1:8b"
    "qwen2.5:7b"
)

for model in "${MODELS[@]}"; do
    echo "Установка $model..."
    ollama pull "$model"
    if [ $? -eq 0 ]; then
        echo "✅ $model установлена"
    else
        echo "❌ Ошибка при установке $model"
    fi
    echo ""
done

echo "=== Установка завершена ==="
echo ""
echo "Проверьте установленные модели:"
ollama list
