package timeutil

import (
	"testing"
	"time"
)

func TestParse_SupportsKnownFormats(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth time.Month
		wantDay   int
		wantErr   bool
	}{
		{
			name:      "ok/rfc3339_format",
			input:     "2023-10-27T10:00:00Z",
			wantYear:  2023,
			wantMonth: 10,
			wantDay:   27,
			wantErr:   false,
		},
		{
			name:      "ok/unix_seconds_string",
			input:     "1698393600",
			wantYear:  2023,
			wantMonth: 10,
			wantDay:   27,
			wantErr:   false,
		},
		{
			name:      "ok/unix_milliseconds_string",
			input:     "1698393600000",
			wantYear:  2023,
			wantMonth: 10,
			wantDay:   27,
			wantErr:   false,
		},
		{
			name:      "ok/datetime_format",
			input:     "2023-10-27 10:00:00",
			wantYear:  2023,
			wantMonth: 10,
			wantDay:   27,
			wantErr:   false,
		},
		{
			name:      "ok/simple_date_format",
			input:     "2023-10-27",
			wantYear:  2023,
			wantMonth: 10,
			wantDay:   27,
			wantErr:   false,
		},
		{
			name:    "err/empty_input",
			input:   "",
			wantErr: true,
		},
		{
			name:    "err/invalid_format",
			input:   "not-a-date",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			got, err := Parse(tc.input)
			if (err != nil) != tc.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tc.wantErr)
				return
			}
			if !tc.wantErr {
				if got.Year() != tc.wantYear || got.Month() != tc.wantMonth || got.Day() != tc.wantDay {
					t.Errorf("Parse() = %v, want date %d-%02d-%02d", got, tc.wantYear, tc.wantMonth, tc.wantDay)
				}
			}
		})
	}
}
