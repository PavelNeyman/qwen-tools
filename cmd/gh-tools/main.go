package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// --- Configuration ---

var (
	ghToken = envOr("GH_TOKEN", envOr("QWEN_GH_TOKEN", envOr("GITHUB_TOKEN", "")))
	ghRepo  = envOr("GH_REPO", "PavelNeyman/qwen-tools")
	apiBase = "https://api.github.com/repos/" + ghRepo
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// --- GitHub API client with retry/backoff ---

type fileContent struct {
	Content string `json:"content"`
	SHA     string `json:"sha"`
	Size    int    `json:"size"`
	Name    string `json:"name"`
	Path    string `json:"path"`
}

type updatePayload struct {
	Message string `json:"message"`
	Content string `json:"content"`
	SHA     string `json:"sha,omitempty"`
}

type updateResponse struct {
	Content fileContent `json:"content"`
	Commit  struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

func apiRequest(method, path string, payload any) ([]byte, int, error) {
	url := apiBase + path
	var body io.Reader
	if payload != nil {
		data, err := json.Marshal(payload)
		if err != nil {
			return nil, 0, err
		}
		body = bytes.NewBuffer(data)
	}

	req, err := http.NewRequest(method, url, body)
	if err != nil {
		return nil, 0, err
	}
	req.Header.Set("Authorization", "Bearer "+ghToken)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("User-Agent", "gh-tools/1.0")
	if payload != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 30 * time.Second}
	backoff := time.Second
	for attempt := 0; attempt < 6; attempt++ {
		resp, err := client.Do(req)
		if err != nil {
			if attempt < 5 {
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
			return nil, 0, err
		}
		defer resp.Body.Close()
		data, _ := io.ReadAll(resp.Body)
		if resp.StatusCode == 403 || resp.StatusCode == 429 {
			if retryAfter := resp.Header.Get("Retry-After"); retryAfter != "" {
				if secs, err := fmt.Sscanf(retryAfter, "%d", new(int)); err == nil {
					time.Sleep(time.Duration(secs) * time.Second)
					continue
				}
			}
			if attempt < 5 {
				time.Sleep(backoff)
				backoff *= 2
				continue
			}
		}
		return data, resp.StatusCode, nil
	}
	return nil, 0, fmt.Errorf("max retries exceeded")
}

// --- File operations ---

func getFile(path string) (string, string, error) {
	data, status, err := apiRequest("GET", "/contents/"+path, nil)
	if err != nil {
		return "", "", err
	}
	if status != 200 {
		return "", "", fmt.Errorf("HTTP %d: %s", status, string(data))
	}
	var f fileContent
	if err := json.Unmarshal(data, &f); err != nil {
		return "", "", err
	}
	decoded, err := base64.StdEncoding.DecodeString(f.Content)
	if err != nil {
		return "", "", err
	}
	return string(decoded), f.SHA, nil
}

func getFileSHA(path string) (string, error) {
	data, status, err := apiRequest("GET", "/contents/"+path, nil)
	if err != nil {
		return "", err
	}
	if status != 200 {
		return "", fmt.Errorf("HTTP %d", status)
	}
	var f fileContent
	json.Unmarshal(data, &f)
	return f.SHA, nil
}

func putFile(path, content, message string) (*updateResponse, error) {
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	payload := updatePayload{Message: message, Content: encoded}
	data, status, err := apiRequest("PUT", "/contents/"+path, payload)
	if err != nil {
		return nil, err
	}
	if status != 200 && status != 201 {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(data))
	}
	var resp updateResponse
	json.Unmarshal(data, &resp)
	return &resp, nil
}

func putFileFromLocal(repoPath, localPath, message string) (*updateResponse, error) {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, fmt.Errorf("read local: %w", err)
	}
	return putFile(repoPath, string(data), message)
}

func updateFile(path, content, message string) (*updateResponse, error) {
	sha, err := getFileSHA(path)
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	payload := updatePayload{Message: message, Content: encoded, SHA: sha}
	data, status, err := apiRequest("PUT", "/contents/"+path, payload)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(data))
	}
	var resp updateResponse
	json.Unmarshal(data, &resp)
	return &resp, nil
}

func patchFile(path, oldText, newText, message string) (*updateResponse, error) {
	content, _, err := getFile(path)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(content, oldText) {
		return nil, fmt.Errorf("pattern not found in %s", path)
	}
	newContent := strings.Replace(content, oldText, newText, 1)
	return updateFile(path, newContent, message)
}

func appendFile(path, text, message string) (*updateResponse, error) {
	content, _, err := getFile(path)
	if err != nil {
		return nil, err
	}
	return updateFile(path, content+text, message)
}

func deleteFile(path, message string) error {
	sha, err := getFileSHA(path)
	if err != nil {
		return err
	}
	payload := updatePayload{Message: message, SHA: sha}
	data, status, err := apiRequest("DELETE", "/contents/"+path, payload)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("HTTP %d: %s", status, string(data))
	}
	return nil
}

// --- Code example helpers ---

var cache = map[string][]string{}

func codeExample(path string, start, end int) (string, error) {
	if _, ok := cache[path]; !ok {
		content, _, err := getFile(path)
		if err != nil {
			return "", err
		}
		cache[path] = strings.Split(content, "\n")
	}
	lines := cache[path]
	if start == 0 && end == 0 {
		return strings.Join(lines, "\n"), nil
	}
	if start < 1 {
		start = 1
	}
	if end > len(lines) {
		end = len(lines)
	}
	return strings.Join(lines[start-1:end], "\n"), nil
}

