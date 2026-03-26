package money

import "testing"

func FuzzParseToMinorUnits(f *testing.F) {
	seeds := []string{
		"123",
		"123.45",
		"0.01",
		"-1",
		"1.234",
		"abc",
		"9223372036854775808",
		"92233720368547758.08",
		"  77.10  ",
		".50",
		"1.",
		"+12.34",
	}
	for _, seed := range seeds {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, raw string) {
		if _, err := ParseToMinorUnits(raw); err != nil {
			return
		}
	})
}
