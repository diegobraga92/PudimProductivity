package nlp

import (
	"fmt"
	"regexp"
	"strings"
	"time"
)

var dayAliases = map[string]string{
	"mon": "mon", "monday": "mon",
	"tue": "tue", "tues": "tue", "tuesday": "tue",
	"wed": "wed", "wednesday": "wed",
	"thu": "thu", "thur": "thu", "thurs": "thu", "thursday": "thu",
	"fri": "fri", "friday": "fri",
	"sat": "sat", "saturday": "sat",
	"sun": "sun", "sunday": "sun",
}

var monthNames = map[string]time.Month{
	"jan": time.January, "january": time.January,
	"feb": time.February, "february": time.February,
	"mar": time.March, "march": time.March,
	"apr": time.April, "april": time.April,
	"may": time.May,
	"jun": time.June, "june": time.June,
	"jul": time.July, "july": time.July,
	"aug": time.August, "august": time.August,
	"sep": time.September, "sept": time.September, "september": time.September,
	"oct": time.October, "october": time.October,
	"nov": time.November, "november": time.November,
	"dec": time.December, "december": time.December,
}

var weekdayKeys = []string{"mon", "tue", "wed", "thu", "fri", "sat", "sun"}

func weekDayKey(d time.Weekday) string {
	switch d {
	case time.Sunday:
		return "sun"
	case time.Monday:
		return "mon"
	case time.Tuesday:
		return "tue"
	case time.Wednesday:
		return "wed"
	case time.Thursday:
		return "thu"
	case time.Friday:
		return "fri"
	case time.Saturday:
		return "sat"
	}
	return ""
}

var (
	reEvery       = regexp.MustCompile(`(?i)\b(?:every|repeat)\s+([a-z,\s]+)`)
	reDurationH   = regexp.MustCompile(`(?i)\b(?:for\s+)?(\d+)\s*(?:hours?|hrs?|h)\b(?:\s*(?:and\s+)?(\d+)\s*(?:minutes?|mins?|m)\b)?`)
	reDurationMin = regexp.MustCompile(`(?i)\b(?:for\s+)?(\d+)\s*(?:minutes?|mins?|m)\b`)
	reTimeColon   = regexp.MustCompile(`(?i)\b(?:at\s+)?(\d{1,2}):(\d{2})\s*(am|pm)?\b`)
	reTimeClock   = regexp.MustCompile(`(?i)\b(?:at\s+)?(\d{1,2})\s*(am|pm)\b`)
	reToday       = regexp.MustCompile(`(?i)\btoday\b`)
	reTomorrow    = regexp.MustCompile(`(?i)\btomorrow\b`)
	reInDays      = regexp.MustCompile(`(?i)\bin\s+(\d+)\s+days?\b`)
	reNextWeekday = regexp.MustCompile(`(?i)\b(?:next\s+|on\s+)?(mon|monday|tue|tues|tuesday|wed|wednesday|thu|thur|thurs|thursday|fri|friday|sat|saturday|sun|sunday)\b`)
	reMonthDay    = regexp.MustCompile(`(?i)\b(?:on\s+)?(jan(?:uary)?|feb(?:ruary)?|mar(?:ch)?|apr(?:il)?|may|jun(?:e)?|jul(?:y)?|aug(?:ust)?|sep(?:t(?:ember)?)?|oct(?:ober)?|nov(?:ember)?|dec(?:ember)?)\s+(\d{1,2})(?:st|nd|rd|th)?\b`)
	reISODate     = regexp.MustCompile(`\b(\d{4})-(\d{2})-(\d{2})\b`)
)

// Parse extracts structured fields from a natural-language task input.
// now resolves relative dates and is injectable for tests.
func Parse(input string, now time.Time) (*ParsedTask, error) {
	s := strings.TrimSpace(input)
	if s == "" {
		return nil, ErrNoParse
	}

	var out ParsedTask
	out.RecurrenceDays, s = extractRecurrence(s)
	out.DurationMinutes, s = extractDuration(s)
	if d, rest, ok := extractDate(s, now); ok {
		out.DueDate = d
		s = rest
	}
	if t, rest, ok := extractTime(s); ok {
		out.StartTime = &t
		s = rest
	}

	title := strings.TrimSpace(s)
	title = strings.Join(strings.Fields(title), " ")
	title = strings.Trim(title, " .,;:-")
	out.Title = title

	if out.DurationMinutes > 0 && out.StartTime != nil {
		end := addMinutes(*out.StartTime, out.DurationMinutes)
		out.EndTime = &end
	}

	if out.Title == "" && out.DueDate == nil && out.StartTime == nil &&
		out.DurationMinutes == 0 && len(out.RecurrenceDays) == 0 {
		return nil, ErrNoParse
	}
	return &out, nil
}

var stopWords = map[string]bool{
	"for": true, "at": true, "on": true, "from": true, "until": true,
	"today": true, "tomorrow": true, "in": true,
}