func codeExampleBetween(path, startPat, endPat string) (string, error) {
	content, err := codeExample(path, 0, 0)
	if err != nil {
		return "", err
	}
	startRe, err := regexp.Compile(startPat)
	if err != nil {
		return "", err
	}
	endRe, err := regexp.Compile(endPat)
	if err != nil {
		return "", err
	}
	var out []string
	on := false
	for _, ln := range strings.Split(content, "\n") {
		if !on && startRe.MatchString(ln) {
			on = true
		}
		if on {
			out = append(out, ln)
		}
		if on && endRe.MatchString(ln) {
			break
		}
	}
	return strings.Join(out, "\n"), nil
}

// --- Doc builder ---

func fence(code, lang string) string {
	n := 3
	for strings.Contains(code, strings.Repeat("`", n)) {
		n++
	}
	fence := strings.Repeat("`", n)
	return fence + lang + "\n" + code + "\n" + fence
}

func addCodeBlock(doc, title, code, lang string) string {
	return doc + "\n\n### " + title + "\n\n" + fence(code, lang) + "\n"
}

// --- Fetch source via tarball ---

func fetchSource(dstDir string, patterns []string) (int, []string, error) {
	os.MkdirAll(dstDir, 0o755)
	url := "https://codeload.github.com/" + ghRepo + "/tar.gz/refs/heads/main"
	resp, err := http.Get(url)
	if err != nil {
		return 0, nil, err
	}
	defer resp.Body.Close()

	// tar.gz parsing would require archive/tar + compress/gzip
	// For now, return placeholder — full implementation below
	_ = resp
	return 0, nil, fmt.Errorf("tarball fetch requires archive/tar; use Python qwen_tools_v3.py for this")
}

// --- Repo meta ---

func getHeadSHA() (string, error) {
	data, status, err := apiRequest("GET", "/git/ref/heads/main", nil)
	if err != nil {
		return "", err
	}
	if status != 200 {
		return "", fmt.Errorf("HTTP %d", status)
	}
	var r struct {
		Object struct {
			SHA string `json:"sha"`
		} `json:"object"`
	}
	json.Unmarshal(data, &r)
	return r.Object.SHA, nil
}

func compareCommits(base, head string) ([]byte, error) {
	data, status, err := apiRequest("GET", "/compare/"+base+"..."+head, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("HTTP %d", status)
	}
	return data, nil
}

// --- CLI ---

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
  gh-tools head-sha                            Get HEAD commit SHA
  gh-tools compare <base> <head>               Compare two commits

Environment:
  GH_TOKEN, QWEN_GH_TOKEN, GITHUB_TOKEN   GitHub token
  GH_REPO                                 owner/repo (default: PavelNeyman/qwen-tools)`)
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
		content, sha, err := getFile(os.Args[2])
		if err != nil {
			fatal(err)
		}
		fmt.Printf("SHA: %s\nSize: %d bytes\n---\n%s\n", sha, len(content), content)

	case "put":
		if len(os.Args) < 5 {
			usage()
		}
		resp, err := putFileFromLocal(os.Args[2], os.Args[3], os.Args[4])
		if err != nil {
			fatal(err)
		}
		fmt.Printf("✓ Uploaded: %s (SHA: %s)\n", resp.Content.Path, resp.Content.SHA)

	case "update":
		if len(os.Args) < 5 {
			usage()
		}
		resp, err := putFileFromLocal(os.Args[2], os.Args[3], os.Args[4])
		if err != nil {
			fatal(err)
		}
		fmt.Printf("✓ Updated: %s (SHA: %s)\n", resp.Content.Path, resp.Content.SHA)

	case "delete":
		if len(os.Args) < 4 {
			usage()
		}
		if err := deleteFile(os.Args[2], os.Args[3]); err != nil {
			fatal(err)
		}
		fmt.Printf("✓ Deleted: %s\n", os.Args[2])

	case "patch":
		if len(os.Args) < 6 {
			usage()
		}
		resp, err := patchFile(os.Args[2], os.Args[3], os.Args[4], os.Args[5])
		if err != nil {
			fatal(err)
		}
		fmt.Printf("✓ Patched: %s (SHA: %s)\n", resp.Content.Path, resp.Content.SHA)

	case "append":
		if len(os.Args) < 5 {
			usage()
		}
		resp, err := appendFile(os.Args[2], os.Args[3], os.Args[4])
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
		content, err := codeExample(os.Args[2], start, end)
		if err != nil {
			fatal(err)
		}
		fmt.Println(content)

	case "head-sha":
		sha, err := getHeadSHA()
		if err != nil {
			fatal(err)
		}
		fmt.Println(sha)

	case "compare":
		if len(os.Args) < 4 {
			usage()
		}
		data, err := compareCommits(os.Args[2], os.Args[3])
		if err != nil {
			fatal(err)
		}
		fmt.Println(string(data))

	default:
		fmt.Fprintf(os.Stderr, "Unknown command: %s\n", cmd)
		usage()
	}
}

func fatal(err error) {
	fmt.Fprintf(os.Stderr, "Error: %v\n", err)
	os.Exit(1)
}

// Ensure filepath is used (for future extensions)
var _ = filepath.Join
