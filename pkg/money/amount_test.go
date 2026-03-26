package money

import (
	"errors"
	"testing"
)

func TestParseToMinorUnits(t *testing.T) {
	cases := []struct {
		name    string
		input   string
		want    int64
		wantErr error
	}{
		{name: "ok/integer_short", input: "123", want: 123},
		{name: "ok/integer", input: "12345", want: 12345},
		{name: "ok/decimal_two_digits", input: "123.45", want: 12345},
		{name: "ok/decimal_small_fraction", input: "0.01", want: 1},
		{name: "ok/decimal_one_digit", input: "123.4", want: 12340},
		{name: "ok/decimal_zero_digit", input: "123.", want: 12300},
		{name: "ok/trim_spaces", input: "  77.10  ", want: 7710},
		{name: "ok/negative_integer", input: "-1", want: -1},
		{name: "ok/negative_value", input: "-12.34", want: -1234},
		{name: "err/empty", input: "", wantErr: ErrEmptyAmount},
		{name: "err/invalid_text", input: "abc", wantErr: ErrInvalidAmount},
		{name: "err/too_many_decimals", input: "1.234", wantErr: ErrTooManyDecimals},
		{name: "err/invalid_decimal_part", input: "1.ab", wantErr: ErrInvalidDecimalPart},
		{name: "err/missing_integer_part", input: ".50", wantErr: ErrInvalidAmount},
		{name: "err/double_dot", input: "1.2.3", wantErr: ErrInvalidAmount},
		{name: "err/overflow", input: "9223372036854775808", wantErr: ErrInvalidAmount},
		{name: "err/overflow_decimal", input: "92233720368547758.08", wantErr: ErrAmountOverflow},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := ParseToMinorUnits(tc.input)
			if !errors.Is(err, tc.wantErr) {
				t.Fatalf("ParseToMinorUnits() error = %v, wantErr %v", err, tc.wantErr)
			}
			if got != tc.want {
				t.Fatalf("ParseToMinorUnits() = %d, want %d", got, tc.want)
			}
		})
	}
}
