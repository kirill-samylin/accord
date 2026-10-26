package worker

import (
	"errors"
	"reflect"
	"time"
)

// prepareBridgeLocked is entered and returned with mu held. The potentially
// blocking native configuration reads operate only on immutable local snapshots
// outside mu, so another run's watchdog and stop can always enter the failsafe.
func (e *Engine) prepareBridgeLocked(plan plan, run *RunState) (bridgeLaunch, error) {
	run.State, run.CommandID, run.Generation = "accepted", plan.command.ID, plan.command.Generation
	if err := e.save(); err != nil {
		return bridgeLaunch{}, preparationFailureAt("prepare_journal", err)
	}
	snapshot := *run
	config := snapshotConfig(e.config)
	epoch := e.stopEpoch
	active := e.active[run.ID]
	preparer := &Engine{config: config, discover: e.discover, verifyMCP: e.verifyMCP, discoverGennadiy: e.discoverGennadiy, verifyGennadiy: e.verifyGennadiy}
	var launch bridgeLaunch
	var prepareError error
	func() {
		e.mu.Unlock()
		defer e.mu.Lock()
		launch, prepareError = preparer.bridgeArgs(snapshot)
	}()
	if e.fatal != nil {
		return bridgeLaunch{}, preparationFailureAt("prepare_fatal", errors.Join(e.fatal, prepareError))
	}
	if e.closed {
		return bridgeLaunch{}, preparationFailureAt("prepare_closed", prepareError)
	}
	current := e.journal.Runs[snapshot.ID]
	if current != run || current.State != "accepted" || current.CommandID != snapshot.CommandID || current.Generation != snapshot.Generation || current.Provider != snapshot.Provider || current.Workspace != snapshot.Workspace || current.SessionID != snapshot.SessionID {
		return bridgeLaunch{}, preparationFailureAt("prepare_state", prepareError)
	}
	if !time.Now().Before(e.deadline) {
		return bridgeLaunch{}, preparationFailureAt("prepare_deadline", prepareError)
	}
	if e.stopEpoch != epoch || e.active[run.ID] != active || !e.allowedController(plan.command.CreatedBy) || !reflect.DeepEqual(e.config, config) {
		return bridgeLaunch{}, preparationFailureAt("prepare_authorization", prepareError)
	}
	if err := e.checkPins(run.Provider, run.Workspace); err != nil {
		return bridgeLaunch{}, preparationFailureAt("prepare_pins", errors.Join(err, prepareError))
	}
	if active != nil && active.automation != nil {
		active.automation.connectors = append([]mcpConnector{}, preparer.preparedConnectors...)
	}
	return launch, prepareError
}

func snapshotConfig(config Config) Config {
	copy := config
	copy.Automation = cloneAutomation(config.Automation)
	copy.Controllers = snapshotStrings(config.Controllers)
	if config.Executables != nil {
		copy.Executables = make(map[string]string, len(config.Executables))
	}
	for provider, path := range config.Executables {
		copy.Executables[provider] = path
	}
	if config.Models != nil {
		copy.Models = make(map[string][]string, len(config.Models))
	}
	for provider, models := range config.Models {
		copy.Models[provider] = snapshotStrings(models)
	}
	if config.NativeMCPApprovedTools != nil {
		copy.NativeMCPApprovedTools = make(map[string][]string, len(config.NativeMCPApprovedTools))
	}
	for provider, tools := range config.NativeMCPApprovedTools {
		copy.NativeMCPApprovedTools[provider] = snapshotStrings(tools)
	}
	if config.Workspaces != nil {
		copy.Workspaces = make(map[string]Workspace, len(config.Workspaces))
	}
	for alias, workspace := range config.Workspaces {
		workspace.Providers = snapshotStrings(workspace.Providers)
		copy.Workspaces[alias] = workspace
	}
	return copy
}

func snapshotStrings(values []string) []string {
	if values == nil {
		return nil
	}
	return append([]string{}, values...)
}
