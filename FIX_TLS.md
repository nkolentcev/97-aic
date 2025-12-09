# Исправление ошибки TLS для GigaChat

## Проблема

При запросах к GigaChat возникает ошибка:
```
tls: failed to verify certificate: x509: certificate signed by unknown authority
```

Это происходит потому, что GigaChat использует сертификат НУЦ Минцифры, который не установлен в системе по умолчанию.

## Решение 1: Установка сертификата (рекомендуется)

### Ubuntu/Debian:

```bash
# Скачайте сертификат
sudo wget https://www.gosuslugi.ru/crt/rootca.cer -O /usr/local/share/ca-certificates/russian_trusted_root_ca.crt

# Обновите список сертификатов
sudo update-ca-certificates

# Перезапустите backend
```

### Проверка:

```bash
# Проверьте, что сертификат установлен
ls -la /usr/local/share/ca-certificates/russian_trusted_root_ca.crt
```

## Решение 2: Временное отключение проверки TLS (только для разработки!)

**ВНИМАНИЕ:** Это небезопасно! Используйте только для разработки и тестирования.

### Вариант A: Через переменную окружения

```bash
export GIGACHAT_SKIP_TLS_VERIFY=true
cd backend
go run .
```

### Вариант B: Использовать скрипт run-dev.sh

```bash
cd backend
./run-dev.sh
```

### Вариант C: Добавить в config.yaml (если поддерживается)

Пока не реализовано, но можно добавить в будущем.

## Анализ ошибок

После исправления TLS, проверьте логи:

```bash
# Анализ логов из SQLite
python3 analyze-logs.py backend/data.db

# Или через bash (если установлен sqlite3)
./analyze-logs.sh backend/data.db
```

## Проверка работы

1. Установите сертификат или установите `GIGACHAT_SKIP_TLS_VERIFY=true`
2. Перезапустите backend
3. Отправьте тестовый запрос:
   ```bash
   curl -X POST http://localhost:8080/api/v2/chat \
     -H "Content-Type: application/json" \
     -d '{
       "message": "Привет",
       "provider": "gigachat",
       "temperature": 0.7
     }'
   ```
4. Проверьте, что ошибка TLS исчезла

