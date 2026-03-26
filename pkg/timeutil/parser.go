package timeutil

import (
	"fmt"
	"strconv"
	"time"
)

func Parse(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("empty time value")
	}

	// 1. Пытаемся распарсить как число (Unix Timestamp)
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return ParseUnix(i), nil
	}

	// 2. Если не число, пробуем стандартные строковые лейауты
	layouts := []string{
		time.RFC3339,     // "2006-01-02T15:04:05Z07:00"
		time.DateTime,    // "2006-01-02 15:04:05"
		time.RFC3339Nano, // "2006-01-02T15:04:05.999999999Z07:00"
		"02/01/2006",
		"2006-01-02",
	}

	for _, l := range layouts {
		if t, err := time.ParseInLocation(l, value, time.UTC); err == nil {
			return t, nil
		}
	}

	return time.Time{}, fmt.Errorf("unsupported time format: %s", value)
}

func ParseUnix(i int64) time.Time {
	// Если число очень большое (больше 10^11), скорее всего это миллисекунды
	if i > 100000000000 {
		return time.UnixMilli(i).UTC()
	}
	// Иначе считаем, что это секунды
	return time.Unix(i, 0).UTC()
}
