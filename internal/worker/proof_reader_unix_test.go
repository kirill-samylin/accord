//go:build darwin || linux

package worker

import (
	"bufio"
	"io"
	"strings"
	"testing"
)

func TestSupervisorProofIsNonceBound(t *testing.T) {
	data := `{"type":"accord.supervisor.exit","nonce":"wrong","proven":true,"success":true}` + "\n"
	owner := &supervisedProcess{nonce: "expected-proof"}
	reader := &proofReader{source: io.NopCloser(strings.NewReader(data)), reader: bufio.NewReader(strings.NewReader(data)), owner: owner}
	if _, err := io.ReadAll(reader); err != nil {
		t.Fatal(err)
	}
	if owner.proof != nil {
		t.Fatal("untrusted native output forged supervisor exit proof")
	}
	data = `{"type":"accord.supervisor.exit","nonce":"expected-proof","proven":true,"success":false,"forced":true}` + "\n"
	reader.reader = bufio.NewReader(strings.NewReader(data))
	if _, err := io.ReadAll(reader); err != nil {
		t.Fatal(err)
	}
	if owner.proof == nil || !owner.proof.Proven || !owner.proof.Forced {
		t.Fatal("genuine supervisor proof was lost")
	}
}
