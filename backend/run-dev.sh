#!/bin/bash

# Скрипт для запуска backend в режиме разработки с отключенной проверкой TLS

export GIGACHAT_SKIP_TLS_VERIFY=true
go run .

