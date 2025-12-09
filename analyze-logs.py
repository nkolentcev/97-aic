#!/usr/bin/env python3
"""
Скрипт для анализа логов из SQLite базы данных
Использование: python3 analyze-logs.py [database_path]
"""

import sqlite3
import sys
import json
from datetime import datetime

DB_PATH = sys.argv[1] if len(sys.argv) > 1 else "backend/data.db"

def analyze_logs():
    try:
        conn = sqlite3.connect(DB_PATH)
        conn.row_factory = sqlite3.Row
        cursor = conn.cursor()
        
        print(f"=== Анализ логов из {DB_PATH} ===\n")
        
        # Общая статистика
        print("--- Общая статистика ---")
        cursor.execute("""
            SELECT 
                COUNT(*) as total_logs,
                SUM(CASE WHEN status_code = 500 THEN 1 ELSE 0 END) as errors_500,
                SUM(CASE WHEN status_code = 200 THEN 1 ELSE 0 END) as success_200,
                SUM(CASE WHEN status_code IS NULL THEN 1 ELSE 0 END) as null_status
            FROM request_logs
        """)
        row = cursor.fetchone()
        print(f"Всего логов: {row['total_logs']}")
        print(f"Ошибок 500: {row['errors_500']}")
        print(f"Успешных 200: {row['success_200']}")
        print(f"Без статуса: {row['null_status']}")
        
        # Последние ошибки 500
        print("\n--- Последние 10 ошибок 500 ---")
        cursor.execute("""
            SELECT 
                id,
                session_id,
                status_code,
                duration_ms,
                datetime(created_at) as created_at
            FROM request_logs
            WHERE status_code = 500
            ORDER BY created_at DESC
            LIMIT 10
        """)
        rows = cursor.fetchall()
        if rows:
            print(f"{'ID':<6} {'Session ID':<30} {'Status':<8} {'Duration':<10} {'Created At'}")
            print("-" * 90)
            for row in rows:
                print(f"{row['id']:<6} {row['session_id'] or 'N/A':<30} {row['status_code']:<8} {row['duration_ms'] or 0:<10} {row['created_at']}")
        else:
            print("Ошибок 500 не найдено")
        
        # Детали ошибок с пустым content
        print("\n--- Детали ошибок 500 с анализом response ---")
        cursor.execute("""
            SELECT 
                id,
                session_id,
                status_code,
                duration_ms,
                datetime(created_at) as created_at,
                request_json,
                response_json
            FROM request_logs
            WHERE status_code = 500
            ORDER BY created_at DESC
            LIMIT 5
        """)
        rows = cursor.fetchall()
        for i, row in enumerate(rows, 1):
            print(f"\nОшибка #{i} (ID: {row['id']}):")
            print(f"  Session: {row['session_id'] or 'N/A'}")
            print(f"  Время: {row['created_at']}")
            print(f"  Длительность: {row['duration_ms'] or 0}ms")
            
            # Анализ request
            try:
                req = json.loads(row['request_json']) if row['request_json'] else {}
                print(f"  Запрос:")
                print(f"    - Provider: {req.get('provider', 'N/A')}")
                print(f"    - Model: {req.get('model', 'N/A')}")
                print(f"    - Message: {req.get('message', 'N/A')[:50]}...")
            except:
                print(f"  Запрос: {row['request_json'][:100]}...")
            
            # Анализ response
            response_json = row['response_json'] or ""
            if not response_json:
                print(f"  Ответ: ПУСТОЙ")
            elif response_json == '{"content":"","status":500}':
                print(f"  Ответ: content пустой, status 500")
            else:
                try:
                    resp = json.loads(response_json)
                    if 'error' in resp:
                        print(f"  Ответ: ОШИБКА - {resp.get('error', 'N/A')}")
                    elif 'content' in resp:
                        content = resp.get('content', '')
                        if not content:
                            print(f"  Ответ: content пустой")
                        else:
                            print(f"  Ответ: {content[:100]}...")
                    else:
                        print(f"  Ответ: {response_json[:150]}...")
                except:
                    print(f"  Ответ: {response_json[:150]}...")
        
        # Статистика по провайдерам
        print("\n--- Статистика по провайдерам ---")
        cursor.execute("""
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
            ORDER BY total DESC
        """)
        rows = cursor.fetchall()
        print(f"{'Provider':<15} {'Total':<10} {'Errors 500':<12}")
        print("-" * 40)
        for row in rows:
            print(f"{row['provider']:<15} {row['total']:<10} {row['errors_500']:<12}")
        
        # Анализ типов ошибок
        print("\n--- Анализ типов ошибок в response_json ---")
        cursor.execute("""
            SELECT 
                id,
                CASE 
                    WHEN response_json = '' OR response_json IS NULL THEN 'пустой'
                    WHEN response_json LIKE '%"error"%' THEN 'содержит error'
                    WHEN response_json LIKE '%"content":""%' THEN 'content пустой'
                    WHEN response_json = '{"content":"","status":500}' THEN 'стандартная ошибка 500'
                    ELSE 'другое'
                END as response_type,
                substr(response_json, 1, 150) as response_preview
            FROM request_logs
            WHERE status_code = 500
            ORDER BY created_at DESC
            LIMIT 10
        """)
        rows = cursor.fetchall()
        for row in rows:
            print(f"ID {row['id']}: {row['response_type']}")
            if row['response_preview']:
                print(f"  {row['response_preview']}")
        
        conn.close()
        print("\n=== Анализ завершен ===")
        
    except sqlite3.Error as e:
        print(f"❌ Ошибка SQLite: {e}")
        sys.exit(1)
    except Exception as e:
        print(f"❌ Ошибка: {e}")
        sys.exit(1)

if __name__ == "__main__":
    analyze_logs()

