#!/bin/bash

# Скрипт для ручного деплоя на VPS

set -e

VPS_HOST="${VPS_HOST:-}"
VPS_USER="${VPS_USER:-root}"
DEPLOY_PATH="/opt/gigachat-chat"

if [ -z "$VPS_HOST" ]; then
  echo "Ошибка: VPS_HOST не установлен"
  exit 1
fi

echo "Сборка frontend..."
cd frontend
npm install
npm run build
cd ..

echo "Копирование frontend в backend/static..."
mkdir -p backend/static
cp -r frontend/dist/* backend/static/

echo "Сборка backend..."
cd backend
go build -o server .
cd ..

echo "Копирование файлов на сервер..."
scp backend/server backend/config.example.yaml ${VPS_USER}@${VPS_HOST}:${DEPLOY_PATH}/

echo "Перезапуск сервиса..."
ssh ${VPS_USER}@${VPS_HOST} "cd ${DEPLOY_PATH} && chmod +x server && sudo systemctl restart gigachat-chat || echo 'Сервис не найден, запустите вручную'"

echo "Деплой завершен!"

