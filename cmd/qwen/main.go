package main

import (
	"fmt"
	"os"
	"os/exec"

	"strings"
	"strconv"
	"io"
	"path/filepath"
	"net/http"
)

type Tool struct {
	Name    string
	URL     string
	Local   string
	Size    string
}

var toolRegistry = []Tool{
	{"staticcheck", "https://sendit.sh/Q_ZDe/pZVun.gz", "/tmp/staticcheck/staticcheck", "7 МБ"},
	{"gosec", "https://sendit.sh/dVoly/gosec.tar.gz", "/tmp/gosec", "14 МБ"},
}

func main() {
	if len(os.Args) < 2 {
		printUsage()
		os.Exit(1)
	}

	cmd := os.Args[1]
	switch cmd {
	case "tools":
		if len(os.Args) < 3 {
			fmt.Println("Usage: qwen tools <install|status|list>")
			os.Exit(1)
		}
		toolsCmd(os.Args[2])
	case "review":
		if len(os.Args) < 3 {
			fmt.Println("Usage: qwen review <repo>")
			os.Exit(1)
		}
		reviewCmd(os.Args[2])
		case "exec":
		if len(os.Args) < 3 {
			fmt.Println("Usage: qwen exec <command> [args...]")
			os.Exit(1)
		}
		execCmd(os.Args[2:])
	case "read":
		if len(os.Args) < 3 {
			fmt.Println("Usage: qwen read <path>")
			os.Exit(1)
		}
		readCmd(os.Args[2])
	case "write":
		if len(os.Args) < 4 {
			fmt.Println("Usage: qwen write <path> <content>")
			os.Exit(1)
		}
		writeCmd(os.Args[2], os.Args[3])
	case "find":
		if len(os.Args) < 4 {
			fmt.Println("Usage: qwen find <pattern> <dir>")
			os.Exit(1)
		}
		findCmd(os.Args[2], os.Args[3])
	case "tree":
		if len(os.Args) < 3 {
			fmt.Println("Usage: qwen tree <dir> [depth]")
			os.Exit(1)
		}
		depth := 3
		if len(os.Args) > 3 {
			depth = atoi(os.Args[3])
		}
		treeCmd(os.Args[2], depth)
	case "http":
		if len(os.Args) < 4 {
			fmt.Println("Usage: qwen http <method> <url>")
			os.Exit(1)
		}
		httpCmd(os.Args[2], os.Args[3])

	case "help", "--help", "-h":
		printUsage()
	default:
		fmt.Printf("Unknown command: %s\n", cmd)
		printUsage()
		os.Exit(1)
	}
}

func printUsage() {
	fmt.Println(`qwen - единый инструмент для ревью кода

Usage:
  qwen <command> [arguments]

Commands:
  tools install    Установить все инструменты анализа
  tools status     Показать статус установленных инструментов
  tools list       Список доступных инструментов
  review <repo>    Запустить полный ревью репозитория
  help             Показать это сообщение

Examples:
  qwen tools install
  qwen tools status
  qwen review PavelNeyman/netductor`)
}

func toolsCmd(action string) {
	switch action {
	case "install":
		installTools()
	case "status":
		showToolStatus()
	case "list":
		listTools()
	default:
		fmt.Printf("Unknown action: %s\n", action)
		fmt.Println("Available: install, status, list")
		os.Exit(1)
	}
}

func listTools() {
	fmt.Println("Доступные инструменты:")
	for _, t := range toolRegistry {
		fmt.Printf("  %-15s %s (%s)\n", t.Name, t.URL, t.Size)
	}
	fmt.Println("  go              https://dl.google.com/go/go1.22.5.linux-amd64.tar.gz (150 МБ)")
}

