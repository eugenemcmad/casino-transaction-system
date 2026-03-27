// Package timeutil parses timestamps from Unix seconds, milliseconds, or common string layouts.
package timeutil

import (
	"fmt"
	"strconv"
	"time"
)

// Parse interprets value as Unix seconds/milliseconds (if numeric) or tries several string layouts in UTC.
func Parse(value string) (time.Time, error) {
	if value == "" {
		return time.Time{}, fmt.Errorf("empty time value")
	}

	// 1. Numeric string: treat as Unix seconds or milliseconds (see ParseUnix).
	if i, err := strconv.ParseInt(value, 10, 64); err == nil {
		return ParseUnix(i), nil
	}

	// 2. Otherwise try common string layouts (UTC).
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

// ParseUnix converts a Unix timestamp in seconds or milliseconds to UTC.
func ParseUnix(i int64) time.Time {
	// Heuristic: values above 1e11 are treated as milliseconds.
	if i > 100000000000 {
		return time.UnixMilli(i).UTC()
	}
	return time.Unix(i, 0).UTC()
}
