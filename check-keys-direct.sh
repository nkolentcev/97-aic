#!/bin/bash

# Прямая проверка ключей GigaChat и Groq без запуска сервера
# Использование: ./check-keys-direct.sh [config.yaml]

CONFIG_FILE="${1:-config.yaml}"

echo "=== Прямая проверка ключей API ==="
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
        echo "✓ Найден access_token"
        echo "  Проверка валидности токена..."
        
        # Проверяем токен через API
        response=$(curl -s -X POST "https://gigachat.devices.sberbank.ru/api/v1/chat/completions" \
            -H "Authorization: Bearer $access_token" \
            -H "Content-Type: application/json" \
            -d '{
                "model": "GigaChat",
                "messages": [{"role": "user", "content": "test"}],
                "max_tokens": 10
            }' 2>&1)
        
        if echo "$response" | grep -q "error"; then
            error_msg=$(echo "$response" | grep -o '"message":"[^"]*"' | head -1 | sed 's/"message":"//g' | sed 's/"$//g')
            echo "❌ GigaChat: ошибка - $error_msg"
            return 1
        elif echo "$response" | grep -q "choices"; then
            echo "✓ GigaChat: токен валиден и работает"
            return 0
        else
            echo "⚠️  GigaChat: неожиданный ответ"
            echo "$response" | head -3
            return 1
        fi
    fi
    
    if [ -n "$auth_key" ]; then
        echo "✓ Найден auth_key (Base64)"
        echo "  Попытка получения токена..."
        
        # Пробуем получить токен
        token_response=$(curl -s -k -X POST "https://ngw.devices.sberbank.ru:9443/api/v2/oauth" \
            -H "Authorization: Basic $auth_key" \
            -H "RqUID: $(uuidgen 2>/dev/null || echo 'test-$(date +%s)')" \
            -H "Content-Type: application/x-www-form-urlencoded" \
            -H "Accept: application/json" \
            -d "scope=GIGACHAT_API_PERS" 2>&1)
        
        if echo "$token_response" | grep -q "access_token"; then
            token=$(echo "$token_response" | grep -o '"access_token":"[^"]*"' | head -1 | sed 's/"access_token":"//g' | sed 's/"$//g')
            echo "✓ GigaChat: токен успешно получен"
            echo "  Токен: ${token:0:30}..."
            
            # Проверяем токен
            echo "  Проверка работоспособности токена..."
            test_response=$(curl -s -X POST "https://gigachat.devices.sberbank.ru/api/v1/chat/completions" \
                -H "Authorization: Bearer $token" \
                -H "Content-Type: application/json" \
                -d '{
                    "model": "GigaChat",
                    "messages": [{"role": "user", "content": "test"}],
                    "max_tokens": 10
                }' 2>&1)
            
            if echo "$test_response" | grep -q "choices"; then
                echo "✓ GigaChat: токен работает"
                return 0
            else
                echo "⚠️  GigaChat: токен получен, но запрос не прошел"
                echo "$test_response" | head -3
                return 1
            fi
        else
            error_msg=$(echo "$token_response" | grep -o '"error_description":"[^"]*"' | head -1 | sed 's/"error_description":"//g' | sed 's/"$//g')
            if [ -n "$error_msg" ]; then
                echo "❌ GigaChat: ошибка получения токена - $error_msg"
            else
                echo "❌ GigaChat: ошибка получения токена"
                echo "$token_response" | head -5
            fi
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
    
    # Проверяем формат ключа
    if [[ ! "$api_key" =~ ^gsk_ ]]; then
        echo "⚠️  Groq: ключ не начинается с 'gsk_' (возможно неверный формат)"
        echo "   Формат должен быть: gsk_xxxxxxxxxxxxxxxxxxxxx"
    else
        echo "✓ Формат ключа корректный"
    fi
    
    echo "  Проверка валидности через Groq API..."
    
    # Проверяем через Groq API напрямую
    response=$(curl -s -X POST "https://api.groq.com/openai/v1/chat/completions" \
        -H "Authorization: Bearer $api_key" \
        -H "Content-Type: application/json" \
        -d '{
            "model": "llama-3.3-70b-versatile",
            "messages": [{"role": "user", "content": "test"}],
            "max_tokens": 10
        }' 2>&1)
    
    if echo "$response" | grep -q '"error"'; then
        error_msg=$(echo "$response" | grep -o '"message":"[^"]*"' | head -1 | sed 's/"message":"//g' | sed 's/"$//g')
        error_type=$(echo "$response" | grep -o '"type":"[^"]*"' | head -1 | sed 's/"type":"//g' | sed 's/"$//g')
        if [ -n "$error_msg" ]; then
            echo "❌ Groq: ошибка - $error_type: $error_msg"
        else
            echo "❌ Groq: ошибка при запросе"
            echo "$response" | head -5
        fi
        return 1
    elif echo "$response" | grep -q '"choices"'; then
        echo "✓ Groq: ключ валиден и работает"
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
    echo "Или укажите путь к конфигу: ./check-keys-direct.sh /path/to/config.yaml"
    exit 1