func extractRecurrence(s string) ([]string, string) {
	m := reEvery.FindStringSubmatch(s)
	if m == nil {
		return nil, s
	}
	// The greedy capture may swallow trailing stop-words ("every mon for 30 min"
	// captures "mon for "). Trim them so the day list is clean.
	tokens := strings.Fields(strings.ToLower(m[1]))
	end := len(tokens)
	for end > 0 && stopWords[tokens[end-1]] {
		end--
	}
	expr := strings.Join(tokens[:end], " ")
	if expr == "" {
		return nil, s
	}

	switch expr {
	case "day", "daily":
		return weekdayKeys, strings.Replace(s, m[0], " ", 1)
	case "weekday", "weekdays":
		return []string{"mon", "tue", "wed", "thu", "fri"}, strings.Replace(s, m[0], " ", 1)
	case "week", "weekly":
		return []string{weekDayKey(time.Now().Weekday())}, strings.Replace(s, m[0], " ", 1)
	}
	var days []string
	seen := map[string]bool{}
	for _, tok := range strings.Split(expr, ",") {
		if key, ok := dayAliases[strings.TrimSpace(tok)]; ok && !seen[key] {
			seen[key] = true
			days = append(days, key)
		}
	}
	if len(days) == 0 {
		return nil, s
	}
	return days, strings.Replace(s, m[0], " ", 1)
}

func extractDuration(s string) (int, string) {
	if m := reDurationH.FindString(s); m != "" {
		sub := reDurationH.FindStringSubmatch(s)
		var h, min int
		_, _ = fmt.Sscanf(sub[1], "%d", &h)
		if sub[2] != "" {
			_, _ = fmt.Sscanf(sub[2], "%d", &min)
		}
		return h*60 + min, strings.Replace(s, m, " ", 1)
	}
	if m := reDurationMin.FindString(s); m != "" {
		sub := reDurationMin.FindStringSubmatch(s)
		var n int
		_, _ = fmt.Sscanf(sub[1], "%d", &n)
		return n, strings.Replace(s, m, " ", 1)
	}
	return 0, s
}

func extractTime(s string) (string, string, bool) {
	// Prefer a colon form: "14:30", "9:15", "2:30pm", "9:15 am".
	if m := reTimeColon.FindString(s); m != "" {
		sub := reTimeColon.FindStringSubmatch(s)
		var h, min int
		_, _ = fmt.Sscanf(sub[1], "%d", &h)
		_, _ = fmt.Sscanf(sub[2], "%d", &min)
		lower := strings.ToLower(m)
		if strings.Contains(lower, "pm") && h < 12 {
			h += 12
		}
		if strings.Contains(lower, "am") && h == 12 {
			h = 0
		}
		return fmt.Sprintf("%02d:%02d", h, min), strings.Replace(s, m, " ", 1), true
	}
	// Clock form requires an explicit am/pm (or "at" prefix) so a bare number
	// like "20" in "Read 20 pages" is never mistaken for a time.
	if m := reTimeClock.FindString(s); m != "" {
		sub := reTimeClock.FindStringSubmatch(s)
		var h int
		_, _ = fmt.Sscanf(sub[1], "%d", &h)
		lower := strings.ToLower(m)
		if strings.Contains(lower, "pm") && h < 12 {
			h += 12
		}
		if strings.Contains(lower, "am") && h == 12 {
			h = 0
		}
		return fmt.Sprintf("%02d:00", h), strings.Replace(s, m, " ", 1), true
	}
	return "", s, false
}

func extractDate(s string, now time.Time) (*string, string, bool) {
	if m := reISODate.FindString(s); m != "" {
		return ptr(m), strings.Replace(s, m, " ", 1), true
	}
	if m := reTomorrow.FindString(s); m != "" {
		date := now.AddDate(0, 0, 1).Format("2006-01-02")
		return &date, strings.Replace(s, m, " ", 1), true
	}
	if m := reToday.FindString(s); m != "" {
		date := now.Format("2006-01-02")
		return &date, strings.Replace(s, m, " ", 1), true
	}
	if m := reInDays.FindString(s); m != "" {
		var n int
		_, _ = fmt.Sscanf(m, "in %d", &n)
		date := now.AddDate(0, 0, n).Format("2006-01-02")
		return &date, strings.Replace(s, m, " ", 1), true
	}
	if m := reNextWeekday.FindString(s); m != "" {
		sub := reNextWeekday.FindStringSubmatch(s)
		key, ok := dayAliases[strings.ToLower(strings.TrimSpace(sub[1]))]
		if !ok {
			return nil, s, false
		}
		current := weekdayIndex(weekDayKey(now.Weekday()))
		target := weekdayIndex(key)
		delta := (target - current + 7) % 7
		if delta == 0 {
			delta = 7 // "next monday" on a monday means a future monday
		}
		// "next <weekday>" conventionally means next week's occurrence, not the
		// one coming up in the current week.
		if strings.Contains(strings.ToLower(m), "next") {
			delta += 7
		}
		date := now.AddDate(0, 0, delta).Format("2006-01-02")
		return &date, strings.Replace(s, m, " ", 1), true
	}
	if m := reMonthDay.FindString(s); m != "" {
		sub := reMonthDay.FindStringSubmatch(s)
		month := monthNames[strings.ToLower(sub[1])]
		var day int
		_, _ = fmt.Sscanf(sub[2], "%d", &day)
		date := time.Date(now.Year(), month, day, 0, 0, 0, 0, now.Location())
		if date.Before(now) {
			date = date.AddDate(1, 0, 0)
		}
		iso := date.Format("2006-01-02")
		return &iso, strings.Replace(s, m, " ", 1), true
	}
	return nil, s, false
}

func weekdayIndex(key string) int {
	for i, k := range weekdayKeys {
		if k == key {
			return i
		}
	}
	return -1
}

func addMinutes(hhmm string, minutes int) string {
	var h, m int
	_, _ = fmt.Sscanf(hhmm, "%d:%d", &h, &m)
	total := h*60 + m + minutes
	return fmt.Sprintf("%02d:%02d", (total/60)%24, total%60)
}

func ptr(s string) *string { return &s }
