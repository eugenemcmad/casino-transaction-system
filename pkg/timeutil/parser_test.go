package timeutil

import (
	"testing"
	"time"
)

func TestParse(t *testing.T) {
	tests := []struct {
		name      string
		input     string
		wantYear  int
		wantMonth time.Month
		wantDay   int
		wantErr   bool
	}{
		{
			name:      "RFC3339 format",
			input:     "2023-10-27T10:00:00Z",
			wantYear:  2023,
			wantMonth: 10,
			wantDay:   27,
			wantErr:   false,
		},
		{
			name:      "Unix Seconds (string)",
			input:     "1698393600",
			wantYear:  2023,
			wantMonth: 10,
			wantDay:   27,
			wantErr:   false,
		},
		{
			name:      "Unix Milliseconds (string)",
			input:     "1698393600000",
			wantYear:  2023,
			wantMonth: 10,
			wantDay:   27,
			wantErr:   false,
		},
		{
			name:      "DateTime format",
			input:     "2023-10-27 10:00:00",
			wantYear:  2023,
			wantMonth: 10,
			wantDay:   27,
			wantErr:   false,
		},
		{
			name:      "Simple Date format",
			input:     "2023-10-27",
			wantYear:  2023,
			wantMonth: 10,
			wantDay:   27,
			wantErr:   false,
		},
		{
			name:      "Empty input",
			input:     "",
			wantErr:   true,
		},
		{
			name:      "Invalid format",
			input:     "not-a-date",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Parse(tt.input)
			if (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				if got.Year() != tt.wantYear || got.Month() != tt.wantMonth || got.Day() != tt.wantDay {
					t.Errorf("Parse() = %v, want date %d-%02d-%02d", got, tt.wantYear, tt.wantMonth, tt.wantDay)
				}
			}
		})
	}
}
