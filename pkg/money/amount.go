// Package money parses monetary amounts from decimal strings into int64 minor units.
package money

import (
	"errors"
	"fmt"
	"math"
	"strconv"
	"strings"
)

// Parsing and range errors for amount strings.
var (
	ErrEmptyAmount        = errors.New("amount is empty")
	ErrInvalidAmount      = errors.New("amount format is invalid")
	ErrTooManyDecimals    = errors.New("amount has more than 2 decimal places")
	ErrInvalidDecimalPart = errors.New("amount decimal part is invalid")
	ErrAmountOverflow     = errors.New("amount overflows int64 range")
)

// ParseToMinorUnits parses a string amount into minor units (int64).
// Supported formats:
// - "12345"   -> 12345
// - "123.45"  -> 12345
// - "-12.34"  -> -1234
func ParseToMinorUnits(raw string) (int64, error) {
	s := strings.TrimSpace(raw)
	if s == "" {
		return 0, ErrEmptyAmount
	}

	if !strings.Contains(s, ".") {
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return 0, fmt.Errorf("%w: %v", ErrInvalidAmount, err)
		}
		return v, nil
	}

	sign := int64(1)
	if strings.HasPrefix(s, "-") {
		sign = -1
		s = strings.TrimPrefix(s, "-")
	} else if strings.HasPrefix(s, "+") {
		s = strings.TrimPrefix(s, "+")
	}

	parts := strings.Split(s, ".")
	if len(parts) != 2 || parts[0] == "" {
		return 0, ErrInvalidAmount
	}

	intPart, err := strconv.ParseInt(parts[0], 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidAmount, err)
	}

	frac := parts[1]
	if len(frac) > 2 {
		return 0, ErrTooManyDecimals
	}
	if len(frac) == 1 {
		frac += "0"
	}
	if len(frac) == 0 {
		frac = "00"
	}

	fracPart, err := strconv.ParseInt(frac, 10, 64)
	if err != nil {
		return 0, fmt.Errorf("%w: %v", ErrInvalidDecimalPart, err)
	}

	if intPart > math.MaxInt64/100 {
		return 0, ErrAmountOverflow
	}
	if intPart < math.MinInt64/100 {
		return 0, ErrAmountOverflow
	}

	base := intPart * 100
	if base > math.MaxInt64-fracPart {
		return 0, ErrAmountOverflow
	}

	result := base + fracPart
	if sign < 0 {
		return -result, nil
	}

	return result, nil
}
