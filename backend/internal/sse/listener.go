package sse

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

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
	logger *zap.Logger
}

func NewListener(cfg config.Config, selfID int64, store *storage.Store, llm *llm.Client, logger *zap.Logger) *Listener {
	return &Listener{cfg: cfg, selfID: selfID, store: store, llm: llm, logger: logger}
}

func (l *Listener) Start(ctx context.Context) {
	if strings.TrimSpace(l.cfg.OneBot.SSEURL) == "" {
		l.logger.Info("onebot sse_url not set, skip listener")
		return
	}

	taskCh := make(chan onebot.Event, 100)
	go l.worker(ctx, taskCh)

	l.logger.Info("sse connecting", zap.String("url", l.cfg.OneBot.SSEURL))
	backoff := time.Second
	for {
		if err := l.consume(ctx, taskCh); err != nil {
			l.logger.Warn("sse disconnected", zap.Error(err))
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
	l.logger.Info("sse connected", zap.String("url", l.cfg.OneBot.SSEURL))
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
		l.logger.Warn("sse payload unmarshal failed", zap.Error(err), zap.String("payload_preview", messagePreview(payload, 200)))
		return
	}
	if event.PostType != "message" {
		if event.PostType != "" && event.PostType != "meta_event" {
			l.logger.Debug("sse event ignored", zap.String("post_type", event.PostType))
		}
		return
	}
	if l.selfID != 0 && event.UserID == l.selfID {
		l.logger.Debug("sse message ignored", zap.Int64("self_id", l.selfID), zap.Int64("user_id", event.UserID))
		return
	}
	if event.MessageType != "group" && event.MessageType != "private" {
		l.logger.Debug("sse message ignored", zap.String("message_type", event.MessageType))
		return
	}

	select {
	case taskCh <- event:
	default:
		l.logger.Warn("todo queue full, dropping event")
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
	l.logger.Debug("qq message received",
		zap.String("source_label", sourceLabel),
		zap.String("source_id", sourceID),
		zap.String("preview", messagePreview(messageText, 30)),
	)
	if l.llm == nil || !l.llm.Enabled() {
		return
	}

	senderName := strings.TrimSpace(event.Sender.Card)
	if senderName == "" {
		senderName = strings.TrimSpace(event.Sender.Nickname)
	}

	now := time.Now()
	promptInput := fmt.Sprintf(
		"来源: %s\n来源ID: %s\n发送者: %s(%d)\n消息时间戳: %d\n当前日期: %04d-%02d-%02d\n当前星期: %s\n当前Unix时间戳: %d\n消息: %s",
		sourceLabel,
		sourceID,
		senderName,
		event.UserID,
		event.Time,
		now.Year(),
		now.Month(),
		now.Day(),
		weekdayToChinese(now.Weekday()),
		now.Unix(),
		messageText,
	)

	result, err := l.llm.ExtractTodo(ctx, promptInput)
	if err != nil {
		l.logger.Error("llm extract failed", zap.Error(err))
		return
	}
	if !result.IsTodo {
		return
	}
	l.logger.Info("todo extracted",
		zap.String("title", result.Title),
		zap.String("detail", result.Detail),
		zap.String("source_label", sourceLabel),
		zap.String("source_id", sourceID),
		zap.String("sender_name", senderName),
		zap.Int64("sender_id", event.UserID),
		zap.Any("message_id", event.MessageID),
	)

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
		DeadlineAt: result.DeadlineAt,
	}
	if _, err := l.store.Create(ctx, input); err != nil {
		l.logger.Error("create todo failed", zap.Error(err))
	}
}

func weekdayToChinese(weekday time.Weekday) string {
	switch weekday {
	case time.Sunday:
		return "星期日"
	case time.Monday:
		return "星期一"
	case time.Tuesday:
		return "星期二"
	case time.Wednesday:
		return "星期三"
	case time.Thursday:
		return "星期四"
	case time.Friday:
		return "星期五"
	case time.Saturday:
		return "星期六"
	default:
		return ""
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
