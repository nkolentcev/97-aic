#!/bin/bash

# Скрипт для тестирования API сравнения моделей

API_URL="${API_URL:-http://localhost:8080}"
ENDPOINT="$API_URL/api/v2/models/compare"

echo "=== Тестирование API сравнения моделей ==="
echo "Endpoint: $ENDPOINT"
echo ""

# Проверка доступности API
if ! curl -s "$API_URL/health" > /dev/null 2>&1; then
    echo "❌ API сервер недоступен на $API_URL"
    echo "Убедитесь, что сервер запущен:"
    echo "  cd backend && go run ."
    exit 1
fi

echo "✅ API сервер доступен"
echo ""

# Тестовый запрос
TEST_MESSAGE="Объясни, что такое машинное обучение простыми словами в 2-3 предложениях."

echo "Отправка запроса..."
echo "Сообщение: $TEST_MESSAGE"
echo ""

# Выполняем запрос с сохранением HTTP статуса и увеличением таймаута
# Используем --max-time 600 (10 минут) для больших моделей
HTTP_CODE=$(curl -s --max-time 600 -o /tmp/response.json -w "%{http_code}" -X POST "$ENDPOINT" \
  -H "Content-Type: application/json" \
  -d "{
    \"message\": \"$TEST_MESSAGE\"
  }" 2>/tmp/curl_error.log)

CURL_EXIT=$?
CURL_ERROR=$(cat /tmp/curl_error.log 2>/dev/null || echo "")
RESPONSE=$(cat /tmp/response.json 2>/dev/null || echo "")

# Проверяем, есть ли валидный JSON ответ (даже если curl вернул ошибку)
if [ -n "$RESPONSE" ] && echo "$RESPONSE" | jq empty 2>/dev/null; then
    # Ответ получен и валиден - проверяем наличие ошибок в самом ответе
    if echo "$RESPONSE" | jq -e '.results' > /dev/null 2>&1; then
        # Успешный ответ с результатами
        if [ $CURL_EXIT -ne 0 ]; then
            echo "⚠️  Curl завершился с кодом $CURL_EXIT, но ответ получен (возможен таймаут после получения данных)"
        fi
    else
        # Ответ есть, но это ошибка
        echo "❌ Ошибка в ответе API:"
        echo "$RESPONSE" | jq '.' 2>/dev/null || echo "$RESPONSE"
        exit 1
    fi
elif [ $CURL_EXIT -ne 0 ]; then
    # Нет валидного ответа и curl вернул ошибку
    echo "❌ Ошибка при отправке запроса (curl exit code: $CURL_EXIT)"
    if [ -n "$CURL_ERROR" ]; then
        echo "Ошибка curl: $CURL_ERROR"
    fi
    if [ -n "$RESPONSE" ]; then
        echo "Ответ сервера (возможно неполный):"
        echo "$RESPONSE"
    fi
    exit 1
fi

# Проверка HTTP статуса (если удалось получить)
if [ -n "$HTTP_CODE" ] && [ "$HTTP_CODE" != "200" ] && [ "$HTTP_CODE" != "000" ]; then
    echo "⚠️  HTTP код: $HTTP_CODE (но ответ получен)"
fi

# Проверка наличия ошибок в ответе
if echo "$RESPONSE" | grep -q '"error"'; then
    echo "⚠️  Обнаружены ошибки в ответе API:"
    echo "$RESPONSE" | jq '.results[] | select(.error != null) | {model: .model, error: .error}' 2>/dev/null || echo "$RESPONSE"
    echo ""
    echo "Продолжаем вывод результатов..."
    echo ""
fi

echo "✅ Запрос выполнен успешно"
echo ""

# Красивый вывод результатов
if command -v jq &> /dev/null; then
    echo "=== Результаты сравнения ==="
    echo ""
    
    echo "Сводка:"
    echo "$RESPONSE" | jq '.summary'
    echo ""
    
    echo "Сравнение:"
    echo "$RESPONSE" | jq '.comparison'
    echo ""
    
    echo "Детальные результаты:"
    echo "$RESPONSE" | jq '.results[] | {model: .model, duration_ms: .duration_ms, tokens: .tokens_total, tokens_per_sec: .tokens_per_sec, error: .error}'
else
    echo "=== Результаты (установите jq для красивого вывода) ==="
    echo "$RESPONSE"
fi
