package core

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// PlainText accepts only valid UTF-8 without Unicode control characters (C0,
// DEL, C1), so stored text cannot carry NUL or terminal escape sequences to
// logs, terminals or the UI. Multi-line bodies may also use \n, \t and CRLF
// (a lone \r is rejected).
func PlainText(value string, multiline bool) bool {
	if !utf8.ValidString(value) {
		return false
	}
	for i, r := range value {
		if !unicode.IsControl(r) || multiline && (r == '\n' || r == '\t' || r == '\r' && strings.HasPrefix(value[i+1:], "\n")) {
			continue
		}
		return false
	}
	return true
}
