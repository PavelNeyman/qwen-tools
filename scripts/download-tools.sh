#!/bin/bash
set -e

REPO="PavelNeyman/qwen-tools"
BRANCH="main"
BASE_RAW="https://raw.githubusercontent.com/$REPO/$BRANCH"

echo "=== Загрузка qwen tools ==="
start_time=$(date +%s)

echo "[1/4] Чтение списка инструментов..."
curl -sL "$BASE_RAW/docs/tools-urls.txt" -o /tmp/tools-urls.txt
VERSION=$(grep "^version:" /tmp/tools-urls.txt | cut -d' ' -f2)
echo "  Версия: $VERSION"

echo "[2/4] Попытка загрузки с sendit.sh..."
SENDIT_AVAILABLE=false
if curl -sL "$BASE_RAW/docs/sendit-urls.txt" -o /tmp/sendit-urls.txt 2>/dev/null; then
    SENDIT_AVAILABLE=true
    echo "  ✓ sendit.sh URLs доступны"
else
    echo "  ✗ sendit.sh URLs недоступны, используем репу"
fi

echo "[3/4] Загрузка инструментов..."

download_tool() {
    local name=$1
    local local_path=$2
    
    if [ -f "$local_path" ]; then
        echo "  ✓ $name уже загружен"
        return 0
    fi
    
    if [ "$SENDIT_AVAILABLE" = true ]; then
        url=$(grep "^$name:" /tmp/sendit-urls.txt | cut -d' ' -f2)
        if [ -n "$url" ]; then
            echo "  Загрузка $name с sendit.sh..."
            if curl -sL -o "$local_path" "$url" --max-time 30; then
                chmod +x "$local_path"
                echo "  ✓ $name загружен с sendit.sh"
                return 0
            fi
            rm -f "$local_path"
        fi
    fi
    
    repo_path=$(grep "^$name:" /tmp/tools-urls.txt | cut -d' ' -f2)
    if [ -n "$repo_path" ]; then
        echo "  Загрузка $name из репо..."
        if curl -sL -o "$local_path" "$BASE_RAW/$repo_path" --max-time 60; then
            chmod +x "$local_path"
            echo "  ✓ $name загружен из репо"
            return 0
        fi
    fi
    
    echo "  ✗ Не удалось загрузить $name"
    return 1
}

TOOLS=(
    "qwen:/tmp/qwen"
    "staticcheck:/tmp/staticcheck"
    "gosec:/tmp/gosec"
    "errcheck:/tmp/errcheck"
    "govulncheck:/tmp/govulncheck"
    "golangci-lint:/tmp/golangci-lint"
    "goimports:/tmp/goimports"
    "revive:/tmp/revive"
    "deadcode:/tmp/deadcode"
    "ineffassign:/tmp/ineffassign"
    "unconvert:/tmp/unconvert"
    "bodyclose:/tmp/bodyclose"
)

for tool_spec in "${TOOLS[@]}"; do
    name=${tool_spec%:*}
    path=${tool_spec#*:}
    download_tool "$name" "$path"
done

echo "[4/4] Загрузка Go..."
if [ ! -f /tmp/go/bin/go ]; then
    curl -fsSL -o /tmp/go.tar.gz 'https://dl.google.com/go/go1.22.5.linux-amd64.tar.gz'
    tar -xzf /tmp/go.tar.gz -C /tmp/
    rm /tmp/go.tar.gz
    rm -rf /tmp/go/test /tmp/go/doc /tmp/go/api /tmp/go/misc
    echo "  ✓ Go 1.22.5 установлен"
else
    echo "  ✓ Go уже установлен"
fi

mkdir -p /dev/shm/go-cache /tmp/gopath/pkg/mod

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
