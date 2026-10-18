package main

import (
	"bytes"
	"strings"
	"testing"
)

func TestHelpListsOperationalCommandsWithoutOpeningDatabase(t *testing.T) {
	for _, args := range [][]string{{"--help"}, {"-h"}, {"help"}, {"mcp", "--help"}, {"token", "revoke-all", "-h"}, {"maintenance", "reset", "--help"}} {
		var out bytes.Buffer
		if err := run(args, strings.NewReader(""), &out); err != nil {
			t.Fatal(err)
		}
		for _, expected := range []string{"agent-board mcp", "token list", "token revoke-all", "sessions list", "maintenance reset", "backup --db", "stop local edits"} {
			if !strings.Contains(out.String(), expected) {
				t.Fatalf("missing help: %s", expected)
			}
		}
	}
}
func TestMCPFlagsDoNotEmitNonProtocolOutput(t *testing.T) {
	var out bytes.Buffer
	if err := run([]string{"mcp", "--heartbeat-interval", "-1s"}, strings.NewReader(""), &out); err == nil {
		t.Fatal("accepted invalid heartbeat interval")
	}
	if out.Len() != 0 {
		t.Fatal("MCP flag failure printed to stdout")
	}
}
