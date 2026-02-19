package timeparse

import (
	"regexp"
	"strconv"
	"strings"
	"time"

	"chat-assist-backend/internal/llm"
)

type Context struct {
	Now         time.Time
	WeekMode    string
	Week1Monday string
	Location    *time.Location
}

type ClassicResult struct {
	Matched bool
	DateAt  int64
}

var (
	weekExprRegexp = regexp.MustCompile(`第?(\d{1,2})周(?:周|星期)([一二三四五六日天])`)
	nextWeekRegexp = regexp.MustCompile(`下周(?:周|星期)?([一二三四五六日天])`)
	weekRegexp     = regexp.MustCompile(`(?:^|[^下第\d])(?:周|星期)([一二三四五六日天])`)
)

func ComputeClassicDate(message string, cfgCtx Context) ClassicResult {
	msg := strings.TrimSpace(message)
	if msg == "" {
		return ClassicResult{}
	}
	ctx := normalizeContext(cfgCtx)

	if strings.Contains(msg, "后天") {
		return ClassicResult{Matched: true, DateAt: dayStart(ctx.Now.AddDate(0, 0, 2), ctx.Location).Unix()}
	}
	if strings.Contains(msg, "明天") {
		return ClassicResult{Matched: true, DateAt: dayStart(ctx.Now.AddDate(0, 0, 1), ctx.Location).Unix()}
	}
	if strings.Contains(msg, "今天") {
		return ClassicResult{Matched: true, DateAt: dayStart(ctx.Now, ctx.Location).Unix()}
	}

	if m := weekExprRegexp.FindStringSubmatch(msg); len(m) == 3 {
		weekNo, _ := strconv.Atoi(m[1])
		if weekNo > 0 {
			if date, ok := resolveWeekNoDate(weekNo, m[2], ctx); ok {
				return ClassicResult{Matched: true, DateAt: date.Unix()}
			}
		}
	}
	if m := nextWeekRegexp.FindStringSubmatch(msg); len(m) == 2 {
		if wd, ok := parseChineseWeekday(m[1]); ok {
			date := dateFromWeekday(ctx.Now, wd, 1, ctx.Location)
			return ClassicResult{Matched: true, DateAt: date.Unix()}
		}
	}
	if m := weekRegexp.FindStringSubmatch(msg); len(m) == 2 {
		if wd, ok := parseChineseWeekday(m[1]); ok {
			date := dateFromWeekday(ctx.Now, wd, 0, ctx.Location)
			return ClassicResult{Matched: true, DateAt: date.Unix()}
		}
	}

	return ClassicResult{}
}

func ComputeFromRelativeInfo(info llm.RelativeDateInfo, cfgCtx Context) ClassicResult {
	ctx := normalizeContext(cfgCtx)
	if info.ResolvedDate != "" {
		if date, ok := parseDate(info.ResolvedDate, ctx.Location); ok {
			return ClassicResult{Matched: true, DateAt: date.Unix()}
		}
	}
	if strings.TrimSpace(info.WeekExpr) != "" {
		if result := ComputeClassicDate(info.WeekExpr, ctx); result.Matched {
			return result
		}
	}
	base := dayStart(ctx.Now, ctx.Location)
	if strings.TrimSpace(info.BaseDate) != "" {
		if date, ok := parseDate(info.BaseDate, ctx.Location); ok {
			base = date
		}
	}
	if info.OffsetDays != 0 || strings.TrimSpace(info.BaseDate) != "" {
		return ClassicResult{Matched: true, DateAt: dayStart(base.AddDate(0, 0, info.OffsetDays), ctx.Location).Unix()}
	}
	return ClassicResult{}
}

func normalizeContext(ctx Context) Context {
	loc := ctx.Location
	if loc == nil {
		loc = time.Local
	}
	now := ctx.Now
	if now.IsZero() {
		now = time.Now().In(loc)
	} else {
		now = now.In(loc)
	}
	ctx.Location = loc
	ctx.Now = now
	ctx.WeekMode = strings.ToLower(strings.TrimSpace(ctx.WeekMode))
	if ctx.WeekMode == "" {
		ctx.WeekMode = "academic"
	}
	return ctx
}

func parseDate(raw string, loc *time.Location) (time.Time, bool) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return time.Time{}, false
	}
	for _, layout := range []string{"2006-01-02", "2006/01/02"} {
		if t, err := time.ParseInLocation(layout, trimmed, loc); err == nil {
			return dayStart(t, loc), true
		}
	}
	return time.Time{}, false
}

func parseChineseWeekday(raw string) (time.Weekday, bool) {
	switch strings.TrimSpace(raw) {
	case "一":
		return time.Monday, true
	case "二":
		return time.Tuesday, true
	case "三":
		return time.Wednesday, true
	case "四":
		return time.Thursday, true
	case "五":
		return time.Friday, true
	case "六":
		return time.Saturday, true
	case "日", "天":
		return time.Sunday, true
	default:
		return 0, false
	}
}

func dateFromWeekday(now time.Time, weekday time.Weekday, addWeeks int, loc *time.Location) time.Time {
	weekStart := weekMonday(now, loc).AddDate(0, 0, addWeeks*7)
	offset := int(weekday - time.Monday)
	if offset < 0 {
		offset += 7
	}
	candidate := weekStart.AddDate(0, 0, offset)
	if addWeeks == 0 && candidate.Before(dayStart(now, loc)) {
		candidate = candidate.AddDate(0, 0, 7)
	}
	return dayStart(candidate, loc)
}

func resolveWeekNoDate(weekNo int, weekdayText string, ctx Context) (time.Time, bool) {
	weekday, ok := parseChineseWeekday(weekdayText)
	if !ok {
		return time.Time{}, false
	}
	if ctx.WeekMode == "natural" {
		monday, ok := isoWeekMonday(ctx.Now.Year(), weekNo, ctx.Location)
		if !ok {
			return time.Time{}, false
		}
		return monday.AddDate(0, 0, isoWeekdayOffset(weekday)), true
	}
	if strings.TrimSpace(ctx.Week1Monday) == "" {
		return time.Time{}, false
	}
	base, err := time.ParseInLocation("2006-01-02", strings.TrimSpace(ctx.Week1Monday), ctx.Location)
	if err != nil {
		return time.Time{}, false
	}
	return dayStart(base, ctx.Location).AddDate(0, 0, (weekNo-1)*7+isoWeekdayOffset(weekday)), true
}

func isoWeekMonday(year int, week int, loc *time.Location) (time.Time, bool) {
	if week <= 0 || week > 53 {
		return time.Time{}, false
	}
	jan4 := time.Date(year, time.January, 4, 0, 0, 0, 0, loc)
	firstMonday := weekMonday(jan4, loc)
	candidate := firstMonday.AddDate(0, 0, (week-1)*7)
	wYear, wNum := candidate.ISOWeek()
	if wYear != year || wNum != week {
		return time.Time{}, false
	}
	return dayStart(candidate, loc), true
}

func weekMonday(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	start := dayStart(t, loc)
	offset := int(start.Weekday() - time.Monday)
	if offset < 0 {
		offset += 7
	}
	return start.AddDate(0, 0, -offset)
}

func dayStart(t time.Time, loc *time.Location) time.Time {
	t = t.In(loc)
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, loc)
}

func isoWeekdayOffset(weekday time.Weekday) int {
	offset := int(weekday - time.Monday)
	if offset < 0 {
		offset += 7
	}
	return offset
}
