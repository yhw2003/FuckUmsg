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
	"chat-assist-backend/internal/timeparse"
)

type llmClient interface {
	Enabled() bool
	ClassifyTodo(ctx context.Context, input string) (bool, error)
	SummarizeTodo(ctx context.Context, input string) (string, string, error)
	ExtractRelativeDate(ctx context.Context, input string, nowContext string) (llm.RelativeDateInfo, error)
	RetryRelativeDate(ctx context.Context, input string, nowContext string, classicDate int64, previousLLM llm.RelativeDateInfo) (llm.RelativeDateInfo, error)
}

type listenerStore interface {
	Create(ctx context.Context, input storage.TodoInput) (int64, error)
	CreateLLMFailedMessage(ctx context.Context, input storage.LLMFailedMessageInput) (int64, error)
}

type Listener struct {
	cfg    config.Config
	selfID int64
	store  listenerStore
	llm    llmClient
	logger *zap.Logger
}

func NewListener(cfg config.Config, selfID int64, store listenerStore, llm llmClient, logger *zap.Logger) *Listener {
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
			_ = eventType
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(line[len("event:"):])
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
	if l.llm == nil || !l.llm.Enabled() {
		return
	}

	sourceLabel := "私聊"
	sourceID := fmt.Sprintf("%d", event.UserID)
	if event.MessageType == "group" {
		sourceLabel = "群聊"
		sourceID = fmt.Sprintf("%d", event.GroupID)
	}

	senderName := strings.TrimSpace(event.Sender.Card)
	if senderName == "" {
		senderName = strings.TrimSpace(event.Sender.Nickname)
	}

	now := time.Now()
	weekContext := weekInterpretationContext(l.cfg)
	nowContext := fmt.Sprintf(
		"当前日期: %04d-%02d-%02d\n当前星期: %s\n当前Unix时间戳: %d\n%s",
		now.Year(),
		now.Month(),
		now.Day(),
		weekdayToChinese(now.Weekday()),
		now.Unix(),
		weekContext,
	)
	promptInput := fmt.Sprintf(
		"来源: %s\n来源ID: %s\n发送者: %s(%d)\n消息时间戳: %d\n%s\n消息: %s",
		sourceLabel,
		sourceID,
		senderName,
		event.UserID,
		event.Time,
		nowContext,
		messageText,
	)

	isTodo, err := l.llm.ClassifyTodo(ctx, promptInput)
	if err != nil {
		l.logger.Error("llm classify failed", zap.Error(err))
		l.persistFailedMessage(ctx, event, sourceID, messageText, "classify", err)
		return
	}
	if !isTodo {
		return
	}
	title, detail, err := l.llm.SummarizeTodo(ctx, promptInput)
	if err != nil {
		l.logger.Error("llm summarize failed", zap.Error(err))
		l.persistFailedMessage(ctx, event, sourceID, messageText, "summarize", err)
		return
	}

	dateCtx := timeparse.Context{
		Now:         now,
		WeekMode:    l.cfg.Calendar.WeekMode,
		Week1Monday: l.cfg.Calendar.Week1Monday,
		Location:    time.Local,
	}

	relative1, err := l.llm.ExtractRelativeDate(ctx, messageText, nowContext)
	if err != nil {
		l.logger.Warn("llm relative date extract failed", zap.Error(err))
		relative1 = llm.RelativeDateInfo{}
	}
	llmDate1 := timeparse.ComputeFromRelativeInfo(relative1, dateCtx).DateAt
	classicDate := timeparse.ComputeClassicDate(messageText, dateCtx).DateAt

	l.logger.Debug("date dual-track",
		zap.String("date_stage", "extract"),
		zap.Int64("llm_date_1", llmDate1),
		zap.Int64("classic_date_1", classicDate),
	)

	llmDate2 := int64(0)
	hasRetry := false
	if llmDate1 != classicDate {
		l.logger.Debug("date dual-track",
			zap.String("date_stage", "compare"),
			zap.Int64("llm_date_1", llmDate1),
			zap.Int64("classic_date_1", classicDate),
		)
		relative2, retryErr := l.llm.RetryRelativeDate(ctx, messageText, nowContext, classicDate, relative1)
		if retryErr != nil {
			l.logger.Warn("llm relative date retry failed", zap.Error(retryErr))
		} else {
			hasRetry = true
			llmDate2 = timeparse.ComputeFromRelativeInfo(relative2, dateCtx).DateAt
			l.logger.Debug("date dual-track",
				zap.String("date_stage", "retry"),
				zap.Int64("llm_date_1", llmDate1),
				zap.Int64("classic_date_1", classicDate),
				zap.Int64("llm_date_2", llmDate2),
			)
		}
	}

	finalDeadline, deadlineLLM, deadlineClassic, deadlineConflict, deadlineNote := finalizeDateDecision(classicDate, llmDate1, hasRetry, llmDate2, time.Local)

	l.logger.Info("todo extracted",
		zap.String("title", title),
		zap.String("detail", detail),
		zap.String("source_label", sourceLabel),
		zap.String("source_id", sourceID),
		zap.String("sender_name", senderName),
		zap.Int64("sender_id", event.UserID),
		zap.Any("message_id", event.MessageID),
		zap.String("date_stage", "finalize"),
		zap.Int64("llm_date_1", llmDate1),
		zap.Int64("classic_date_1", classicDate),
		zap.Int64("llm_date_2", llmDate2),
		zap.Bool("conflict", deadlineConflict),
		zap.Int64("final_deadline_at", finalDeadline),
	)

	messageID := fmt.Sprintf("%v", event.MessageID)
	input := storage.TodoInput{
		Title:             title,
		Detail:            detail,
		SourceType:        event.MessageType,
		SourceID:          sourceID,
		SourceName:        senderName,
		SenderID:          event.UserID,
		RawMessage:        messageText,
		MessageID:         messageID,
		CreatedAt:         event.Time,
		DeadlineAt:        finalDeadline,
		DeadlineLLMAt:     deadlineLLM,
		DeadlineClassicAt: deadlineClassic,
		DeadlineConflict:  deadlineConflict,
		DeadlineNote:      deadlineNote,
	}
	if _, err := l.store.Create(ctx, input); err != nil {
		l.logger.Error("create todo failed", zap.Error(err))
	}
}

