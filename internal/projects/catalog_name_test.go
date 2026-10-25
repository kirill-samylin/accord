package projects

import (
	"strings"
	"testing"
)

// Project names are shown in terminals, logs and the UI; they must stay
// printable text.
func TestProjectNameRejectsControlsAndInvalidUTF8(t *testing.T) {
	for _, name := range []string{"bad\xff", "red\x1b[31m", "nul\x00", "bell\a", "c1\u009b", "tab\tname", "line\nname", " padded", strings.Repeat("a", 121), ""} {
		if validName(name) {
			t.Fatalf("accepted %q", name)
		}
	}
	for _, name := range []string{"Accord", "Тестовый проект", "web-ui v2", strings.Repeat("a", 120)} {
		if !validName(name) {
			t.Fatalf("rejected %q", name)
		}
	}
}
