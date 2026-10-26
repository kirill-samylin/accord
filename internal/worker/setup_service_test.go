package worker

import (
	"strings"
	"testing"
)

const testPlist = `<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
	<key>EnvironmentVariables</key>
	<dict><key>PATH</key><string>/usr/bin</string></dict>
	<key>Label</key>
	<string>com.accord.gui.worker</string>
	<key>ProgramArguments</key>
	<array>
		<string>/Users/owner/.config/accord-gui/managed/agent-board</string>
		<string>worker</string>
		<string>run</string>
	</array>
	<key>RunAtLoad</key>
	<true/>
</dict>
</plist>
`

func TestPlistProgramExtractsFirstArgument(t *testing.T) {
	program, err := plistProgram([]byte(testPlist))
	if err != nil || program != "/Users/owner/.config/accord-gui/managed/agent-board" {
		t.Fatal(program, err)
	}
	if _, err = plistProgram([]byte(strings.Replace(testPlist, "ProgramArguments", "Program", 1))); err == nil {
		t.Fatal("missing ProgramArguments accepted")
	}
	if _, err = plistProgram([]byte(strings.Replace(testPlist, "/Users/owner/.config/accord-gui/managed/agent-board", "agent-board", 1))); err == nil {
		t.Fatal("relative program accepted")
	}
}
