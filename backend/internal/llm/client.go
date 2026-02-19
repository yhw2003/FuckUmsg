package llm

import (
	"bytes"
	"context"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"text/template"
	"time"

	"chat-assist-backend/internal/config"
)

type Client struct {
	baseURL    string
	apiKey     string
	model      string
	httpClient *http.Client
}

type Result struct {
	IsTodo     bool   `json:"is_todo"`
	Title      string `json:"title"`
	Detail     string `json:"detail"`
	DeadlineAt int64  `json:"deadline_at"`
}

type ResponseEnvelope struct {
	OutputText string `json:"output_text"`
	Output     []struct {
		Type    string `json:"type"`
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	} `json:"output"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
	} `json:"error"`
}

//go:embed prompts/*.tmpl
var promptFS embed.FS

var (
	systemPromptTemplate = template.Must(template.ParseFS(promptFS, "prompts/system.tmpl"))
	userPromptTemplate   = template.Must(template.ParseFS(promptFS, "prompts/user.tmpl"))
)

func NewClient(cfg config.Config) *Client {
	timeout := time.Duration(cfg.OpenAI.TimeoutSeconds) * time.Second
	return &Client{
		baseURL: strings.TrimRight(cfg.OpenAI.BaseURL, "/"),
		apiKey:  strings.TrimSpace(cfg.OpenAI.APIKey),
		model:   strings.TrimSpace(cfg.OpenAI.Model),
		httpClient: &http.Client{
			Timeout: timeout,
		},
	}
}

func (c *Client) Enabled() bool {
	return c.apiKey != "" && c.model != ""
}

func (c *Client) ExtractTodo(ctx context.Context, input string) (Result, error) {
	var result Result
	if !c.Enabled() {
		return result, errors.New("openai not configured")
	}

	expectedFormat := "IS_TODO: true|false\nTITLE: <待办标题或空>\nDETAIL: <待办详情或空>\nDEADLINE_AT: <Unix秒级时间戳，无截止或无法确定时为0>"
	systemPrompt, userPrompt, err := buildExtractPrompts(input, expectedFormat)
	if err != nil {
		return result, err
	}

	payload := map[string]interface{}{
		"model":        c.model,
		"instructions": systemPrompt,
	}
	payload["input"] = []interface{}{
		map[string]interface{}{
			"role": "user",
			"content": []map[string]interface{}{
				{
					"type": "input_text",
					"text": userPrompt,
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return result, err
	}

	endpoint := buildOpenAIURL(c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return result, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return result, err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		bodyText := strings.TrimSpace(string(bodyBytes))
		if bodyText != "" {
			return result, fmt.Errorf("openai status %d: %s", resp.StatusCode, bodyText)
		}
		return result, fmt.Errorf("openai status %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	bodyText := strings.TrimSpace(string(bodyBytes))

	output := ""
	var envelope ResponseEnvelope
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		output = extractTextFromSSE(bodyText, resp.Header.Get("Content-Type"))
		if output == "" {
			if bodyText != "" {
				return result, fmt.Errorf("invalid response body: %s", bodyText)
			}
			return result, errors.New("invalid response body")
		}
	} else {
		if envelope.Error != nil {
			return result, fmt.Errorf("openai error: %s", envelope.Error.Message)
		}
		output = strings.TrimSpace(envelope.OutputText)
		if output == "" {
			output = strings.TrimSpace(extractOutputText(envelope.Output))
		}
	}
	if output == "" {
		return result, errors.New("empty openai output")
	}

	parsed, err := parsePlainTodoOutput(output)
	if err != nil {
		return result, fmt.Errorf("invalid llm output: %s\nexpected format:\n%s", output, expectedFormat)
	}

	result = parsed

	return result, nil
}

func buildOpenAIURL(base string) string {
	trimmed := strings.TrimRight(base, "/")
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed + "/responses"
	}
	return trimmed + "/v1/responses"
}

func buildExtractPrompts(input string, expectedFormat string) (string, string, error) {
	systemPrompt, err := renderPrompt(systemPromptTemplate, nil)
	if err != nil {
		return "", "", err
	}
	userPrompt, err := renderPrompt(userPromptTemplate, map[string]string{
		"ExpectedFormat": expectedFormat,
		"Input":          input,
	})
	if err != nil {
		return "", "", err
	}
	return systemPrompt, userPrompt, nil
}

func renderPrompt(tmpl *template.Template, data interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, data); err != nil {
		return "", fmt.Errorf("render prompt failed: %w", err)
	}
	return buf.String(), nil
}

func extractOutputText(outputs []struct {
	Type    string `json:"type"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
}) string {
	for _, output := range outputs {
		for _, content := range output.Content {
			if content.Text != "" {
				return content.Text
			}
		}
	}
	return ""
}

