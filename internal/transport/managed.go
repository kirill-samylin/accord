package transport

import (
	"agent-board/internal/core"
	"context"
)

// ManagedService is optional so existing MCP clients and small transport
// adapters do not acquire worker control by implementing ordinary task APIs.
type ManagedService interface {
	ManagedList(context.Context, core.Actor, core.ManagedListInput) (core.ManagedPage, error)
	ManagedRun(context.Context, core.Actor, core.ManagedRunInput) (core.ManagedRunPage, error)
	ManagedHostEnroll(context.Context, core.Actor, core.ManagedHostEnrollInput) (core.ManagedHost, error)
	ManagedHostUpdate(context.Context, core.Actor, core.ManagedHostUpdateInput) (core.ManagedHost, error)
	ManagedHostDisable(context.Context, core.Actor, core.ManagedHostDisableInput) (core.ManagedHost, error)
	ManagedStart(context.Context, core.Actor, core.ManagedStartInput) (core.ManagedResult, error)
	ManagedCommand(context.Context, core.Actor, core.ManagedCommandInput) (core.ManagedResult, error)
	ManagedPoll(context.Context, core.Actor, core.ManagedPollInput) (core.ManagedPollResult, error)
	ManagedReport(context.Context, core.Actor, core.ManagedReportInput) (core.ManagedResult, error)
}

func managedOperations(s Service) []operation {
	m, ok := s.(ManagedService)
	if !ok {
		return nil
	}
	return []operation{
		strictCoordination(bind("managed_list", "Read enrolled hosts and managed runs.", true, m.ManagedList)),
		strictCoordination(bind("managed_run", "Read a managed run and command delivery history.", true, m.ManagedRun)),
		strictCoordination(bind("managed_host_enroll", "Enroll a locally consenting worker.", false, m.ManagedHostEnroll)),
		strictCoordination(bind("managed_host_update", "Update locally authorized host grants.", false, m.ManagedHostUpdate)),
		strictCoordination(bind("managed_host_disable", "Disable further worker execution.", false, m.ManagedHostDisable)),
		strictCoordination(bind("managed_start", "Queue a native provider run.", false, m.ManagedStart)),
		strictCoordination(bind("managed_command", "Queue stop, resume, or clarification.", false, m.ManagedCommand)),
		strictCoordination(bind("managed_poll", "Poll an explicitly enrolled worker.", false, m.ManagedPoll)),
		strictCoordination(bind("managed_report", "Acknowledge a fenced worker command.", false, m.ManagedReport)),
	}
}
