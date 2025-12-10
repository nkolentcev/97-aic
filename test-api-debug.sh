#!/bin/bash

# Скрипт для детальной диагностики API

API_URL="${API_URL:-http://localhost:8080}"

echo "=== Диагностика API ==="
echo ""

# 1. Проверка health endpoint
echo "1. Проверка health endpoint:"
HEALTH_RESPONSE=$(curl -s --max-time 5 "$API_URL/health" 2>&1)
if [ $? -eq 0 ] && [ -n "$HEALTH_RESPONSE" ]; then
    echo "$HEALTH_RESPONSE" | jq '.' 2>/dev/null || echo "$HEALTH_RESPONSE"
else
    echo "❌ Сервер недоступен на $API_URL"
    echo "Запустите сервер: cd backend && go run ."
    exit 1
fi
echo ""
echo ""

# 2. Проверка списка провайдеров
echo "2. Проверка доступных провайдеров:"
PROVIDERS_RESPONSE=$(curl -s --max-time 5 "$API_URL/api/v2/providers" 2>&1)
if [ $? -eq 0 ] && [ -n "$PROVIDERS_RESPONSE" ]; then
    echo "$PROVIDERS_RESPONSE" | jq '.' 2>/dev/null || echo "$PROVIDERS_RESPONSE"
else
    echo "❌ Не удалось получить список провайдеров"
    echo "$PROVIDERS_RESPONSE"
fi
echo ""
echo ""

# 3. Тест сравнения моделей с детальным выводом
echo "3. Тест сравнения моделей:"
echo "Запрос:"
echo '{"message": "Привет"}'
echo ""
echo "Ответ:"

HTTP_CODE=$(curl -s --max-time 300 -o /tmp/response.json -w "%{http_code}" -X POST "$API_URL/api/v2/models/compare" \
  -H "Content-Type: application/json" \
  -d '{"message": "Привет"}' 2>/tmp/curl_error.log)

CURL_EXIT=$?
CURL_ERROR=$(cat /tmp/curl_error.log 2>/dev/null || echo "")

echo "HTTP код: $HTTP_CODE"
if [ $CURL_EXIT -ne 0 ]; then
    echo "❌ Ошибка curl (exit code: $CURL_EXIT)"
    if [ -n "$CURL_ERROR" ]; then
        echo "Ошибка: $CURL_ERROR"
    fi
fi

if [ -f /tmp/response.json ]; then
    echo "Тело ответа:"
    cat /tmp/response.json | jq '.' 2>/dev/null || cat /tmp/response.json
else
    echo "❌ Файл ответа не создан"
fi
echo ""
