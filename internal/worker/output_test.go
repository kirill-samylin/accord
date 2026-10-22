package worker

import (
	"strings"
	"testing"
	"unicode/utf8"
)

func TestOutputRedactsKnownCredentialsAndChunksUTF8(t *testing.T) {
	raw := "plain synthetic-bearer and synthetic-proof\x1b[31m red\x1b[0m\x00 https://username:password@example.invalid/path?token=synthetic-query " + strings.Repeat("П", 8192)
	clean := cleanOutput(raw, "synthetic-bearer", "synthetic-proof")
	for _, secret := range []string{"synthetic-bearer", "synthetic-proof", "username", "password", "synthetic-query", "\x1b", "\x00"} {
		if strings.Contains(clean, secret) {
			t.Fatalf("output retained protected value %q", secret)
		}
	}
	chunks := outputLines(Receipt{Output: clean})
	if len(chunks) < 2 {
		t.Fatal("large output was not chunked")
	}
	for _, chunk := range chunks {
		if len(chunk) > 4096 || !utf8.ValidString(chunk) {
			t.Fatal("chunk is oversized or invalid UTF8")
		}
	}
	if !strings.Contains(clean, "[output truncated]") {
		t.Fatal("truncation was not explicit")
	}
}

func TestBareProjectCredentialIsRemovedBeforeJournal(t *testing.T) {
	e := testEngine(t)
	clean := cleanOutput("synthetic-private-token "+e.journal.HostSecret, e.outputSecrets()...)
	if strings.Contains(clean, "synthetic-private-token") || strings.Contains(clean, e.journal.HostSecret) {
		t.Fatal("local secret reached journal text")
	}
}
