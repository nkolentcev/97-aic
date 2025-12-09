#!/bin/bash

# Скрипт для проверки валидности ключей GigaChat и Groq
# Использование: ./check-keys.sh [config.yaml]

CONFIG_FILE="${1:-config.yaml}"
API_URL="${API_URL:-http://localhost:8080}"

echo "=== Проверка ключей API ==="
echo "Конфиг: $CONFIG_FILE"
echo ""

# Функция для проверки GigaChat
check_gigachat() {
    local auth_key="$1"
    local access_token="$2"
    
    echo "--- Проверка GigaChat ---"
    
    if [ -z "$auth_key" ] && [ -z "$access_token" ]; then
        echo "❌ GigaChat: ключ не найден в конфиге"
        return 1
    fi
    
    if [ -n "$access_token" ]; then
        echo "✓ Найден access_token (готовый токен)"
        echo "  Проверка валидности через API..."
        
        # Проверяем через API приложения
        response=$(curl -s -X POST "$API_URL/api/v2/chat" \
            -H "Content-Type: application/json" \
            -d '{
                "message": "Привет",
                "provider": "gigachat",
                "temperature": 0.7
            }' 2>&1)
        
        if echo "$response" | grep -q "error"; then
            echo "❌ GigaChat: ошибка при запросе"
            echo "$response" | grep -o '"error":"[^"]*"' | head -1
            return 1
        else
            echo "✓ GigaChat: ключ работает"
            return 0
        fi
    fi
    
    if [ -n "$auth_key" ]; then
        echo "✓ Найден auth_key (Base64)"
        echo "  Проверка получения токена..."
        
        # Пробуем получить токен
        token_response=$(curl -s -X POST "https://ngw.devices.sberbank.ru:9443/api/v2/oauth" \
            -H "Authorization: Basic $auth_key" \
            -H "RqUID: $(uuidgen 2>/dev/null || echo 'test')" \
            -H "Content-Type: application/x-www-form-urlencoded" \
            -H "Accept: application/json" \
            -d "scope=GIGACHAT_API_PERS" 2>&1)
        
        if echo "$token_response" | grep -q "access_token"; then
            echo "✓ GigaChat: токен успешно получен"
            return 0
        else
            echo "❌ GigaChat: ошибка получения токена"
            echo "$token_response" | head -3
            return 1
        fi
    fi
}

# Функция для проверки Groq
check_groq() {
    local api_key="$1"
    
    echo "--- Проверка Groq ---"
    
    if [ -z "$api_key" ]; then
        echo "❌ Groq: ключ не найден в конфиге"
        return 1
    fi
    
    if [[ ! "$api_key" =~ ^gsk_ ]]; then
        echo "⚠️  Groq: ключ не начинается с 'gsk_' (возможно неверный формат)"
    fi
    
    echo "✓ Найден API ключ"
    echo "  Проверка валидности через API..."
    
    # Проверяем через Groq API напрямую
    response=$(curl -s -X POST "https://api.groq.com/openai/v1/chat/completions" \
        -H "Authorization: Bearer $api_key" \
        -H "Content-Type: application/json" \
        -d '{
            "model": "llama-3.3-70b-versatile",
            "messages": [{"role": "user", "content": "test"}],
            "max_tokens": 10
        }' 2>&1)
    
    if echo "$response" | grep -q "error"; then
        echo "❌ Groq: ошибка при запросе"
        echo "$response" | grep -o '"message":"[^"]*"' | head -1 | sed 's/"message":"//g' | sed 's/"$//g'
        return 1
    elif echo "$response" | grep -q "choices"; then
        echo "✓ Groq: ключ работает"
        return 0
    else
        echo "⚠️  Groq: неожиданный ответ"
        echo "$response" | head -5
        return 1
    fi
}

# Читаем конфиг
if [ ! -f "$CONFIG_FILE" ]; then
    echo "❌ Файл конфигурации не найден: $CONFIG_FILE"
    echo ""
    echo "Создайте config.yaml на основе config.example.yaml"
    exit 1
fi

# Извлекаем ключи из конфига
GIGACHAT_AUTH_KEY=$(grep -E "^gigachat_auth_key:" "$CONFIG_FILE" | sed 's/.*: *"\(.*\)"/\1/' | sed 's/.*: *\(.*\)/\1/' | head -1 | tr -d ' ')
GIGACHAT_ACCESS_TOKEN=$(grep -E "^gigachat_access_token:" "$CONFIG_FILE" | sed 's/.*: *"\(.*\)"/\1/' | sed 's/.*: *\(.*\)/\1/' | head -1 | tr -d ' ')

# Проверяем Groq в новом формате
GROQ_ENABLED=$(grep -A 5 "^providers:" "$CONFIG_FILE" | grep -A 3 "groq:" | grep "enabled:" | sed 's/.*enabled: *\(.*\)/\1/' | tr -d ' ')
GROQ_API_KEY=$(grep -A 5 "^providers:" "$CONFIG_FILE" | grep -A 3 "groq:" | grep "api_key:" | sed 's/.*api_key: *"\(.*\)"/\1/' | sed 's/.*api_key: *\(.*\)/\1/' | head -1 | tr -d ' ')

# Убираем placeholder значения
if [[ "$GIGACHAT_AUTH_KEY" == *"your-authorization-key"* ]] || [[ "$GIGACHAT_AUTH_KEY" == *"your-access-token"* ]]; then
    GIGACHAT_AUTH_KEY=""
fi
if [[ "$GIGACHAT_ACCESS_TOKEN" == *"your-access-token"* ]]; then
    GIGACHAT_ACCESS_TOKEN=""
fi
if [[ "$GROQ_API_KEY" == *"your_groq_api_key"* ]] || [[ "$GROQ_API_KEY" == *"gsk_your"* ]]; then
    GROQ_API_KEY=""
fi

echo "Найденные ключи:"
[ -n "$GIGACHAT_AUTH_KEY" ] && echo "  GigaChat auth_key: ${GIGACHAT_AUTH_KEY:0:20}..." || echo "  GigaChat auth_key: не найден"
[ -n "$GIGACHAT_ACCESS_TOKEN" ] && echo "  GigaChat access_token: ${GIGACHAT_ACCESS_TOKEN:0:20}..." || echo "  GigaChat access_token: не найден"
[ -n "$GROQ_API_KEY" ] && echo "  Groq api_key: ${GROQ_API_KEY:0:20}..." || echo "  Groq api_key: не найден"
echo ""

# Проверяем ключи
GIGACHAT_OK=false
GROQ_OK=false

if [ -n "$GIGACHAT_AUTH_KEY" ] || [ -n "$GIGACHAT_ACCESS_TOKEN" ]; then
    if check_gigachat "$GIGACHAT_AUTH_KEY" "$GIGACHAT_ACCESS_TOKEN"; then
        GIGACHAT_OK=true
    fi
else
    echo "--- GigaChat ---"
    echo "⚠️  Ключ не настроен"
fi

echo ""

if [ "$GROQ_ENABLED" = "true" ] && [ -n "$GROQ_API_KEY" ]; then
    if check_groq "$GROQ_API_KEY"; then
        GROQ_OK=true
    fi
else
    echo "--- Groq ---"
    if [ "$GROQ_ENABLED" != "true" ]; then
        echo "⚠️  Groq отключен в конфиге (enabled: false)"
    else
        echo "⚠️  Ключ не настроен"
    fi
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

if [ "$GIGACHAT_OK" = "true" ] || [ "$GROQ_OK" = "true" ]; then
    exit 0
else
    exit 1
fi

