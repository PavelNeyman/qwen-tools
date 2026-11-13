#!/bin/bash
# qwen-tools - загрузка инструментов для статического анализа Go-кода
# Использует sendit.sh для персистентного хранения staticcheck и gosec

set -e

echo "=== Загрузка инструментов ==="
start_time=$(date +%s)

# 1. staticcheck (7 МБ с sendit.sh)
if [ ! -f /tmp/staticcheck/staticcheck ]; then
    echo "[1/4] Скачивание staticcheck..."
    curl -s -o /tmp/staticcheck.tar.gz https://sendit.sh/Q_ZDe/pZVun.gz
    tar -xzf /tmp/staticcheck.tar.gz -C /tmp/
    rm /tmp/staticcheck.tar.gz
    echo "  ✓ staticcheck готов"
else
    echo "[1/4] staticcheck уже загружен"
fi

# 2. gosec (14 МБ с sendit.sh)
if [ ! -f /tmp/gosec ]; then
    echo "[2/4] Скачивание gosec..."
    curl -s -o /tmp/gosec.tar.gz https://sendit.sh/dVoly/gosec.tar.gz
    tar -xzf /tmp/gosec.tar.gz -C /tmp/
    chmod +x /tmp/gosec
    rm /tmp/gosec.tar.gz
    echo "  ✓ gosec готов"
else
    echo "[2/4] gosec уже загружен"
fi

# 3. Go 1.22.5 (с dl.google.com)
if [ ! -f /tmp/go/bin/go ]; then
    echo "[3/4] Скачивание Go 1.22.5..."
    curl -fsSL -o /tmp/go.tar.gz https://dl.google.com/go/go1.22.5.linux-amd64.tar.gz
    tar -xzf /tmp/go.tar.gz -C /tmp/
    rm /tmp/go.tar.gz
    # Удаляем ненужные директории для экономии места
    rm -rf /tmp/go/test /tmp/go/doc /tmp/go/api /tmp/go/misc
    echo "  ✓ Go готов"
else
    echo "[3/4] Go уже загружен"
fi

# 4. Настройка окружения
echo "[4/4] Настройка окружения..."
mkdir -p /tmp/go-cache /tmp/gopath/pkg/mod /dev/shm/go-cache

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
echo "  staticcheck: /tmp/staticcheck/staticcheck -f json ./..."
echo "  gosec:       /tmp/gosec -fmt json -quiet ./..."
echo "  go:          /tmp/go/bin/go"
