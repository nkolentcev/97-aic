#!/bin/bash

# Скрипт для анализа логов в SQLite
# Использование: ./analyze-logs.sh [database_path]

DB_PATH="${1:-backend/data.db}"

if [ ! -f "$DB_PATH" ]; then
    echo "❌ База данных не найдена: $DB_PATH"
    exit 1
fi

echo "=== Анализ логов из $DB_PATH ==="
echo ""

# Проверяем наличие sqlite3
if ! command -v sqlite3 &> /dev/null; then
    echo "❌ sqlite3 не установлен. Установите: sudo apt install sqlite3"
    exit 1
fi

echo "--- Общая статистика ---"
sqlite3 "$DB_PATH" <<EOF
SELECT 
    COUNT(*) as total_logs,
    SUM(CASE WHEN status_code = 500 THEN 1 ELSE 0 END) as errors_500,
    SUM(CASE WHEN status_code = 200 THEN 1 ELSE 0 END) as success_200,
    SUM(CASE WHEN status_code IS NULL THEN 1 ELSE 0 END) as null_status
FROM request_logs;
EOF

echo ""
echo "--- Последние 10 ошибок 500 ---"
sqlite3 -header -column "$DB_PATH" <<EOF
SELECT 
    id,
    session_id,
    status_code,
    duration_ms,
    datetime(created_at) as created_at
FROM request_logs
WHERE status_code = 500
ORDER BY created_at DESC
LIMIT 10;
EOF

echo ""
echo "--- Детали ошибок 500 с пустым content ---"
sqlite3 -header -column "$DB_PATH" <<EOF
SELECT 
    id,
    session_id,
    status_code,
    duration_ms,
    datetime(created_at) as created_at,
    substr(request_json, 1, 100) as request_preview,
    substr(response_json, 1, 200) as response_preview
FROM request_logs
WHERE status_code = 500
ORDER BY created_at DESC
LIMIT 5;
EOF

echo ""
echo "--- Полные детали последней ошибки 500 ---"
sqlite3 -header -column "$DB_PATH" <<EOF
SELECT 
    id,
    session_id,
    request_json,
    response_json,
    status_code,
    duration_ms,
    datetime(created_at) as created_at
FROM request_logs
WHERE status_code = 500
ORDER BY created_at DESC
LIMIT 1;
EOF

echo ""
echo "--- Анализ response_json для ошибок 500 ---"
sqlite3 "$DB_PATH" <<EOF
SELECT 
    id,
    CASE 
        WHEN response_json = '' THEN 'пустой'
        WHEN response_json LIKE '%"error"%' THEN 'содержит error'
        WHEN response_json LIKE '%"content":""%' THEN 'content пустой'
        ELSE 'другое'
    END as response_type,
    substr(response_json, 1, 150) as response_preview
FROM request_logs
WHERE status_code = 500
ORDER BY created_at DESC
LIMIT 5;
EOF

echo ""
echo "--- Статистика по провайдерам (из request_json) ---"
sqlite3 -header -column "$DB_PATH" <<EOF
SELECT 
    CASE 
        WHEN request_json LIKE '%"provider":"gigachat"%' THEN 'gigachat'
        WHEN request_json LIKE '%"provider":"groq"%' THEN 'groq'
        WHEN request_json LIKE '%"provider":"ollama"%' THEN 'ollama'
        ELSE 'не указан'
    END as provider,
    COUNT(*) as total,
    SUM(CASE WHEN status_code = 500 THEN 1 ELSE 0 END) as errors_500
FROM request_logs
GROUP BY provider
ORDER BY total DESC;
EOF

echo ""
echo "--- Ошибки по времени (последние 24 часа) ---"
sqlite3 -header -column "$DB_PATH" <<EOF
SELECT 
    strftime('%Y-%m-%d %H:00', created_at) as hour,
    COUNT(*) as total_errors
FROM request_logs
WHERE status_code = 500 
    AND created_at >= datetime('now', '-24 hours')
GROUP BY hour
ORDER BY hour DESC;
EOF

echo ""
echo "=== Анализ завершен ==="

