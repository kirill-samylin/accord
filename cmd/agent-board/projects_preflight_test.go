package main

import (
	"agent-board/internal/core"
	"agent-board/internal/projects"
	"agent-board/internal/store"
	"agent-board/internal/web"
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestMultiprojectServeRefusesDefaultMetadataLossBeforeConstructors(t *testing.T) {
	for _, kind := range []string{"web", "board"} {
		t.Run(kind, func(t *testing.T) {
			dir := t.TempDir()
			os.Chmod(dir, 0700)
			boardPath := filepath.Join(dir, "board.db")
			webDir := filepath.Join(dir, "web")
			root := filepath.Join(dir, "projects-data")
			board, e := store.Open(boardPath, core.Config{Project: "existing"}, nil)
			if e != nil {
				t.Fatal(e)
			}
			browser, e := web.New(board, web.Options{Origin: "http://127.0.0.1:8080", DataDir: webDir})
			if e != nil {
				t.Fatal(e)
			}
			registry, e := projects.Open(projects.Options{DataDir: root, DefaultDB: boardPath, DefaultWeb: webDir, Default: board, Offline: true, Config: core.Config{Project: "existing"}})
			if e != nil {
				t.Fatal(e)
			}
			if e = registry.Ready(context.Background()); e != nil {
				t.Fatal(e)
			}
			registry.Close()
			browser.Close()
			board.Close()
			blob := filepath.Join(webDir, "blobs", strings.Repeat("a", 48))
			if e = os.WriteFile(blob, []byte("retained design bytes"), 0600); e != nil {
				t.Fatal(e)
			}
			missing := boardPath
			if kind == "web" {
				missing = filepath.Join(webDir, "web.db")
			}
			if e = os.Remove(missing); e != nil {
				t.Fatal(e)
			}
			var out bytes.Buffer
			e = serve([]string{"--listen", "127.0.0.1:0", "--db", boardPath, "--web-origin", "http://127.0.0.1:8080", "--web-data", webDir, "--projects-data", root}, &out)
			if e == nil {
				t.Fatal("serve recreated lost published metadata")
			}
			if _, e = os.Lstat(missing); !os.IsNotExist(e) {
				t.Fatal("constructor recreated missing database", e)
			}
			bytes, e := os.ReadFile(blob)
			if e != nil || string(bytes) != "retained design bytes" {
				t.Fatal("default orphan cleanup deleted retained design", e)
			}
		})
	}
}
