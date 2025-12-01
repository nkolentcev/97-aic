# Руководство по развертыванию

## Подготовка сервера

### Требования

- Ubuntu/Debian сервер с SSH доступом
- Go 1.21+ (опционально, если собираетесь компилировать на сервере)
- Systemd для управления сервисом

### Установка на VPS

1. Подключитесь к серверу:
```bash
ssh user@your-server
```

2. Создайте директорию для приложения:
```bash
sudo mkdir -p /opt/gigachat-chat
sudo chown $USER:$USER /opt/gigachat-chat
```

3. Скопируйте файлы на сервер:
```bash
# С вашего локального компьютера
scp backend/server backend/config.yaml user@your-server:/opt/gigachat-chat/
```

4. Создайте systemd сервис:
```bash
sudo nano /etc/systemd/system/gigachat-chat.service
```

Содержимое файла:
```ini
[Unit]
Description=GigaChat Chat Server
After=network.target

[Service]
Type=simple
User=your-user
WorkingDirectory=/opt/gigachat-chat
ExecStart=/opt/gigachat-chat/server
Restart=always
RestartSec=5

[Install]
WantedBy=multi-user.target
```

5. Запустите сервис:
```bash
sudo systemctl daemon-reload
sudo systemctl enable gigachat-chat
sudo systemctl start gigachat-chat
```

6. Проверьте статус:
```bash
sudo systemctl status gigachat-chat
```

## Автоматический деплой через CI/CD

### Настройка Gitea Secrets

В настройках репозитория Gitea добавьте следующие secrets:

- `VPS_HOST` - IP адрес или домен вашего сервера
- `VPS_USER` - пользователь для SSH подключения
- `VPS_SSH_KEY` - приватный SSH ключ для доступа к серверу

### Workflow

При каждом push в ветку `main` автоматически:
1. Собирается frontend
2. Копируется в `backend/static/`
3. Собирается Go backend
4. Деплоится на VPS
5. Перезапускается сервис

## Ручной деплой

Используйте скрипт из директории `deploy/`:

```bash
export VPS_HOST="your-server.com"
export VPS_USER="user"
./deploy/deploy.sh
```

## Настройка Nginx (опционально)

Для работы через домен и HTTPS:

```nginx
server {
    listen 80;
    server_name your-domain.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_http_version 1.1;
        proxy_set_header Upgrade $http_upgrade;
        proxy_set_header Connection 'upgrade';
        proxy_set_header Host $host;
        proxy_cache_bypass $http_upgrade;
    }
}
```

## Обновление

Для обновления приложения:

1. Через CI/CD: просто сделайте push в `main`
2. Вручную: повторите шаги копирования файлов и перезапустите сервис:
```bash
sudo systemctl restart gigachat-chat
```

## Логи

Просмотр логов сервиса:
```bash
sudo journalctl -u gigachat-chat -f
```