func (l *Listener) persistFailedMessage(ctx context.Context, event onebot.Event, sourceID string, messageText string, stage string, err error) {
	messageID := fmt.Sprintf("%v", event.MessageID)
	input := storage.LLMFailedMessageInput{
		UserID:     event.UserID,
		SourceType: event.MessageType,
		SourceID:   sourceID,
		MessageID:  messageID,
		RawMessage: messageText,
		FailStage:  stage,
		ErrorText:  err.Error(),
		CreatedAt:  event.Time,
	}
	if _, createErr := l.store.CreateLLMFailedMessage(ctx, input); createErr != nil {
		l.logger.Warn("persist llm failed message failed", zap.Error(createErr))
	}
}

func finalizeDateDecision(classicDate int64, llmDate1 int64, hasRetry bool, llmDate2 int64, loc *time.Location) (int64, int64, int64, bool, string) {
	if llmDate1 == classicDate {
		return llmDate1, llmDate1, classicDate, false, ""
	}
	llmSelected := llmDate1
	if hasRetry {
		llmSelected = llmDate2
		if llmDate2 == classicDate {
			return llmDate2, llmDate2, classicDate, false, ""
		}
	}
	note := fmt.Sprintf(
		"未能确认的日期：%s(llm) 或 %s(classic)",
		formatMonthDay(llmSelected, loc),
		formatMonthDay(classicDate, loc),
	)
	return 0, llmSelected, classicDate, true, note
}

func formatMonthDay(ts int64, loc *time.Location) string {
	if ts <= 0 {
		return "未知日期"
	}
	if loc == nil {
		loc = time.Local
	}
	t := time.Unix(ts, 0).In(loc)
	return fmt.Sprintf("%d月%d日", int(t.Month()), t.Day())
}

func weekInterpretationContext(cfg config.Config) string {
	if strings.EqualFold(strings.TrimSpace(cfg.Calendar.WeekMode), "natural") {
		return "周解释策略: natural（自然周）\n当消息出现“第N周周X/13周星期二”时，按当前日期所在自然周体系解释，不使用学期周基准"
	}
	return fmt.Sprintf("周解释策略: academic（学周体系）\n当消息出现“第N周周X/13周星期二”时，按学期周解释\n当前学期第1周的星期一: %s", cfg.Calendar.Week1Monday)
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
