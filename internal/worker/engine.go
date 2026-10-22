package worker

import (
	"errors"
	"path/filepath"
	"sync"
	"time"

	"agent-board/internal/core"
)

type plan struct {
	command core.ManagedCommand
	run     core.ManagedRun
}
type activeRun struct {
	process       process
	stop          bool
	stopCommand   string
	complete      bool
	failed        bool
	observed      bool
	uncertain     bool
	workspaceLock *workspaceLock
}

// Engine never reconstructs a process ownership handle from a persisted PID.
type Engine struct {
	mu               sync.Mutex
	wg               sync.WaitGroup
	config           Config
	disk             *disk
	journal          journal
	active           map[string]*activeRun
	plans            map[string]plan
	deadline         time.Time
	fatal            error
	start            func(launchSpec) (process, error)
	discover         func(string, string) ([]string, error)
	workspaceLockDir string
	probe            func(Config) error
}

func Open(config Config) (*Engine, error) {
	if err := config.Validate(); err != nil {
		return nil, err
	}
	d, j, err := openDisk(config.StateDir)
	if err != nil {
		return nil, err
	}
	if j.Server != "" && (j.Server != config.Server || j.Project != config.Project || j.Machine != config.Machine) {
		d.close()
		return nil, errors.New("worker state is bound to another origin, project or token machine")
	}
	j.Server, j.Project, j.Machine = config.Server, config.Project, config.Machine
	if len(j.PathPins) == 0 {
		j.PathPins, err = pinPaths(config)
		if err != nil {
			d.close()
			return nil, err
		}
	}
	if err = d.save(j); err != nil {
		d.close()
		return nil, err
	}
	return &Engine{config: config, disk: d, journal: j, active: map[string]*activeRun{}, plans: map[string]plan{}, start: startSupervised, discover: discoverCodexMCP, probe: probeProviders}, nil
}

func (e *Engine) save() error {
	if e.fatal != nil {
		return e.fatal
	}
	if err := e.disk.save(e.journal); err != nil {
		e.fatal = errors.New("cannot persist worker state; all execution is quarantined")
		return e.fatal
	}
	return nil
}

func (e *Engine) SetLease(epoch int64, deadline time.Time) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.fatal != nil {
		return e.fatal
	}
	e.deadline = deadline
	e.journal.Epoch = epoch
	return e.save()
}

func (e *Engine) Accept(command core.ManagedCommand, run core.ManagedRun) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.fatal != nil {
		return e.fatal
	}
	if command.Kind != "stop" && !e.allowedController(command.CreatedBy) {
		return errors.New("command issuer is not included in this machine's explicit local consent")
	}
	if !identifier.MatchString(command.ID) || !identifier.MatchString(run.ID) || command.RunID != run.ID || command.Generation != run.Generation || command.DispatchID == "" {
		return errors.New("invalid managed dispatch")
	}
	if old := e.journal.Receipts[command.ID]; old != nil {
		if old.DispatchID != command.DispatchID || old.RunID != run.ID || old.Generation != command.Generation {
			return errors.New("managed dispatch identity changed")
		}
		return nil
	}
	if len(e.journal.Receipts) >= 4096 || len(e.journal.Runs) >= 1024 || len(e.journal.Pending) >= 128 {
		return errors.New("private journal capacity reached; manual maintenance required")
	}
	local := e.journal.Runs[run.ID]
	if command.Kind == "start" {
		if local != nil {
			return errors.New("run already exists locally")
		}
		if _, _, err := e.config.launch(run.Provider, run.Workspace); err != nil {
			return err
		}
		for _, other := range e.journal.Runs {
			if e.workspacePath(other.Workspace) == e.workspacePath(run.Workspace) && (other.State == "launching" || other.State == "running" || other.State == "stopping" || other.State == "accepted" || other.State == "needs_recovery") {
				return errors.New("workspace has another active or unresolved local run")
			}
		}
		local = &RunState{ID: run.ID, Provider: run.Provider, Workspace: run.Workspace, Generation: run.Generation, State: "accepted", CommandID: command.ID}
		e.journal.Runs[run.ID] = local
	} else if local == nil || local.Provider != run.Provider || local.Workspace != run.Workspace {
		return errors.New("managed run is not known to this worker")
	}
	if local.State == "needs_recovery" {
		return errors.New("local process identity is ambiguous; manual recovery required")
	}
	if run.ProviderSessionID != "" && local.SessionID != run.ProviderSessionID {
		return errors.New("provider session differs from the private journal")
	}
	if command.Kind != "start" && command.Kind != "stop" && command.Kind != "resume" && command.Kind != "clarify" {
		return errors.New("unsupported managed command")
	}
	if command.Kind != "stop" && command.Kind != "start" {
		if e.active[run.ID] != nil || (local.State != "stopped" && local.State != "completed") {
			return errors.New("continuation requires proven native exit")
		}
		local.Generation = run.Generation
	}
	receipt := &Receipt{RequestID: randomID("report_"), CommandID: command.ID, RunID: run.ID, DispatchID: command.DispatchID, Generation: command.Generation, Phase: "accepted"}
	e.journal.Receipts[command.ID] = receipt
	e.enqueue(*receipt)
	e.plans[command.ID] = plan{command: command, run: run}
	if command.Kind == "stop" {
		local.State = "stopping"
	}
	if err := e.save(); err != nil {
		return err
	}
	// Stopping creates no new execution authority. Do not make a local
	// failsafe wait behind an earlier report whose response was lost.
	if command.Kind == "stop" {
		if active := e.active[run.ID]; active != nil {
			active.stop = true
			active.stopCommand = command.ID
			go e.stopOwned(run.ID, active, command.ID)
		}
	}
	return nil
}

