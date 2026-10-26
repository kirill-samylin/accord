package worker

import (
	"os"
	"testing"
)

// TestSetupDiscoverInstalled runs the real discovery of `worker setup` on
// this computer (PATH, typical locations, one bounded `--version` per CLI).
// It never writes anything and is NOT-RUN in CI: set ACCORD_TEST_SETUP_DISCOVER=1
// to log what the dialogue would offer on the current machine.
func TestSetupDiscoverInstalled(t *testing.T) {
	if os.Getenv("ACCORD_TEST_SETUP_DISCOVER") == "" {
		t.Skip("set ACCORD_TEST_SETUP_DISCOVER=1 to probe the installed CLIs")
	}
	found := defaultSetupProbe().discover()
	for _, provider := range nativeProviders {
		if len(found[provider]) == 0 {
			t.Logf("%s: not found", provider)
		}
		for _, cli := range found[provider] {
			t.Logf("%s: %s", provider, cli.describe())
		}
	}
}
