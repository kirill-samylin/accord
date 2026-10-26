// Package connect registers Accord as an MCP server in the owner's own
// native CLIs (Codex, Claude Code, Gennadiy) with one command and no dialogue.
package connect

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"agent-board/internal/worker"
)

// ServerName is the MCP connector name written into every CLI; the managed
// worker reserves accord_managed, so a plain name never collides with it.
const ServerName = "accord"

// Options are the inputs of Connect. Only TokenFile, Server and Project come
// from the command line; the rest are seams with real defaults.
type Options struct {
	TokenFile string
	Server    string
	Project   string
	Out       io.Writer

	Home       string
	Executable string
	CLIs       []worker.NativeCLI
	Run        func(ctx context.Context, executable string, args ...string) error
}

// Entry reports what happened for one provider.
type Entry struct {
	Provider string
	Status   string // configured, manual, absent
	Detail   string
}

type Report struct {
	TokenPath string
	Binary    string
	Entries   []Entry
}

func runCommand(ctx context.Context, executable string, args ...string) error {
	ctx, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, executable, args...)
	cmd.Env = worker.ProviderEnv()
	cmd.Stdin = strings.NewReader("")
	if home, err := os.UserHomeDir(); err == nil {
		cmd.Dir = home
	}
	if err := cmd.Run(); err != nil {
		return errors.New(filepath.Base(executable) + " " + strings.Join(args, " ") + " failed")
	}
	return nil
}

func (o *Options) defaults() error {
	if o.Out == nil {
		o.Out = io.Discard
	}
	if o.Home == "" {
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			return errors.New("cannot determine the home directory")
		}
		o.Home = home
	}
	if o.Executable == "" {
		self, err := os.Executable()
		if err != nil {
			return err
		}
		if o.Executable, err = filepath.EvalSymlinks(self); err != nil {
			return err
		}
	}
	if o.CLIs == nil {
		o.CLIs = worker.DiscoverNativeCLIs()
	}
	if o.Run == nil {
		o.Run = runCommand
	}
	return nil
}

// Connect validates the token file, fills server/project from the local
// connection.json when flags are absent, installs private copies of the
// token and this binary, and registers the MCP server in each found CLI.
func Connect(ctx context.Context, o Options) (Report, error) {
	var report Report
	if err := o.defaults(); err != nil {
		return report, err
	}
	if !filepath.IsAbs(o.TokenFile) {
		return report, errors.New("connect requires an absolute --token-file path")
	}
	token, err := worker.ReadToken(o.TokenFile)
	if err != nil {
		return report, fmt.Errorf("token file rejected: %w", err)
	}
	hints := worker.ConnectionHints(o.TokenFile)
	if o.Server == "" {
		o.Server = hints.Origin
	}
	if o.Project == "" {
		o.Project = hints.ProjectID
	}
	o.Server = strings.TrimRight(o.Server, "/")
	if !strings.HasPrefix(o.Server, "https://") && !strings.HasPrefix(o.Server, "http://127.0.0.1") && !strings.HasPrefix(o.Server, "http://localhost") {
		return report, errors.New("server origin is unknown: put connection.json beside the token or pass --server https://ORIGIN")
	}
	if o.Project == "" || strings.ContainsAny(o.Project, " \t\r\n\"") {
		return report, errors.New("project ID is unknown: put connection.json beside the token or pass --project ID")
	}
	dir := filepath.Join(o.Home, ".config", "accord")
	if err = privateDir(dir); err != nil {
		return report, err
	}
	report.TokenPath, err = installToken(o.TokenFile, token, filepath.Join(dir, o.Project+".token"))
	if err != nil {
		return report, err
	}
	report.Binary, err = installBinary(o.Executable, filepath.Join(dir, binaryName()))
	if err != nil {
		return report, err
	}
	args := []string{report.Binary, "mcp", "--server", o.Server, "--token-file", report.TokenPath, "--project", o.Project}
	for _, provider := range []string{"codex", "claude", "gennadiy"} {
		report.Entries = append(report.Entries, o.register(ctx, provider, args))
	}
	return report, nil
}

func binaryName() string {
	if runtime.GOOS == "windows" {
		return "agent-board.exe"
	}
	return "agent-board"
}

// register writes the connector into the first installation of a provider.
func (o *Options) register(ctx context.Context, provider string, command []string) Entry {
	var cli *worker.NativeCLI
	for i := range o.CLIs {
		if o.CLIs[i].Provider == provider {
			cli = &o.CLIs[i]
			break
		}
	}
	if cli == nil {
		return Entry{Provider: provider, Status: "absent", Detail: "не установлен"}
	}
	var err error
	switch provider {
	case "codex":
		_ = o.Run(ctx, cli.Path, "mcp", "remove", ServerName)
		err = o.Run(ctx, cli.Path, append([]string{"mcp", "add", ServerName, "--"}, command...)...)
	case "claude":
		_ = o.Run(ctx, cli.Path, "mcp", "remove", "--scope", "user", ServerName)
		err = o.Run(ctx, cli.Path, append([]string{"mcp", "add", "--scope", "user", ServerName, "--"}, command...)...)
	case "gennadiy":
		path := filepath.Join(o.Home, ".config", "gennadiy", "gennadiy.json")
		if err = writeGennadiyConnector(path, command); err != nil {
			return Entry{Provider: provider, Status: "manual", Detail: err.Error() + "; добавьте в " + path + " вручную: " + gennadiySnippet(command)}
		}
		return Entry{Provider: provider, Status: "configured", Detail: path + " (" + cli.Version + ")"}
	}
	if err != nil {
		return Entry{Provider: provider, Status: "manual", Detail: err.Error()}
	}
	return Entry{Provider: provider, Status: "configured", Detail: cli.Path + " (" + cli.Version + ")"}
}

// Print renders the report for the owner; the token value never appears.
func (r Report) Print(out io.Writer) {
	fmt.Fprintln(out, "Токен сохранён:", r.TokenPath)
	fmt.Fprintln(out, "Клиент Accord:", r.Binary)
	for _, entry := range r.Entries {
		switch entry.Status {
		case "configured":
			fmt.Fprintf(out, "  %s: подключён — %s\n", entry.Provider, entry.Detail)
		case "manual":
			fmt.Fprintf(out, "  %s: требует ручной правки — %s\n", entry.Provider, entry.Detail)
		default:
			fmt.Fprintf(out, "  %s: %s\n", entry.Provider, entry.Detail)
		}
	}
	fmt.Fprintln(out, "Готово. Откройте свой CLI в папке проекта и попросите агента взять следующую задачу с доски Accord.")
}
