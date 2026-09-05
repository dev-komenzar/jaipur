package main

import (
	"reflect"
	"testing"
)

func TestParseSkipInput(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		total   int
		want    []int
		wantErr bool
	}{
		{"empty means skip none", "", 5, nil, false},
		{"whitespace means skip none", "   ", 5, nil, false},
		{"all skips everything", "all", 3, []int{0, 1, 2}, false},
		{"all is case insensitive", "ALL", 2, []int{0, 1}, false},
		{"single index", "2", 5, []int{1}, false},
		{"comma list", "1,3", 5, []int{0, 2}, false},
		{"comma list with spaces", " 1 , 3 ", 5, []int{0, 2}, false},
		{"range", "1-3", 5, []int{0, 1, 2}, false},
		{"reversed range", "3-1", 5, []int{0, 1, 2}, false},
		{"dedupe", "2,2,2", 5, []int{1}, false},
		{"mixed list and range", "1,3-4", 5, []int{0, 2, 3}, false},
		{"out of range", "6", 5, nil, true},
		{"zero", "0", 5, nil, true},
		{"negative in range", "-1-2", 5, nil, true},
		{"non numeric", "abc", 5, nil, true},
		{"empty part ignored", "1,,2", 5, []int{0, 1}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := parseSkipInput(tt.input, tt.total)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("parseSkipInput(%q, %d) expected error, got nil", tt.input, tt.total)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseSkipInput(%q, %d) unexpected error: %v", tt.input, tt.total, err)
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Fatalf("parseSkipInput(%q, %d) = %v, want %v", tt.input, tt.total, got, tt.want)
			}
		})
	}
}
