package main

import (
	"io"
	"os"
	"path/filepath"
	"testing"
)

func TestServeRejectsInvalidAdvertisedHostBeforeOpeningDatabase(t *testing.T) {
	for _, source := range []string{"flag", "environment"} {
		t.Run(source, func(t *testing.T) {
			database := filepath.Join(t.TempDir(), "must-not-exist.db")
			args := []string{"--db", database, "--listen", "127.0.0.1:0"}
			if source == "flag" {
				t.Setenv("ACCORD_EXTERNAL_HOST", "board.example:8443")
				args = append(args, "--external-host", "*.example:8443")
			} else {
				t.Setenv("ACCORD_EXTERNAL_HOST", "board.example:8443/path")
			}
			if err := serve(args, io.Discard); err == nil {
				t.Fatal("invalid host did not reject startup")
			}
			if _, err := os.Stat(database); !os.IsNotExist(err) {
				t.Fatal("invalid startup configuration touched database")
			}
		})
	}
}
