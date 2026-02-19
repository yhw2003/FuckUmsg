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

type RelativeDateInfo struct {
	BaseDate        string `json:"base_date"`
	OffsetDays      int    `json:"offset_days"`
	WeekExpr        string `json:"week_expr"`
	ResolvedDate    string `json:"resolved_date"`
	RawReasoningTag string `json:"raw_reasoning_tag"`
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
	classifySystemTemplate            = template.Must(template.ParseFS(promptFS, "prompts/classify_system.tmpl"))
	classifyUserTemplate              = template.Must(template.ParseFS(promptFS, "prompts/classify_user.tmpl"))
	summarizeSystemTemplate           = template.Must(template.ParseFS(promptFS, "prompts/summarize_system.tmpl"))
	summarizeUserTemplate             = template.Must(template.ParseFS(promptFS, "prompts/summarize_user.tmpl"))
	relativeDateExtractSystemTemplate = template.Must(template.ParseFS(promptFS, "prompts/relative_date_extract_system.tmpl"))
	relativeDateExtractUserTemplate   = template.Must(template.ParseFS(promptFS, "prompts/relative_date_extract_user.tmpl"))
	relativeDateRetrySystemTemplate   = template.Must(template.ParseFS(promptFS, "prompts/relative_date_retry_system.tmpl"))
	relativeDateRetryUserTemplate     = template.Must(template.ParseFS(promptFS, "prompts/relative_date_retry_user.tmpl"))
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

func (c *Client) ClassifyTodo(ctx context.Context, input string) (bool, error) {
	if !c.Enabled() {
		return false, errors.New("openai not configured")
	}
	systemPrompt, err := renderPrompt(classifySystemTemplate, nil)
	if err != nil {
		return false, err
	}
	userPrompt, err := renderPrompt(classifyUserTemplate, map[string]string{"Input": input})
	if err != nil {
		return false, err
	}
	output, err := c.call(ctx, systemPrompt, userPrompt)
	if err != nil {
		return false, err
	}
	return parseClassifyOutput(output)
}

func (c *Client) SummarizeTodo(ctx context.Context, input string) (string, string, error) {
	if !c.Enabled() {
		return "", "", errors.New("openai not configured")
	}
	systemPrompt, err := renderPrompt(summarizeSystemTemplate, nil)
	if err != nil {
		return "", "", err
	}
	userPrompt, err := renderPrompt(summarizeUserTemplate, map[string]string{"Input": input})
	if err != nil {
		return "", "", err
	}
	output, err := c.call(ctx, systemPrompt, userPrompt)
	if err != nil {
		return "", "", err
	}
	return parseSummarizeOutput(output)
}

func (c *Client) ExtractRelativeDate(ctx context.Context, input string, nowContext string) (RelativeDateInfo, error) {
	if !c.Enabled() {
		return RelativeDateInfo{}, errors.New("openai not configured")
	}
	systemPrompt, err := renderPrompt(relativeDateExtractSystemTemplate, nil)
	if err != nil {
		return RelativeDateInfo{}, err
	}
	userPrompt, err := renderPrompt(relativeDateExtractUserTemplate, map[string]string{
		"Input":      input,
		"NowContext": nowContext,
	})
	if err != nil {
		return RelativeDateInfo{}, err
	}
	output, err := c.call(ctx, systemPrompt, userPrompt)
	if err != nil {
		return RelativeDateInfo{}, err
	}
	return parseRelativeDateOutput(output)
}

func (c *Client) RetryRelativeDate(ctx context.Context, input string, nowContext string, classicDate int64, previousLLM RelativeDateInfo) (RelativeDateInfo, error) {
	if !c.Enabled() {
		return RelativeDateInfo{}, errors.New("openai not configured")
	}
	systemPrompt, err := renderPrompt(relativeDateRetrySystemTemplate, nil)
	if err != nil {
		return RelativeDateInfo{}, err
	}
	previousJSON, err := json.Marshal(previousLLM)
	if err != nil {
		return RelativeDateInfo{}, err
	}
	userPrompt, err := renderPrompt(relativeDateRetryUserTemplate, map[string]string{
		"Input":       input,
		"NowContext":  nowContext,
		"ClassicDate": strconv.FormatInt(classicDate, 10),
		"PreviousLLM": string(previousJSON),
	})
	if err != nil {
		return RelativeDateInfo{}, err
	}
	output, err := c.call(ctx, systemPrompt, userPrompt)
	if err != nil {
		return RelativeDateInfo{}, err
	}
	return parseRelativeDateOutput(output)
}

// ExtractTodo is kept as a compatibility wrapper.
func (c *Client) ExtractTodo(ctx context.Context, input string) (Result, error) {
	var result Result
	isTodo, err := c.ClassifyTodo(ctx, input)
	if err != nil {
		return result, err
	}
	if !isTodo {
		return result, nil
	}
	title, detail, err := c.SummarizeTodo(ctx, input)
	if err != nil {
		return result, err
	}
	result.IsTodo = true
	result.Title = title
	result.Detail = detail
	result.DeadlineAt = 0
	return result, nil
}

func (c *Client) call(ctx context.Context, systemPrompt string, userPrompt string) (string, error) {
	payload := map[string]interface{}{
		"model":        c.model,
		"instructions": systemPrompt,
		"input": []interface{}{
			map[string]interface{}{
				"role": "user",
				"content": []map[string]interface{}{
					{
						"type": "input_text",
						"text": userPrompt,
					},
				},
			},
		},
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}

	endpoint := buildOpenAIURL(c.baseURL)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.apiKey)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2048))
		bodyText := strings.TrimSpace(string(bodyBytes))
		if bodyText != "" {
			return "", fmt.Errorf("openai status %d: %s", resp.StatusCode, bodyText)
		}
		return "", fmt.Errorf("openai status %d", resp.StatusCode)
	}

	bodyBytes, _ := io.ReadAll(io.LimitReader(resp.Body, 2<<20))
	bodyText := strings.TrimSpace(string(bodyBytes))
	output := ""
	var envelope ResponseEnvelope
	if err := json.Unmarshal(bodyBytes, &envelope); err != nil {
		output = extractTextFromSSE(bodyText, resp.Header.Get("Content-Type"))
		if output == "" {
			if bodyText != "" {
				return "", fmt.Errorf("invalid response body: %s", bodyText)
			}
			return "", errors.New("invalid response body")
		}
	} else {
		if envelope.Error != nil {
			return "", fmt.Errorf("openai error: %s", envelope.Error.Message)
		}
		output = strings.TrimSpace(envelope.OutputText)
		if output == "" {
			output = strings.TrimSpace(extractOutputText(envelope.Output))
		}
	}
	if output == "" {
		return "", errors.New("empty openai output")
	}
	return output, nil
}

