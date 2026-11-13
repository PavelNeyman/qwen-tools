package tools

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/PavelNeyman/qwen-tools/internal/github"
)

// --- Patch / Append ---

func PatchFile(path, oldText, newText, message string) (*github.UpdateResponse, error) {
	content, _, err := github.GetFile(path)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(content, oldText) {
		return nil, fmt.Errorf("pattern not found in %s", path)
	}
	newContent := strings.Replace(content, oldText, newText, 1)
	return github.UpdateFile(path, newContent, message)
}

func AppendFile(path, text, message string) (*github.UpdateResponse, error) {
	content, _, err := github.GetFile(path)
	if err != nil {
		return nil, err
	}
	return github.UpdateFile(path, content+text, message)
}

// --- Code example extraction ---

var cache = map[string][]string{}

func CodeExample(path string, start, end int) (string, error) {
	if _, ok := cache[path]; !ok {
		content, _, err := github.GetFile(path)
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

func CodeExampleBetween(path, startPat, endPat string) (string, error) {
	content, err := CodeExample(path, 0, 0)
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

func Fence(code, lang string) string {
	n := 3
	for strings.Contains(code, strings.Repeat("`", n)) {
		n++
	}
	fence := strings.Repeat("`", n)
	return fence + lang + "\n" + code + "\n" + fence
}

func AddCodeBlock(doc, title, code, lang string) string {
	return doc + "\n\n### " + title + "\n\n" + Fence(code, lang) + "\n"
}

// --- Render template ---

func RenderTemplate(tpl string, vars map[string]string) string {
	out := tpl
	for k, v := range vars {
		out = strings.ReplaceAll(out, "{{"+k+"}}", v)
	}
	// {{FILE:path}} — include file content
	fileRe := regexp.MustCompile(`\{\{FILE:([^}]+)\}\}`)
	out = fileRe.ReplaceAllStringFunc(out, func(match string) string {
		m := fileRe.FindStringSubmatch(match)
		content, _, err := github.GetFile(m[1])
		if err != nil {
			return fmt.Sprintf("<!-- error: %v -->", err)
		}
		return content
	})
	// {{CODEFILE:path:lang}} — include as fenced code block
	codeRe := regexp.MustCompile(`\{\{CODEFILE:([^:}]+)(?::([^}]+))?\}\}`)
	out = codeRe.ReplaceAllStringFunc(out, func(match string) string {
		m := codeRe.FindStringSubmatch(match)
		content, _, err := github.GetFile(m[1])
		if err != nil {
			return fmt.Sprintf("<!-- error: %v -->", err)
		}
		lang := m[2]
		if lang == "" {
			lang = "text"
		}
		return Fence(content, lang)
	})
	return out
}
