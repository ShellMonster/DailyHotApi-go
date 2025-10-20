package timeutil

import (
	"encoding/json"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	hhmmPattern          = regexp.MustCompile(`^\d{1,2}:\d{2}$`)
	yesterdayPattern     = regexp.MustCompile(`^昨日\s*\d{1,2}:\d{2}$`)
	monthDayPattern      = regexp.MustCompile(`^\d{1,2}月\d{1,2}日$`)
	monthDayTimePattern  = regexp.MustCompile(`^\d{1,2}月\d{1,2}日\s+\d{1,2}:\d{2}$`)
	monthDayDashPattern  = regexp.MustCompile(`^\d{1,2}-\d{1,2}$`)
	hoursAgoPattern      = regexp.MustCompile(`^(\d+)\s*小时前$`)
	minutesAgoPattern    = regexp.MustCompile(`^(\d+)\s*分钟前$`)
	numericPattern       = regexp.MustCompile(`^-?\d+(\.\d+)?$`)
	whitespaceCondenseRe = regexp.MustCompile(`\s+`)
	defaultTimeLayouts   = []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02 15:04",
		"2006-01-02 15",
		"2006-01-02",
		"2006/01/02 15:04:05",
		"2006/01/02 15:04",
		"2006/01/02",
		"2006.01.02 15:04:05",
		"2006.01.02 15:04",
		"2006.01.02",
		"20060102",
	}
)

// ParseTime 尝试将各种格式的时间字符串/数字转换为毫秒级 Unix 时间戳
// 逻辑参考 Node 版本的 getTime 工具,保持输入兼容性
func ParseTime(val interface{}) int64 {
	switch v := val.(type) {
	case int:
		return normalizeTimestamp(int64(v))
	case int64:
		return normalizeTimestamp(v)
	case int32:
		return normalizeTimestamp(int64(v))
	case float64:
		return normalizeTimestamp(int64(v))
	case float32:
		return normalizeTimestamp(int64(v))
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return normalizeTimestamp(i)
		}
		if f, err := v.Float64(); err == nil {
			return normalizeTimestamp(int64(f))
		}
	case string:
		return parseStringTime(v)
	}
	return 0
}

