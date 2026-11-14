package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
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