func buildOpenAIURL(base string) string {
	trimmed := strings.TrimRight(base, "/")
	if strings.HasSuffix(trimmed, "/v1") {
		return trimmed + "/responses"
	}
	return trimmed + "/v1/responses"
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

func parseClassifyOutput(text string) (bool, error) {
	var payload struct {
		IsTodo *bool `json:"is_todo"`
	}
	if err := parseJSONPayload(text, &payload); err != nil {
		return false, err
	}
	if payload.IsTodo == nil {
		return false, errors.New("missing is_todo")
	}
	return *payload.IsTodo, nil
}

func parseSummarizeOutput(text string) (string, string, error) {
	var payload struct {
		Title  string `json:"title"`
		Detail string `json:"detail"`
	}
	if err := parseJSONPayload(text, &payload); err != nil {
		return "", "", err
	}
	return strings.TrimSpace(payload.Title), strings.TrimSpace(payload.Detail), nil
}

func parseRelativeDateOutput(text string) (RelativeDateInfo, error) {
	var payload RelativeDateInfo
	if err := parseJSONPayload(text, &payload); err != nil {
		return RelativeDateInfo{}, err
	}
	payload.BaseDate = strings.TrimSpace(payload.BaseDate)
	payload.WeekExpr = strings.TrimSpace(payload.WeekExpr)
	payload.ResolvedDate = strings.TrimSpace(payload.ResolvedDate)
	payload.RawReasoningTag = strings.TrimSpace(payload.RawReasoningTag)
	return payload, nil
}

func parseJSONPayload(text string, out interface{}) error {
	candidate := strings.TrimSpace(text)
	candidate = strings.TrimPrefix(candidate, "```json")
	candidate = strings.TrimPrefix(candidate, "```")
	candidate = strings.TrimSuffix(candidate, "```")
	candidate = strings.TrimSpace(candidate)
	if candidate == "" {
		return errors.New("empty payload")
	}

	if err := json.Unmarshal([]byte(candidate), out); err == nil {
		return nil
	}

	if err := decodeFirstJSONObject(candidate, out); err == nil {
		return nil
	}

	start := strings.Index(candidate, "{")
	if start == -1 {
		return errors.New("invalid json payload")
	}
	if err := decodeFirstJSONObject(candidate[start:], out); err != nil {
		return errors.New("invalid json payload")
	}
	return nil
}

func decodeFirstJSONObject(text string, out interface{}) error {
	decoder := json.NewDecoder(strings.NewReader(text))
	return decoder.Decode(out)
}

