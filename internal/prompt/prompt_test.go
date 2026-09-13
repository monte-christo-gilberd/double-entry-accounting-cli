package prompt

import (
	"testing"
)

func TestParseIntDefault(t *testing.T) {
	cases := []struct {
		name      string
		input     string
		def       int
		want      int
		wantOK    bool
	}{
		{name: "empty uses default", input: "", def: 1, want: 1, wantOK: true},
		{name: "valid number", input: "5", def: 1, want: 5, wantOK: true},
		{name: "zero accepted", input: "0", def: 1, want: 0, wantOK: true},
		{name: "negative accepted", input: "-3", def: 1, want: -3, wantOK: true},
		{name: "garbage rejected", input: "abc", def: 1, want: 0, wantOK: false},
		{name: "float rejected", input: "2.5", def: 1, want: 0, wantOK: false},
		{name: "whitespace rejected", input: "  ", def: 1, want: 0, wantOK: false},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			got, ok := parseIntDefault(tc.input, tc.def)
			if ok != tc.wantOK || got != tc.want {
				t.Fatalf("parseIntDefault(%q, %d) = (%d, %v), want (%d, %v)",
					tc.input, tc.def, got, ok, tc.want, tc.wantOK)
			}
		})
	}
}
