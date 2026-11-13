package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	)

type Pattern struct {
	ID          string
	Severity    string
	Description string
	Regex       *regexp.Regexp
}

type Finding struct {
	File     string
	Line     int
	Pattern  Pattern
	Code     string
}

var patterns = []Pattern{
	{
		ID:          "EXEC-SHELL",
		Severity:    "HIGH",
		Description: "exec.Command with bash/sh",
		Regex:       regexp.MustCompile(`exec\.Command\("(bash|sh)"`),
	},
	{
		ID:          "ERROR-IGNORE",
		Severity:    "MED",
		Description: "Error ignored",
		Regex:       regexp.MustCompile(`_\s*=\s*\w+\(`),
	},
	{
		ID:          "TODO-FIXME",
		Severity:    "LOW",
		Description: "TODO/FIXME comment",
		Regex:       regexp.MustCompile(`(TODO|FIXME|HACK|XXX)`),
	},
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: qwen-analyzer <scan|patterns>")
		os.Exit(1)
	}
	switch os.Args[1] {
	case "scan":
		if len(os.Args) < 3 {
			fmt.Println("Usage: qwen-analyzer scan <dir>")
			os.Exit(1)
		}
		scanDir(os.Args[2])
	case "patterns":
		for _, p := range patterns {
			fmt.Printf("[%s] %s: %s\n", p.Severity, p.ID, p.Description)
		}
	}
}

func scanDir(dir string) {
	var findings []Finding
	filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		f, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer f.Close()
		sc := bufio.NewScanner(f)
		ln := 0
		for sc.Scan() {
			ln++
			line := sc.Text()
			for _, p := range patterns {
				if p.Regex.MatchString(line) && !strings.HasPrefix(strings.TrimSpace(line), "//") {
					findings = append(findings, Finding{File: path, Line: ln, Pattern: p, Code: strings.TrimSpace(line)})
				}
			}
		}
		return nil
	})
	fmt.Printf("Found %d issues\n", len(findings))
	for _, f := range findings {
		fmt.Printf("  %s:%d [%s] %s\n", f.File, f.Line, f.Pattern.ID, f.Code)
	}
}
