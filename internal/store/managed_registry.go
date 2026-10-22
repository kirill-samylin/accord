package store

import (
	"agent-board/internal/core"
	"context"
	"errors"
)

// GuardManagedCredentials snapshots all legacy issuers while holding the same
// authority write lock as revocation. The callback must use another board.
func (s *Store) GuardManagedCredentials(ctx context.Context, required string, principals []core.Principal, call func(map[string]bool) error) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	if _, err = tx.ExecContext(ctx, `UPDATE metadata SET sequence=sequence WHERE id=1`); err != nil {
		return err
	}
	valid := map[string]bool{}
	for _, p := range principals {
		err = authenticatePrincipal(ctx, &board{tx: tx}, p, s.clock().UTC())
		if err == nil {
			valid[p.ID] = true
			continue
		}
		var domain *core.Error
		if !errors.As(err, &domain) {
			return err
		}
	}
	if required != "" && !valid[required] {
		return core.Fail("unauthorized", "Legacy credential отозван")
	}
	if err = call(valid); err != nil {
		return err
	}
	return tx.Commit()
}
func managedTarget(input any) (string, string) {
	switch v := input.(type) {
	case core.ManagedHostUpdateInput:
		return v.HostID, ""
	case core.ManagedHostDisableInput:
		return v.HostID, ""
	case core.ManagedStartInput:
		return v.HostID, ""
	case core.ManagedCommandInput:
		return "", v.RunID
	case core.ManagedPollInput:
		return v.HostID, ""
	case core.ManagedReportInput:
		return v.HostID, ""
	}
	return "", ""
}
