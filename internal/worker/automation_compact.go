package worker

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
)

// Only duplicate presentation data is compacted, after server ACK and proven
// local release. IDs, generation, hashes, receipts and observed proofs remain.
// A later continuation always supplies a fresh full immutable server snapshot.
func compactAutomation(a *automationAttempt) bool {
	if a == nil || a.Compacted || !a.NativeExited || !a.ExitDurable || !a.ManagedAck || !a.ExitAck || !a.LockReleased || len(a.Pending) > 0 {
		return false
	}
	raw, _ := json.Marshal(a.Run)
	digest := sha256.Sum256(raw)
	a.RunSnapshotSHA256 = hex.EncodeToString(digest[:])
	if a.Run.Profile != nil {
		profile := *a.Run.Profile
		raw, _ = json.Marshal(profile)
		digest = sha256.Sum256(raw)
		a.ProfileSnapshotSHA256 = hex.EncodeToString(digest[:])
		profile.Description = ""
		profile.Instructions = ""
		a.Run.Profile = &profile
	}
	a.Run.Prompt = ""
	a.Run.Output = nil
	a.IPC = nil
	a.Compacted = true
	return true
}
