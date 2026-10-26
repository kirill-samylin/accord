package worker

import (
	"os"
	"strings"
	"testing"
)

// TestGennadiyInstalledConfigOnly runs the two config-only invocations the
// worker performs before spawn (discovery and verification) against a locally
// installed gennadiy binary named by ACCORD_TEST_GENNADIY. It never prompts
// the model and never contacts the gateway (models fetch disabled). It is
// NOT-RUN in CI and documents only the local preflight on that one machine.
func TestGennadiyInstalledConfigOnly(t *testing.T) {
	executable := os.Getenv("ACCORD_TEST_GENNADIY")
	if executable == "" {
		t.Skip("set ACCORD_TEST_GENNADIY to an installed gennadiy executable")
	}
	directory := t.TempDir()
	version, err := readNative(executable, directory, []string{"--version"})
	if err != nil || len(strings.TrimSpace(string(version))) == 0 {
		t.Fatal("version preflight", err)
	}
	help, err := readGennadiyHelp(executable, directory)
	if err != nil {
		t.Fatal("run --help preflight", err)
	}
	for _, flag := range []string{"--format", "--session", "--agent", "json"} {
		if !strings.Contains(string(help), flag) {
			t.Fatalf("installed CLI lacks %s", flag)
		}
	}
	connectors, err := discoverGennadiyMCP(executable, directory)
	if err != nil {
		t.Fatal("discovery", err)
	}
	self, args := "/synthetic/self", []string{"worker", "bridge", "--config", "/synthetic/attempt-1/bridge.json"}
	permission := gennadiyPermissions("read-only", []string{"describe", "next"})
	content, err := gennadiyConfigContent(connectors, self, args, permission)
	if err != nil {
		t.Fatal(err)
	}
	if err = verifyGennadiyConfig(executable, directory, content, self, args, permission); err != nil {
		t.Fatalf("installed fork did not resolve the injected binding/policy exactly: %v", err)
	}
	// A tampered binding must be caught by the same verification path.
	if err = verifyGennadiyConfig(executable, directory, content, "/synthetic/other", args, permission); err == nil {
		t.Fatal("verification accepted another bridge executable")
	}
	t.Logf("installed gennadiy %s: %d foreign connector(s) disabled, binding and policy verified", strings.TrimSpace(string(version)), len(connectors))
}