func (e *Engine) workspacePath(alias string) string {
	path := e.config.Workspaces[alias].Path
	if resolved, err := filepath.EvalSymlinks(path); err == nil {
		return resolved
	}
	return filepath.Clean(path)
}

func (e *Engine) enqueue(receipt Receipt) {
	receipt.RequestID = randomID("report_")
	e.journal.Pending = append(e.journal.Pending, &receipt)
}

func (e *Engine) Pending() *Receipt {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.journal.Pending) == 0 {
		return nil
	}
	copy := *e.journal.Pending[0]
	return &copy
}

func (e *Engine) Ack(receipt Receipt) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	if len(e.journal.Pending) == 0 || e.journal.Pending[0].RequestID != receipt.RequestID {
		return errors.New("report receipt is not current")
	}
	e.journal.Pending = e.journal.Pending[1:]
	if receipt.Phase == "completed" || receipt.Phase == "failed" || receipt.Phase == "stopped" || receipt.Phase == "recovered_stopped" {
		if stored := e.journal.Receipts[receipt.CommandID]; stored != nil {
			stored.Output = ""
		}
	}
	if err := e.save(); err != nil {
		return err
	}
	if receipt.Phase == "accepted" {
		return e.apply(receipt.CommandID)
	}
	return nil
}

func (e *Engine) apply(id string) error {
	plan, ok := e.plans[id]
	if !ok {
		return nil
	}
	delete(e.plans, id)
	run := e.journal.Runs[plan.run.ID]
	if plan.command.Kind != "stop" && !e.allowedController(plan.command.CreatedBy) {
		run.State = "failed"
		run.Error = "local controller consent was revoked"
		e.reportFor(id, "failed", run, "local controller consent was revoked", "")
		return e.save()
	}
	switch plan.command.Kind {
	case "stop":
		if receipt := e.journal.Receipts[id]; receipt != nil && receipt.Phase == "stopped" {
			return e.save()
		}
		if active := e.active[run.ID]; active != nil {
			active.stop = true
			active.stopCommand = id
			go e.stopOwned(run.ID, active, id)
		} else {
			run.State = "stopped"
			e.reportFor(id, "stopped", run, "", "")
		}
	case "start", "resume", "clarify":
		if run.State == "stopping" {
			e.reportFor(id, "failed", run, "cancelled before native launch", "")
			return e.save()
		}
		if e.active[run.ID] != nil {
			return errors.New("native run is already active")
		}
		if plan.command.Kind != "start" && (run.SessionID == "" || plan.run.ProviderSessionID != run.SessionID) {
			e.reportFor(id, "failed", run, "resume_unavailable: exact native session is missing", "")
			return e.save()
		}
		return e.launch(plan)
	default:
		return errors.New("unsupported managed command")
	}
	return e.save()
}

func (e *Engine) allowedController(id string) bool {
	if id != "" && id == e.journal.OwnerPrincipalID {
		return true
	}
	for _, allowed := range e.config.Controllers {
		if id != "" && id == allowed {
			return true
		}
	}
	return false
}

func (e *Engine) reportFor(id, phase string, run *RunState, reason, output string) {
	receipt := e.journal.Receipts[id]
	if receipt == nil {
		return
	}
	receipt.Phase, receipt.SessionID, receipt.Error, receipt.Output = phase, run.SessionID, reason, output
	e.enqueue(*receipt)
}
