package main

import (
	"fmt"
	"os"

	"github.com/PavelNeyman/qwen-tools/internal/github"
	"github.com/PavelNeyman/qwen-tools/internal/tools"
)

func usage() {
	fmt.Fprintln(os.Stderr, `gh-tools - GitHub file operations via API

Usage:
  gh-tools get <path>                          Read file content
  gh-tools put <path> <local-file> <message>   Upload local file
  gh-tools update <path> <local-file> <msg>    Update existing file
  gh-tools delete <path> <message>             Delete file
  gh-tools patch <path> <old> <new> <message>  Replace text in file
  gh-tools append <path> <text> <message>      Append text to file
  gh-tools example <path> [start] [end]        Extract lines from file
  gh-tools example-between <path> <start> <end> Extract between patterns
  gh-tools head-sha                            Get HEAD commit SHA
  gh-tools compare <base> <head>               Compare two commits
  gh-tools fence <lang>                        Read code from stdin, output fenced
  gh-tools render <template-file>              Render template with {{VAR}} substitution

Environment:
  GH_TOKEN, QWEN_GH_TOKEN, GITHUB_TOKEN   GitHub token
  GH_REPO                                 owner/repo (default: PavelNeyman/qwen-tools)`)
	os.Exit(1)
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

func main() {
	if len(os.Args) < 2 {
		usage()
	}

	cmd := os.Args[1]

	switch cmd {
	case "get":
		if len(os.Args) < 3 {
			usage()
		}
		content, sha, err := github.GetFile(os.Args[2])
		if err != nil {
			fatal(err)
		}
		fmt.Printf("SHA: %s\nSize: %d bytes\n---\n%s\n", sha, len(content), content)

	case "put":
		if len(os.Args) < 5 {
			usage()
		}
		resp, err := github.PutFileFromLocal(os.Args[2], os.Args[3], os.Args[4])
		if err != nil {
			fatal(err)
		}
		fmt.Printf("✓ Uploaded: %s (SHA: %s)\n", resp.Content.Path, resp.Content.SHA)

	case "update":
		if len(os.Args) < 5 {
			usage()
		}
		resp, err := github.PutFileFromLocal(os.Args[2], os.Args[3], os.Args[4])
		if err != nil {
			fatal(err)
		}
		fmt.Printf("✓ Updated: %s (SHA: %s)\n", resp.Content.Path, resp.Content.SHA)

	case "delete":
		if len(os.Args) < 4 {
			usage()
		}
		if err := github.DeleteFile(os.Args[2], os.Args[3]); err != nil {
			fatal(err)
		}
		fmt.Printf("✓ Deleted: %s\n", os.Args[2])

	case "patch":
		if len(os.Args) < 6 {
			usage()
		}
		resp, err := tools.PatchFile(os.Args[2], os.Args[3], os.Args[4], os.Args[5])
		if err != nil {
			fatal(err)
		}
		fmt.Printf("✓ Patched: %s (SHA: %s)\n", resp.Content.Path, resp.Content.SHA)

	case "append":
		if len(os.Args) < 5 {
			usage()
		}
		resp, err := tools.AppendFile(os.Args[2], os.Args[3], os.Args[4])
		if err != nil {
			fatal(err)
		}
		fmt.Printf("✓ Appended: %s (SHA: %s)\n", resp.Content.Path, resp.Content.SHA)

	case "example":
		if len(os.Args) < 3 {
			usage()
		}
		start, end := 0, 0
		if len(os.Args) >= 4 {
			fmt.Sscanf(os.Args[3], "%d", &start)
		}
		if len(os.Args) >= 5 {
			fmt.Sscanf(os.Args[4], "%d", &end)
		}
		content, err := tools.CodeExample(os.Args[2], start, end)
		if err != nil {
			fatal(err)
		}
		fmt.Println(content)

	case "example-between":
		if len(os.Args) < 5 {
			usage()
		}
		content, err := tools.CodeExampleBetween(os.Args[2], os.Args[3], os.Args[4])
		if err != nil {
			fatal(err)
		}
		fmt.Println(content)

	case "head-sha":
		sha, err := github.GetHeadSHA()
		if err != nil {
			fatal(err)
		}
		fmt.Println(sha)

	case "compare":
		if len(os.Args) < 4 {
			usage()
		}
		data, err := github.CompareCommits(os.Args[2], os.Args[3])
		if err != nil {
			fatal(err)
		}
		fmt.Println(string(data))

	case "fence":
		lang := "go"
		if len(os.Args) >= 3 {
			lang = os.Args[2]
		}
		buf := make([]byte, 1024*1024)
		n, _ := os.Stdin.Read(buf)
		code := string(buf[:n])
		fmt.Print(tools.Fence(code, lang))

	case "render":
		if len(os.Args) < 3 {
			usage()
		}
		tpl, err := os.ReadFile(os.Args[2])
		if err != nil {
			fatal(err)
		}
		fmt.Print(tools.RenderTemplate(string(tpl), nil))

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		usage()
	}
}