fi

echo "Чтение конфигурации из: $CONFIG_FILE"
echo ""

# Извлекаем ключи из конфига (улучшенный парсинг YAML)
GIGACHAT_AUTH_KEY=$(grep -E "^gigachat_auth_key:" "$CONFIG_FILE" | sed -E 's/^[^:]*:[[:space:]]*["'\'']?([^"'\'']*)["'\'']?[[:space:]]*$/\1/' | head -1)
GIGACHAT_ACCESS_TOKEN=$(grep -E "^gigachat_access_token:" "$CONFIG_FILE" | sed -E 's/^[^:]*:[[:space:]]*["'\'']?([^"'\'']*)["'\'']?[[:space:]]*$/\1/' | head -1)

# Проверяем Groq в новом формате (более надежный парсинг)
GROQ_ENABLED=$(grep -A 10 "^providers:" "$CONFIG_FILE" | grep -A 5 "groq:" | grep "enabled:" | sed -E 's/.*enabled:[[:space:]]*(true|false).*/\1/' | head -1)
GROQ_API_KEY=$(grep -A 10 "^providers:" "$CONFIG_FILE" | grep -A 5 "groq:" | grep "api_key:" | sed -E 's/.*api_key:[[:space:]]*["'\'']?([^"'\'']*)["'\'']?[[:space:]]*$/\1/' | head -1)

# Убираем placeholder значения
if [[ "$GIGACHAT_AUTH_KEY" == *"your-authorization-key"* ]] || [[ "$GIGACHAT_AUTH_KEY" == *"your-access-token"* ]] || [ -z "$GIGACHAT_AUTH_KEY" ]; then
    GIGACHAT_AUTH_KEY=""
fi
if [[ "$GIGACHAT_ACCESS_TOKEN" == *"your-access-token"* ]] || [ -z "$GIGACHAT_ACCESS_TOKEN" ]; then
    GIGACHAT_ACCESS_TOKEN=""
fi
if [[ "$GROQ_API_KEY" == *"your_groq_api_key"* ]] || [[ "$GROQ_API_KEY" == *"gsk_your"* ]] || [ -z "$GROQ_API_KEY" ]; then
    GROQ_API_KEY=""
fi

# Выводим найденные ключи (частично)
echo "Найденные ключи в конфиге:"
if [ -n "$GIGACHAT_AUTH_KEY" ]; then
    echo "  ✓ GigaChat auth_key: ${GIGACHAT_AUTH_KEY:0:30}..."
elif [ -n "$GIGACHAT_ACCESS_TOKEN" ]; then
    echo "  ✓ GigaChat access_token: ${GIGACHAT_ACCESS_TOKEN:0:30}..."
else
    echo "  ❌ GigaChat: ключ не найден"
fi

if [ "$GROQ_ENABLED" = "true" ] && [ -n "$GROQ_API_KEY" ]; then
    echo "  ✓ Groq api_key: ${GROQ_API_KEY:0:30}..."
elif [ "$GROQ_ENABLED" != "true" ]; then
    echo "  ⚠️  Groq: отключен в конфиге (enabled: false)"
else
    echo "  ❌ Groq: ключ не найден"
fi
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
    echo "⚠️  Ключ не настроен в конфиге"
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
    echo "✓ GigaChat: ключ валиден и работает"
else
    echo "❌ GigaChat: ключ не работает или не настроен"
fi

if [ "$GROQ_OK" = "true" ]; then
    echo "✓ Groq: ключ валиден и работает"
else
    echo "❌ Groq: ключ не работает или не настроен"
fi

echo ""
if [ "$GIGACHAT_OK" = "true" ] || [ "$GROQ_OK" = "true" ]; then
    echo "✓ Хотя бы один провайдер работает"
    exit 0
else
    echo "❌ Ни один провайдер не работает"
    echo ""
    echo "Проверьте:"
    echo "1. Правильность ключей в config.yaml"
    echo "2. Что ключи не являются placeholder значениями"
    echo "3. Что для Groq установлено enabled: true"
    exit 1
fi