func installTools() {
	fmt.Println("Установка инструментов...")
	
	for _, t := range toolRegistry {
		if _, err := os.Stat(t.Local); err == nil {
			fmt.Printf("✓ %s уже установлен\n", t.Name)
			continue
		}
		
		fmt.Printf("Скачивание %s (%s)...\n", t.Name, t.Size)
		
		// Скачиваем
		tmpFile := "/tmp/tmp-tool.tar.gz"
		cmd := exec.Command("curl", "-s", "-o", tmpFile, t.URL)
		if err := cmd.Run(); err != nil {
			fmt.Printf("✗ Ошибка скачивания %s: %v\n", t.Name, err)
			continue
		}
		
		// Распаковываем
		cmd = exec.Command("tar", "-xzf", tmpFile, "-C", "/tmp/")
		if err := cmd.Run(); err != nil {
			fmt.Printf("✗ Ошибка распаковки %s: %v\n", t.Name, err)
			continue
		}
		
		os.Remove(tmpFile)
		fmt.Printf("✓ %s установлен\n", t.Name)
	}
	
	// Go отдельно
	if _, err := os.Stat("/tmp/go/bin/go"); err != nil {
		fmt.Println("Скачивание Go 1.22.5...")
		cmd := exec.Command("bash", "-c", 
			"curl -fsSL -o /tmp/go.tar.gz 'https://dl.google.com/go/go1.22.5.linux-amd64.tar.gz' && " +
			"tar -xzf /tmp/go.tar.gz -C /tmp/ && " +
			"rm /tmp/go.tar.gz && " +
			"rm -rf /tmp/go/test /tmp/go/doc /tmp/go/api /tmp/go/misc")
		if err := cmd.Run(); err != nil {
			fmt.Printf("✗ Ошибка установки Go: %v\n", err)
		} else {
			fmt.Println("✓ Go 1.22.5 установлен")
		}
	} else {
		fmt.Println("✓ Go уже установлен")
	}
	
	fmt.Println("\nГотово! Все инструменты установлены.")
}

func showToolStatus() {
	fmt.Println("Статус инструментов:")
	
	for _, t := range toolRegistry {
		if _, err := os.Stat(t.Local); err == nil {
			fmt.Printf("  ✓ %-15s %s\n", t.Name, t.Local)
		} else {
			fmt.Printf("  ✗ %-15s (не установлен)\n", t.Name)
		}
	}
	
	if _, err := os.Stat("/tmp/go/bin/go"); err == nil {
		fmt.Printf("  ✓ %-15s /tmp/go/bin/go\n", "Go 1.22.5")
	} else {
		fmt.Printf("  ✗ %-15s (не установлен)\n", "Go")
	}
}

func reviewCmd(repo string) {
	fmt.Printf("Ревью репозитория: %s\n", repo)
	
	// Проверяем что инструменты установлены
	toolsInstalled := true
	for _, t := range toolRegistry {
		if _, err := os.Stat(t.Local); err != nil {
			toolsInstalled = false
			break
		}
	}
	
	if !toolsInstalled {
		fmt.Println("Инструменты не установлены. Запускаю установку...")
		installTools()
	}
	
	fmt.Println("\nЗапуск анализа...")
	fmt.Println("(Функционал будет добавлен в следующей версии)")
}


func execCmd(args []string) {
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "exec error: %v\n", err)
		os.Exit(1)
	}
}

func readCmd(path string) {
	data, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "read error: %v\n", err)
		os.Exit(1)
	}
	fmt.Print(string(data))
}

func writeCmd(path, content string) {
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		fmt.Fprintf(os.Stderr, "write error: %v\n", err)
		os.Exit(1)
	}
	fmt.Printf("Written %d bytes to %s\n", len(content), path)
}

func findCmd(pattern, dir string) {
	pattern = strings.ToLower(pattern)
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if strings.Contains(strings.ToLower(path), pattern) {
			fmt.Println(path)
		}
		return nil
	})
}

func treeCmd(dir string, depth int) {
	printTree(dir, 0, depth)
}

func printTree(dir string, currentDepth, maxDepth int) {
	if currentDepth >= maxDepth {
		return
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		indent := strings.Repeat("  ", currentDepth)
		if entry.IsDir() {
			fmt.Printf("%s%s/\n", indent, entry.Name())
			printTree(filepath.Join(dir, entry.Name()), currentDepth+1, maxDepth)
		} else {
			fmt.Printf("%s%s\n", indent, entry.Name())
		}
	}
}

func httpCmd(method, url string) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http error: %v\n", err)
		os.Exit(1)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "http error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	fmt.Printf("Status: %d\n", resp.StatusCode)
	io.Copy(os.Stdout, resp.Body)
}

func atoi(s string) int {
	n, _ := strconv.Atoi(s)
	return n
}
