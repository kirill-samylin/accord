package worker

import (
	"bufio"
	"encoding/json"
	"io"
)

// ReadSlice keeps any single allocation bounded even for malformed native
// output. Provider event framing still belongs to the ordinary bounded scanner.
type proofReader struct {
	source io.ReadCloser
	reader *bufio.Reader
	owner  interface{ acceptProof(supervisorProof) bool }
	buffer []byte
}

func (r *proofReader) Read(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if len(r.buffer) > 0 {
		n := copy(data, r.buffer)
		r.buffer = r.buffer[n:]
		return n, nil
	}
	fragment, err := r.reader.ReadSlice('\n')
	if len(fragment) > 0 {
		var proof supervisorProof
		if len(fragment) < 1024 && json.Unmarshal(fragment, &proof) == nil && proof.Type == "accord.supervisor.exit" && r.owner.acceptProof(proof) {
			return r.Read(data)
		}
		n := copy(data, fragment)
		r.buffer = append(r.buffer[:0], fragment[n:]...)
		return n, nil
	}
	return 0, err
}
func (r *proofReader) Close() error { return r.source.Close() }
