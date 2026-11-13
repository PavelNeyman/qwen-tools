package github

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"time"
)

// Configuration
var (
	Token = envOr("GH_TOKEN", envOr("QWEN_GH_TOKEN", envOr("GITHUB_TOKEN", "")))
	Repo  = envOr("GH_REPO", "PavelNeyman/qwen-tools")
	APIBase = "https://api.github.com/repos/" + Repo
)

func envOr(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// --- Types ---

type FileContent struct {
	Content string `json:"content"`
	SHA     string `json:"sha"`
	Size    int    `json:"size"`
	Name    string `json:"name"`
	Path    string `json:"path"`
}

type UpdatePayload struct {
	Message string `json:"message"`
	Content string `json:"content"`
	SHA     string `json:"sha,omitempty"`
}

type UpdateResponse struct {
	Content FileContent `json:"content"`
	Commit  struct {
		SHA string `json:"sha"`
	} `json:"commit"`
}

// --- Low-level API with retry/backoff ---

func Request(method, path string, payload any) ([]byte, int, error) {
	url := APIBase + path
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
	req.Header.Set("Authorization", "Bearer "+Token)
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
				var secs int
				if _, err := fmt.Sscanf(retryAfter, "%d", &secs); err == nil {
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

func GetFile(path string) (string, string, error) {
	data, status, err := Request("GET", "/contents/"+path, nil)
	if err != nil {
		return "", "", err
	}
	if status != 200 {
		return "", "", fmt.Errorf("HTTP %d: %s", status, string(data))
	}
	var f FileContent
	if err := json.Unmarshal(data, &f); err != nil {
		return "", "", err
	}
	decoded, err := base64.StdEncoding.DecodeString(f.Content)
	if err != nil {
		return "", "", err
	}
	return string(decoded), f.SHA, nil
}

func GetFileSHA(path string) (string, error) {
	data, status, err := Request("GET", "/contents/"+path, nil)
	if err != nil {
		return "", err
	}
	if status != 200 {
		return "", fmt.Errorf("HTTP %d", status)
	}
	var f FileContent
	json.Unmarshal(data, &f)
	return f.SHA, nil
}

func PutFile(path, content, message string) (*UpdateResponse, error) {
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	payload := UpdatePayload{Message: message, Content: encoded}
	data, status, err := Request("PUT", "/contents/"+path, payload)
	if err != nil {
		return nil, err
	}
	if status != 200 && status != 201 {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(data))
	}
	var resp UpdateResponse
	json.Unmarshal(data, &resp)
	return &resp, nil
}

func PutFileFromLocal(repoPath, localPath, message string) (*UpdateResponse, error) {
	data, err := os.ReadFile(localPath)
	if err != nil {
		return nil, fmt.Errorf("read local: %w", err)
	}
	return PutFile(repoPath, string(data), message)
}

func UpdateFile(path, content, message string) (*UpdateResponse, error) {
	sha, err := GetFileSHA(path)
	if err != nil {
		return nil, err
	}
	encoded := base64.StdEncoding.EncodeToString([]byte(content))
	payload := UpdatePayload{Message: message, Content: encoded, SHA: sha}
	data, status, err := Request("PUT", "/contents/"+path, payload)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("HTTP %d: %s", status, string(data))
	}
	var resp UpdateResponse
	json.Unmarshal(data, &resp)
	return &resp, nil
}

func DeleteFile(path, message string) error {
	sha, err := GetFileSHA(path)
	if err != nil {
		return err
	}
	payload := UpdatePayload{Message: message, SHA: sha}
	data, status, err := Request("DELETE", "/contents/"+path, payload)
	if err != nil {
		return err
	}
	if status != 200 {
		return fmt.Errorf("HTTP %d: %s", status, string(data))
	}
	return nil
}

// --- Repo meta ---

func GetHeadSHA() (string, error) {
	data, status, err := Request("GET", "/git/ref/heads/main", nil)
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

func CompareCommits(base, head string) ([]byte, error) {
	data, status, err := Request("GET", "/compare/"+base+"..."+head, nil)
	if err != nil {
		return nil, err
	}
	if status != 200 {
		return nil, fmt.Errorf("HTTP %d", status)
	}
	return data, nil
}
