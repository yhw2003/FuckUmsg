package sse

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strings"
	"time"

	"chat-assist-backend/internal/config"
	"chat-assist-backend/internal/llm"
	"chat-assist-backend/internal/onebot"
	"chat-assist-backend/internal/storage"
)

type Listener struct {
	cfg    config.Config
	selfID int64
	store  *storage.Store
	llm    *llm.Client
}

func NewListener(cfg config.Config, selfID int64, store *storage.Store, llm *llm.Client) *Listener {
	return &Listener{cfg: cfg, selfID: selfID, store: store, llm: llm}
}

func (l *Listener) Start(ctx context.Context) {
	if strings.TrimSpace(l.cfg.OneBot.SSEURL) == "" {
		log.Println("onebot sse_url not set, skip listener")
		return
	}

	taskCh := make(chan onebot.Event, 100)
	go l.worker(ctx, taskCh)

	log.Printf("sse connecting: %s", l.cfg.OneBot.SSEURL)
	backoff := time.Second
	for {
		if err := l.consume(ctx, taskCh); err != nil {
			log.Printf("sse disconnected: %v", err)
		}
		select {
		case <-ctx.Done():
			return
		case <-time.After(backoff):
			if backoff < 30*time.Second {
				backoff *= 2
			}
		}
	}
}

func (l *Listener) consume(ctx context.Context, taskCh chan<- onebot.Event) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, l.cfg.OneBot.SSEURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "text/event-stream")
	if token := strings.TrimSpace(l.cfg.OneBot.AccessToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		resp.Body.Close()
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	log.Printf("sse connected: %s", l.cfg.OneBot.SSEURL)
	defer resp.Body.Close()

	scanner := bufio.NewScanner(resp.Body)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var dataLines []string
	var eventType string
	for scanner.Scan() {
		line := strings.TrimRight(scanner.Text(), "\r")
		if line == "" {
			if len(dataLines) > 0 {
				payload := strings.Join(dataLines, "\n")
				dataLines = nil
				l.handleEvent(ctx, payload, taskCh)
			}
			eventType = ""
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(line[len("event:"):])
			_ = eventType
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(line[len("data:"):]))
			continue
		}
		if strings.HasPrefix(line, "{") {
			l.handleEvent(ctx, line, taskCh)
			continue
		}
	}
	if len(dataLines) > 0 {
		payload := strings.Join(dataLines, "\n")
		dataLines = nil
		l.handleEvent(ctx, payload, taskCh)
	}
	if err := scanner.Err(); err != nil {
		return err
	}
	return fmt.Errorf("sse stream closed")
}

func (l *Listener) handleEvent(ctx context.Context, payload string, taskCh chan<- onebot.Event) {
	if strings.TrimSpace(payload) == "" {
		return
	}
	var event onebot.Event
	if err := json.Unmarshal([]byte(payload), &event); err != nil {
		log.Printf("sse payload unmarshal failed: %v payload=%q", err, messagePreview(payload, 200))
		return
	}
	if event.PostType != "message" {
		if event.PostType != "" && event.PostType != "meta_event" {
			log.Printf("sse event ignored: post_type=%s", event.PostType)
		}
		return
	}
	if l.selfID != 0 && event.UserID == l.selfID {
		log.Printf("sse message ignored: self_id=%d user_id=%d", l.selfID, event.UserID)
		return
	}
	if event.MessageType != "group" && event.MessageType != "private" {
		log.Printf("sse message ignored: message_type=%s", event.MessageType)
		return
	}

	select {
	case taskCh <- event:
	default:
		log.Println("todo queue full, dropping event")
	}
}

func (l *Listener) worker(ctx context.Context, taskCh <-chan onebot.Event) {
	for {
		select {
		case <-ctx.Done():
			return
		case event := <-taskCh:
			l.processEvent(ctx, event)
		}
	}
}

func (l *Listener) processEvent(ctx context.Context, event onebot.Event) {
	messageText := normalizeMessage(event)
	if strings.TrimSpace(messageText) == "" {
		return
	}
	sourceLabel := "私聊"
	sourceID := fmt.Sprintf("%d", event.UserID)
	if event.MessageType == "group" {
		sourceLabel = "群聊"
		sourceID = fmt.Sprintf("%d", event.GroupID)
	}
	log.Printf("qq message received: source=%s:%s preview=%q", sourceLabel, sourceID, messagePreview(messageText, 30))
	if l.llm == nil || !l.llm.Enabled() {
		return
	}

	senderName := strings.TrimSpace(event.Sender.Card)
	if senderName == "" {
		senderName = strings.TrimSpace(event.Sender.Nickname)
	}

	promptInput := fmt.Sprintf("来源: %s\n来源ID: %s\n发送者: %s(%d)\n时间: %d\n消息: %s", sourceLabel, sourceID, senderName, event.UserID, event.Time, messageText)

	result, err := l.llm.ExtractTodo(ctx, promptInput)
	if err != nil {
		log.Printf("llm extract failed: %v", err)
		return
	}
	if !result.IsTodo {
		return
	}
	log.Printf("todo extracted: title=%q detail=%q source=%s:%s sender=%s(%d) message_id=%v", result.Title, result.Detail, sourceLabel, sourceID, senderName, event.UserID, event.MessageID)

	messageID := fmt.Sprintf("%v", event.MessageID)
	input := storage.TodoInput{
		Title:      result.Title,
		Detail:     result.Detail,
		SourceType: event.MessageType,
		SourceID:   sourceID,
		SourceName: senderName,
		SenderID:   event.UserID,
		RawMessage: messageText,
		MessageID:  messageID,
		CreatedAt:  event.Time,
	}
	if _, err := l.store.Create(ctx, input); err != nil {
		log.Printf("create todo failed: %v", err)
	}
}

func messagePreview(text string, limit int) string {
	if limit <= 0 {
		return ""
	}
	trimmed := strings.TrimSpace(text)
	cleaned := strings.NewReplacer("\r", " ", "\n", " ", "\t", " ").Replace(trimmed)
	runes := []rune(cleaned)
	if len(runes) <= limit {
		return cleaned
	}
	return string(runes[:limit]) + "..."
}

func normalizeMessage(event onebot.Event) string {
	if strings.TrimSpace(event.RawMessage) != "" {
		return event.RawMessage
	}
	if text, ok := event.Message.(string); ok {
		return text
	}
	if segments, ok := onebot.DecodeSegments(event.Message); ok {
		var builder strings.Builder
		for _, seg := range segments {
			switch seg.Type {
			case "text":
				if value, ok := seg.Data["text"]; ok {
					builder.WriteString(fmt.Sprint(value))
				}
			case "at":
				if value, ok := seg.Data["qq"]; ok {
					builder.WriteString("@")
					builder.WriteString(fmt.Sprint(value))
				}
			case "image":
				builder.WriteString("[图片]")
			default:
				builder.WriteString("[")
				builder.WriteString(seg.Type)
				builder.WriteString("]")
			}
		}
		return builder.String()
	}
	if raw, err := json.Marshal(event.Message); err == nil {
		return string(raw)
	}
	return ""
}
