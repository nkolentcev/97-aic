#!/bin/bash

# Скрипт для тестирования температуры с разными значениями
# Использование: ./test-temperature.sh [provider] [message]

API_URL="${API_URL:-http://localhost:8080}"
PROVIDER="${1:-gigachat}"
MESSAGE="${2:-Расскажи про квантовые компьютеры в 3 предложениях}"

echo "=== Тестирование температуры для провайдера: $PROVIDER ==="
echo "Запрос: $MESSAGE"
echo ""

# Функция для отправки запроса
send_request() {
    local temp=$1
    local label=$2
    
    echo "--- Температура: $temp ($label) ---"
    
    curl -s -X POST "$API_URL/api/v2/chat" \
        -H "Content-Type: application/json" \
        -d "{
            \"message\": \"$MESSAGE\",
            \"provider\": \"$PROVIDER\",
            \"temperature\": $temp
        }" | grep -o '"content":"[^"]*"' | sed 's/"content":"//g' | sed 's/"$//g' | tr -d '\n'
    
    echo ""
    echo ""
}

# Тестируем с разными температурами
send_request 0 "Точность"
sleep 1
send_request 0.7 "Баланс"
sleep 1
send_request 1.2 "Креативность"

echo "=== Тестирование завершено ==="

