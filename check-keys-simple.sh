#!/bin/bash

# Упрощенная проверка ключей через API приложения
# Использование: ./check-keys-simple.sh

API_URL="${API_URL:-http://localhost:8080}"

echo "=== Проверка ключей через API приложения ==="
echo "API URL: $API_URL"
echo ""

# Проверяем доступность API
if ! curl -s "$API_URL/health" > /dev/null 2>&1; then
    echo "❌ API недоступен по адресу $API_URL"
    echo "   Убедитесь, что сервер запущен"
    exit 1
fi

echo "✓ API доступен"
echo ""

# Получаем список провайдеров
echo "--- Список доступных провайдеров ---"
providers_response=$(curl -s "$API_URL/api/v2/providers")

if [ $? -ne 0 ]; then
    echo "❌ Ошибка получения списка провайдеров"
    exit 1
fi

echo "$providers_response" | python3 -m json.tool 2>/dev/null || echo "$providers_response"
echo ""

# Проверяем GigaChat
echo "--- Проверка GigaChat ---"
gigachat_response=$(curl -s -X POST "$API_URL/api/v2/chat" \
    -H "Content-Type: application/json" \
    -d '{
        "message": "Тест",
        "provider": "gigachat",
        "temperature": 0.7
    }' 2>&1)

if echo "$gigachat_response" | grep -q "error"; then
    error_msg=$(echo "$gigachat_response" | grep -o '"error":"[^"]*"' | head -1 | sed 's/"error":"//g' | sed 's/"$//g')
    echo "❌ GigaChat: $error_msg"
    GIGACHAT_OK=false
elif echo "$gigachat_response" | grep -q "data:"; then
    echo "✓ GigaChat: работает"
    GIGACHAT_OK=true
else
    echo "⚠️  GigaChat: неожиданный ответ"
    echo "$gigachat_response" | head -3
    GIGACHAT_OK=false
fi

echo ""

# Проверяем Groq
echo "--- Проверка Groq ---"
groq_response=$(curl -s -X POST "$API_URL/api/v2/chat" \
    -H "Content-Type: application/json" \
    -d '{
        "message": "Тест",
        "provider": "groq",
        "temperature": 0.7
    }' 2>&1)

if echo "$groq_response" | grep -q "error"; then
    error_msg=$(echo "$groq_response" | grep -o '"error":"[^"]*"' | head -1 | sed 's/"error":"//g' | sed 's/"$//g')
    echo "❌ Groq: $error_msg"
    GROQ_OK=false
elif echo "$groq_response" | grep -q "data:"; then
    echo "✓ Groq: работает"
    GROQ_OK=true
else
    echo "⚠️  Groq: неожиданный ответ"
    echo "$groq_response" | head -3
    GROQ_OK=false
fi

echo ""
echo "=== Итоги ==="
if [ "$GIGACHAT_OK" = "true" ]; then
    echo "✓ GigaChat: работает"
else
    echo "❌ GigaChat: не работает или не настроен"
fi

if [ "$GROQ_OK" = "true" ]; then
    echo "✓ Groq: работает"
else
    echo "❌ Groq: не работает или не настроен"
fi

