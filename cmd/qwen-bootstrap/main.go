package main

import (
	"compress/gzip"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	)

const (
	toolsRepo = "PavelNeyman/qwen-tools"
	cacheDir  = "/tmp/qwen-tools-cache"
)

type Tool struct {
	Name        string
	Description string
	Asset       string
	Version     string
}

var registry = []Tool{
	{"gh-tools", "GitHub API file operations", "gh-tools-linux-amd64", "v1.0.2"},
	{"qwen-analyzer", "Code analysis and review", "qwen-analyzer-linux-amd64", "v1.0.0"},
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	cmd := os.Args[1]
	os.MkdirAll(cacheDir, 0o755)
	switch cmd {
	case "list":
		listTools()
	case "install":
		if len(os.Args) < 3 {
			fmt.Println("Usage: qwen-bootstrap install <tool>")
			os.Exit(1)
		}
		installTool(os.Args[2])
	case "run":
		if len(os.Args) < 3 {
			fmt.Println("Usage: qwen-bootstrap run <tool> [args...]")
			os.Exit(1)
		}
		runTool(os.Args[2], os.Args[3:])
	case "update":
		updateAll()
	default:
		usage()
	}
}

func usage() {
	fmt.Println("Usage: qwen-bootstrap <list|install|run|update>")
	os.Exit(1)
}

func listTools() {
	fmt.Println("Available tools:")
	for _, t := range registry {
		installed := "no"
		if _, err := os.Stat(filepath.Join(cacheDir, t.Name)); err == nil {
			installed = "yes"
		}
		fmt.Printf("  [%s] %s - %s\n", installed, t.Name, t.Description)
	}
}

func installTool(name string) {
	var tool *Tool
	for _, t := range registry {
		if t.Name == name {
			tool = &t
			break
		}
	}
	if tool == nil {
		fmt.Fprintf(os.Stderr, "Unknown tool: %s\n", name)
		os.Exit(1)
	}
	binPath := filepath.Join(cacheDir, tool.Name)
	if _, err := os.Stat(binPath); err == nil {
		fmt.Printf("Already installed: %s\n", binPath)
		return
	}
	fmt.Printf("Downloading %s...\n", name)
	url := fmt.Sprintf("https://api.github.com/repos/%s/contents/%s.gz", toolsRepo, tool.Asset)
	req, _ := http.NewRequest("GET", url, nil)
	req.Header.Set("Accept", "application/vnd.github.v3.raw")
	if token := os.Getenv("GH_TOKEN"); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %v\n", err)
		os.Exit(1)
	}
	defer resp.Body.Close()
	data, _ := io.ReadAll(resp.Body)
	if strings.HasSuffix(tool.Asset, ".gz") {
		gr, err := gzip.NewReader(strings.NewReader(string(data)))
		if err == nil {
			decompressed, _ := io.ReadAll(gr)
			data = decompressed
		}
	}
	os.WriteFile(binPath, data, 0o755)
	fmt.Printf("Installed: %s (%d bytes)\n", binPath, len(data))
}

func runTool(name string, args []string) {
	binPath := filepath.Join(cacheDir, name)
	if _, err := os.Stat(binPath); os.IsNotExist(err) {
		installTool(name)
	}
	cmd := exec.Command(binPath, args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if err := cmd.Run(); err != nil {
		os.Exit(1)
	}
}

func updateAll() {
	for _, t := range registry {
		binPath := filepath.Join(cacheDir, t.Name)
		os.Remove(binPath)
		installTool(t.Name)
	}
}
