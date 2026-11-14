#!/bin/bash
# qwen-tools v1.3.0 - загрузка инструментов для статического анализа Go-кода
# Использует GitHub API для персистентного хранения

set -e

echo "=== Загрузка qwen tools v1.3.0 ==="
start_time=$(date +%s)

# 1. Скачиваем qwen бинарник через GitHub API
if [ ! -f /tmp/qwen ]; then
    echo "[1/3] Скачивание qwen бинарника через API..."
    curl -s -H "Authorization: Bearer $GH_TOKEN" \
         -H "Accept: application/vnd.github.v3.raw" \
         "https://api.github.com/repos/PavelNeyman/qwen-tools/contents/qwen-linux-amd64.gz" \
         | gunzip > /tmp/qwen
    chmod +x /tmp/qwen
    echo "  ✓ qwen v1.3.0 готов"
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
echo "Новые команды v1.3.0:"
echo "  qwen exec <cmd> [args]     - выполнить команду"
echo "  qwen read <path>           - прочитать файл"
echo "  qwen write <path> <content>- записать файл"
echo "  qwen find <pattern> <dir>  - найти файлы"
echo "  qwen tree <dir> [depth]    - дерево директорий"
echo "  qwen http <method> <url>   - HTTP запрос"
echo ""
echo "Старые команды:"
echo "  qwen tools status          - статус инструментов"
echo "  qwen review <repo>         - полный ревью"
