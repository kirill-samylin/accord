package worker

import (
	"context"
	"encoding/xml"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// launchAgentLabel is the user LaunchAgent the Mac onboarding installed.
const launchAgentLabel = "com.accord.gui.worker"

// workerService is the background service that owns the private state lock.
// Setup must stop it before enroll/consent and start it again afterwards.
type workerService struct {
	label  string
	plist  string
	binary string
	domain string
	run    func(ctx context.Context, name string, args ...string) error
}

func runCommand(ctx context.Context, name string, args ...string) error {
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Stdout, cmd.Stderr = nil, nil
	if err := cmd.Run(); err != nil {
		return errors.New(name + " " + strings.Join(args, " ") + " failed")
	}
	return nil
}

// findWorkerService returns the installed LaunchAgent on macOS, or nil when
// there is none (the owner then runs `worker run` manually).
func findWorkerService(home string) (*workerService, error) {
	if runtime.GOOS != "darwin" || home == "" {
		return nil, nil
	}
	plist := filepath.Join(home, "Library", "LaunchAgents", launchAgentLabel+".plist")
	data, err := os.ReadFile(plist)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	binary, err := plistProgram(data)
	if err != nil {
		return nil, err
	}
	return &workerService{label: launchAgentLabel, plist: plist, binary: binary, domain: "gui/" + strconv.Itoa(os.Getuid()), run: runCommand}, nil
}

// plistProgram extracts ProgramArguments[0] from a launchd property list.
func plistProgram(data []byte) (string, error) {
	decoder := xml.NewDecoder(strings.NewReader(string(data)))
	decoder.Strict = false
	state := 0 // 0: looking for key, 1: key matched, 2: inside array
	var text string
	for {
		token, err := decoder.Token()
		if err != nil {
			return "", errors.New("LaunchAgent plist has no ProgramArguments")
		}
		switch element := token.(type) {
		case xml.StartElement:
			text = ""
			if element.Name.Local == "array" && state == 1 {
				state = 2
			}
		case xml.CharData:
			text += string(element)
		case xml.EndElement:
			switch {
			case element.Name.Local == "key" && strings.TrimSpace(text) == "ProgramArguments":
				state = 1
			case element.Name.Local == "string" && state == 2:
				program := strings.TrimSpace(text)
				if !filepath.IsAbs(program) {
					return "", errors.New("LaunchAgent program is not an absolute path")
				}
				return program, nil
			case element.Name.Local == "key":
				state = 0
			}
		}
	}
}

func (s *workerService) stop(ctx context.Context) error {
	return s.run(ctx, "launchctl", "bootout", s.domain+"/"+s.label)
}

func (s *workerService) start(ctx context.Context) error {
	return s.run(ctx, "launchctl", "bootstrap", s.domain, s.plist)
}

// sameExecutable reports whether the service already runs this binary.
func (s *workerService) sameExecutable(self string) bool {
	a, errA := os.Stat(s.binary)
	b, errB := os.Stat(self)
	return errA == nil && errB == nil && os.SameFile(a, b)
}
