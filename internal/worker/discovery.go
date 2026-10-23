package worker

import (
	"encoding/json"
	"errors"
	"io"
	"sync/atomic"
	"time"
)

type mcpConnector struct {
	Name      string
	Transport string
	Enabled   bool
	Command   string
	Args      []string
}

func discoverCodexMCP(executable, directory string) ([]mcpConnector, error) {
	data, err := readNative(executable, directory, []string{"mcp", "list", "--json"})
	if err != nil {
		return nil, err
	}
	return parseMCPFor("discovery", data)
}

func readNative(executable, directory string, args []string) ([]byte, error) {
	return readNativeFor("discovery", executable, directory, args, startProcess)
}

// starter is an internal fake-process seam; production always uses startProcess.
func readNativeFor(context, executable, directory string, args []string, starter func(launchSpec) (process, error)) ([]byte, error) {
	child, err := starter(launchSpec{executable: executable, directory: directory, args: args, env: providerEnv()})
	if err != nil {
		return nil, preparationFailureAt("prepare_"+context+"_start", err)
	}
	var timedOut atomic.Bool
	timer := time.AfterFunc(5*time.Second, func() { timedOut.Store(true); _ = child.Stop() })
	defer timer.Stop()
	data, readErr := io.ReadAll(io.LimitReader(child.Output(), 1<<20+1))
	child.Output().Close()
	if readErr != nil || len(data) > 1<<20 {
		stopErr := child.Stop()
		waitErr := child.Wait()
		step := "read"
		if len(data) > 1<<20 {
			step = "limit"
		}
		if timedOut.Load() {
			step = "timeout"
		}
		return nil, preparationFailureAt("prepare_"+context+"_"+step, errors.Join(waitErr, stopErr, readErr))
	}
	if err = child.Wait(); err != nil {
		step := "wait"
		if timedOut.Load() {
			step = "timeout"
		}
		return nil, preparationFailureAt("prepare_"+context+"_"+step, err)
	}
	return data, nil
}

func parseMCPConnectors(data []byte) ([]mcpConnector, error) {
	return parseMCPFor("discovery", data)
}

func parseMCPFor(context string, data []byte) ([]mcpConnector, error) {
	var entries []struct {
		Name      string `json:"name"`
		Enabled   *bool  `json:"enabled"`
		Transport struct {
			Type    string   `json:"type"`
			Command string   `json:"command"`
			Args    []string `json:"args"`
		} `json:"transport"`
	}
	if json.Unmarshal(data, &entries) != nil || entries == nil || len(entries) > 128 {
		return nil, preparationFailureAt("prepare_"+context+"_json", nil)
	}
	var connectors []mcpConnector
	seen := map[string]bool{}
	for _, entry := range entries {
		if entry.Name == "" || len(entry.Name) > 128 || seen[entry.Name] || entry.Enabled == nil {
			return nil, preparationFailureAt("prepare_"+context+"_metadata", nil)
		}
		if entry.Transport.Type != "stdio" && entry.Transport.Type != "streamable_http" {
			return nil, preparationFailureAt("prepare_"+context+"_transport", nil)
		}
		seen[entry.Name] = true
		connectors = append(connectors, mcpConnector{Name: entry.Name, Transport: entry.Transport.Type, Enabled: *entry.Enabled, Command: entry.Transport.Command, Args: entry.Transport.Args})
	}
	return connectors, nil
}
