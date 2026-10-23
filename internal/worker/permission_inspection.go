package worker

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"reflect"
	"time"
)

// Both passes use the same snapshot, cwd, environment and permission overrides.
// Only MCP activation differs from native exec; it grants no filesystem access.
func inspectPermissionProcess(ctx context.Context, spec launchSpec) (permissionProof, error) {
	return inspectPermissionProcessWithStarter(ctx, spec, startSupervised)
}
func inspectPermissionProcessWithStarter(ctx context.Context, spec launchSpec, start func(launchSpec) (process, error)) (permissionProof, error) {
	if ctx.Err() != nil || spec.inspection == nil || spec.inspection.Directory != spec.directory {
		return permissionProof{}, permissionError()
	}
	child, err := start(spec)
	if err != nil {
		return permissionProof{}, errors.Join(permissionError(), err)
	}
	type capture struct {
		raw []byte
		err error
	}
	result := make(chan capture, 1)
	go func() {
		raw, err := io.ReadAll(io.LimitReader(child.Output(), 4097))
		if err == nil && len(raw) <= 4096 {
			_, err = io.Copy(io.Discard, child.Output())
		}
		if err == nil {
			err = child.Wait()
		}
		result <- capture{raw, err}
	}()
	var raw []byte
	var waitErr error
	cleaned := false
	select {
	case out := <-result:
		raw, waitErr = out.raw, out.err
	case <-ctx.Done():
		waitErr = cleanupInspection(child)
		cleaned = true
		if !errors.Is(waitErr, errOwnership) {
			waitErr = permissionError()
		}
	}
	if !cleaned && (len(raw) > 4096 || waitErr != nil) {
		cleanupErr := cleanupInspection(child)
		if errors.Is(cleanupErr, errOwnership) {
			waitErr = cleanupErr
		}
	}
	child.Output().Close()
	if errors.Is(waitErr, errOwnership) {
		return permissionProof{}, waitErr
	}
	var proof permissionProof
	if ctx.Err() != nil || waitErr != nil || len(raw) > 4096 || json.Unmarshal(raw, &proof) != nil || !validSHA256(proof.Digest) || proof.ExecutableSHA256 != spec.inspection.ExecutableSHA256 || proof.GitIdentity != spec.inspection.GitIdentity {
		return permissionProof{}, permissionError()
	}
	return proof, nil
}

// Entered/returned with Engine.mu. No config/native IO holds the global mutex.
func (e *Engine) prepareGitPermissionsLocked(p plan, active *activeRun, epoch uint64, c Config, previous *permissionProof, env []string) (*permissionProof, []string, error) {
	if !gitMetadataMode(c, p.run) {
		return nil, nil, nil
	}
	if c.CodexSandbox != "workspace-write" {
		return nil, nil, permissionError()
	}
	directory, executable := c.Workspaces[p.run.Workspace].Path, c.Executables["codex"]
	ctx, cancel := context.WithDeadline(active.automation.ctx, minDeadline(active.automation.deadline, time.Now().Add(20*time.Second)))
	defer cancel()
	inspect := e.inspectPermissions
	if inspect == nil {
		inspect = inspectPermissionProcess
	}
	connectors := append([]mcpConnector{}, active.automation.connectors...)
	e.mu.Unlock()
	var err error
	var proof permissionProof
	var i configInspection
	var args []string
	if err == nil {
		i, args, err = newConfigInspection(directory, executable, connectors)
	}
	if err == nil && ctx.Err() != nil {
		err = permissionError()
	}
	if err == nil {
		proof, err = inspect(ctx, launchSpec{executable: executable, directory: directory, args: append(append([]string{}, args...), "app-server", "--listen", "stdio://"), env: env, inspection: &i})
	}
	if err == nil {
		identity, pathErr := gitDirectoryIdentity(directory)
		if pathErr != nil || identity != i.GitIdentity || verifyExecutable(executable, i.ExecutableSHA256) != nil || !validSHA256(proof.Digest) || proof.ExecutableSHA256 != i.ExecutableSHA256 || proof.GitIdentity != i.GitIdentity {
			err = permissionError()
		}
	}
	e.mu.Lock()
	if err != nil {
		return nil, nil, err
	}
	if !e.automationPreparing(p, active, epoch, c) || e.checkPins("codex", p.run.Workspace) != nil || ctx.Err() != nil {
		return nil, nil, permissionError()
	}
	if previous != nil && !reflect.DeepEqual(*previous, proof) {
		return nil, nil, permissionError()
	}
	return &proof, gitPermissionOverrides(i), nil
}