func parseStringTime(input string) int64 {
	s := strings.TrimSpace(input)
	if s == "" {
		return 0
	}

	// 直接解析为数字
	if numericPattern.MatchString(s) {
		if strings.Contains(s, ".") {
			if f, err := strconv.ParseFloat(s, 64); err == nil {
				return normalizeTimestamp(int64(f))
			}
		} else if num, err := strconv.ParseInt(s, 10, 64); err == nil {
			return normalizeTimestamp(num)
		}
	}

	now := time.Now()
	loc := now.Location()

	// HH:mm -> 当天
	if hhmmPattern.MatchString(s) {
		parts := strings.Split(s, ":")
		hour, _ := strconv.Atoi(parts[0])
		minute, _ := strconv.Atoi(parts[1])
		return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc).UnixMilli()
	}

	// 昨日 HH:mm
	if yesterdayPattern.MatchString(s) {
		clean := whitespaceCondenseRe.ReplaceAllString(s, " ")
		timePart := strings.TrimSpace(strings.TrimPrefix(clean, "昨日"))
		parts := strings.Split(timePart, ":")
		hour, _ := strconv.Atoi(parts[0])
		minute, _ := strconv.Atoi(parts[1])
		ts := now.AddDate(0, 0, -1)
		return time.Date(ts.Year(), ts.Month(), ts.Day(), hour, minute, 0, 0, loc).UnixMilli()
	}

	// X月X日
	if monthDayPattern.MatchString(s) {
		month, day := parseMonthDay(s)
		if month > 0 && day > 0 {
			return time.Date(now.Year(), time.Month(month), day, 0, 0, 0, 0, loc).UnixMilli()
		}
	}

	// MM-DD
	if monthDayDashPattern.MatchString(s) {
		parts := strings.Split(s, "-")
		if len(parts) == 2 {
			month, _ := strconv.Atoi(parts[0])
			day, _ := strconv.Atoi(parts[1])
			if month > 0 && day > 0 {
				return time.Date(now.Year(), time.Month(month), day, 0, 0, 0, 0, loc).UnixMilli()
			}
		}
	}

	// X月X日 HH:mm
	if monthDayTimePattern.MatchString(s) {
		parts := whitespaceCondenseRe.ReplaceAllString(s, " ")
		splits := strings.Split(parts, " ")
		if len(splits) == 2 {
			month, day := parseMonthDay(splits[0])
			hourMinute := strings.Split(splits[1], ":")
			if month > 0 && day > 0 && len(hourMinute) == 2 {
				hour, _ := strconv.Atoi(hourMinute[0])
				minute, _ := strconv.Atoi(hourMinute[1])
				return time.Date(now.Year(), time.Month(month), day, hour, minute, 0, 0, loc).UnixMilli()
			}
		}
	}

	// 今天 HH:mm
	if strings.Contains(s, "今天") {
		timeStr := strings.TrimSpace(strings.ReplaceAll(s, "今天", ""))
		if hhmmPattern.MatchString(timeStr) {
			parts := strings.Split(timeStr, ":")
			hour, _ := strconv.Atoi(parts[0])
			minute, _ := strconv.Atoi(parts[1])
			return time.Date(now.Year(), now.Month(), now.Day(), hour, minute, 0, 0, loc).UnixMilli()
		}
	}

	// 昨天 HH:mm
	if strings.Contains(s, "昨天") {
		timeStr := strings.TrimSpace(strings.ReplaceAll(s, "昨天", ""))
		if hhmmPattern.MatchString(timeStr) {
			parts := strings.Split(timeStr, ":")
			hour, _ := strconv.Atoi(parts[0])
			minute, _ := strconv.Atoi(parts[1])
			ts := now.AddDate(0, 0, -1)
			return time.Date(ts.Year(), ts.Month(), ts.Day(), hour, minute, 0, 0, loc).UnixMilli()
		}
	}

	// X小时前
	if matches := hoursAgoPattern.FindStringSubmatch(s); len(matches) == 2 {
		hours, _ := strconv.Atoi(matches[1])
		return now.Add(-time.Duration(hours) * time.Hour).UnixMilli()
	}

	// X分钟前
	if matches := minutesAgoPattern.FindStringSubmatch(s); len(matches) == 2 {
		minutes, _ := strconv.Atoi(matches[1])
		return now.Add(-time.Duration(minutes) * time.Minute).UnixMilli()
	}

	// 中文日期格式: YYYY年MM月DD日 HH:mm:ss
	if strings.Contains(s, "年") && strings.Contains(s, "月") && strings.Contains(s, "日") {
		clean := strings.NewReplacer("年", "-", "月", "-", "日", " ").Replace(s)
		clean = whitespaceCondenseRe.ReplaceAllString(strings.TrimSpace(clean), " ")
		if ts := parseWithLayouts(clean, loc); ts != 0 {
			return ts
		}
	}

	// 通用日期解析
	if ts := parseWithLayouts(s, loc); ts != 0 {
		return ts
	}

	return 0
}

func parseMonthDay(input string) (int, int) {
	clean := strings.ReplaceAll(input, "日", "")
	clean = strings.ReplaceAll(clean, "月", "-")
	parts := strings.Split(clean, "-")
	if len(parts) != 2 {
		return 0, 0
	}
	month, _ := strconv.Atoi(parts[0])
	day, _ := strconv.Atoi(parts[1])
	return month, day
}

func parseWithLayouts(value string, loc *time.Location) int64 {
	value = whitespaceCondenseRe.ReplaceAllString(strings.TrimSpace(value), " ")
	for _, layout := range defaultTimeLayouts {
		if ts, err := time.ParseInLocation(layout, value, loc); err == nil {
			return ts.UnixMilli()
		}
	}
	// 尝试 RFC3339 without location fallback
	if ts, err := time.Parse(time.RFC3339Nano, value); err == nil {
		return ts.UnixMilli()
	}
	if ts, err := time.Parse(time.RFC3339, value); err == nil {
		return ts.UnixMilli()
	}
	return 0
}

func normalizeTimestamp(ts int64) int64 {
	const threshold = 946684800000 // 2000-01-01
	if ts > threshold {
		return ts
	}
	return ts * 1000
}
