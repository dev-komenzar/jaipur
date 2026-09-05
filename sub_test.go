package main

import "testing"

func TestGetFileNameWithoutExt(t *testing.T) {
	tests := []struct{ in, want string }{
		{"foo.zip", "foo"},
		{"foo.bar.zip", "foo.bar"},
		{"noext", "noext"},
		{"(画集) [anthology] LOVE ADDICTION.zip", "(画集) [anthology] LOVE ADDICTION"},
	}
	for _, tt := range tests {
		if got := getFileNameWithoutExt(tt.in); got != tt.want {
			t.Errorf("getFileNameWithoutExt(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestRemoveWords(t *testing.T) {
	got := removeWords("hello world foo", []string{"world", "foo"})
	if got != "hello  " {
		t.Errorf("removeWords = %q, want %q", got, "hello  ")
	}
}

func TestRename(t *testing.T) {
	tests := []struct {
		name    string
		removes []string
		want    string
	}{
		{"foo.zip", nil, "foo.zip"},
		{"foo.zip", []string{"foo"}, ".zip"},
		{"  spaced  .zip", nil, "spaced.zip"},
	}
	for _, tt := range tests {
		if got := rename(tt.name, tt.removes); got != tt.want {
			t.Errorf("rename(%q, %v) = %q, want %q", tt.name, tt.removes, got, tt.want)
		}
	}
}

func TestHaveSomePrefix(t *testing.T) {
	tests := []struct {
		name string
		want bool
	}{
		{"(ERROR) foo.zip", true},
		{"(BIG) foo.zip", true},
		{"(SKIP) foo.zip", true},
		{"foo.zip", false},
		{"ERROR foo.zip", false},
		{"(error) foo.zip", false},
	}
	for _, tt := range tests {
		if got := haveSomePrefix(tt.name); got != tt.want {
			t.Errorf("haveSomePrefix(%q) = %v, want %v", tt.name, got, tt.want)
		}
	}
}

func TestUnescapeShellPath(t *testing.T) {
	tests := []struct{ in, want string }{
		{`path\ with\ spaces`, "path with spaces"},
		{`no\ escape`, "no escape"},
		{"plain", "plain"},
		{``, ""},
	}
	for _, tt := range tests {
		if got := unescapeShellPath(tt.in); got != tt.want {
			t.Errorf("unescapeShellPath(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}