func extractTextFromSSE(body string, contentType string) string {
	lowerType := strings.ToLower(contentType)
	if !strings.Contains(lowerType, "text/event-stream") && !strings.Contains(body, "event:") {
		return ""
	}
	var builder strings.Builder
	lines := strings.Split(body, "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" || data == "[DONE]" {
			continue
		}
		var payload map[string]interface{}
		if err := json.Unmarshal([]byte(data), &payload); err != nil {
			continue
		}
		if delta, ok := payload["delta"].(string); ok {
			builder.WriteString(delta)
			continue
		}
		if text, ok := payload["text"].(string); ok {
			builder.WriteString(text)
			continue
		}
		if outputText, ok := payload["output_text"].(string); ok {
			builder.WriteString(outputText)
			continue
		}
		if response, ok := payload["response"].(map[string]interface{}); ok {
			if outputText, ok := response["output_text"].(string); ok {
				builder.WriteString(outputText)
				continue
			}
		}
	}
	return strings.TrimSpace(builder.String())
}

func parsePlainTodoOutput(text string) (Result, error) {
	var result Result
	hasTodo := false
	hasTitle := false
	hasDetail := false

	lines := strings.Split(text, "\n")
	for _, raw := range lines {
		line := strings.TrimSpace(raw)
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "```") {
			continue
		}
		key, value, ok := splitKV(line)
		if !ok {
			continue
		}
		switch strings.ToUpper(key) {
		case "IS_TODO":
			value = strings.ToLower(strings.TrimSpace(value))
			if value == "true" {
				result.IsTodo = true
			} else if value == "false" {
				result.IsTodo = false
			} else {
				return result, fmt.Errorf("invalid is_todo value: %s", value)
			}
			hasTodo = true
		case "TITLE":
			result.Title = strings.TrimSpace(value)
			hasTitle = true
		case "DETAIL":
			result.Detail = strings.TrimSpace(value)
			hasDetail = true
		case "DEADLINE_AT":
			deadlineText := strings.TrimSpace(value)
			if deadlineText == "" {
				result.DeadlineAt = 0
				continue
			}
			deadlineAt, err := strconv.ParseInt(deadlineText, 10, 64)
			if err != nil || deadlineAt < 0 {
				result.DeadlineAt = 0
				continue
			}
			result.DeadlineAt = deadlineAt
		}
	}

	if !hasTodo || !hasTitle || !hasDetail {
		return result, errors.New("missing required fields")
	}
	if result.IsTodo && result.Title == "" {
		return result, errors.New("title is empty for todo")
	}
	if !result.IsTodo {
		result.Title = ""
		result.Detail = ""
		result.DeadlineAt = 0
	}
	return result, nil
}

func splitKV(line string) (string, string, bool) {
	idx := strings.IndexRune(line, ':')
	alt := strings.IndexRune(line, '：')
	if idx == -1 || (alt != -1 && alt < idx) {
		idx = alt
	}
	if idx == -1 {
		return "", "", false
	}
	key := strings.TrimSpace(line[:idx])
	value := strings.TrimSpace(line[idx+1:])
	if key == "" {
		return "", "", false
	}
	return key, value, true
}
