package worker

import (
	"bytes"
	"io"
)

// A saturated capture keeps reading the ownership proof after truncation. Closing
// stdout at the byte limit would throw away the supervisor's final nonce proof.
type fixedCapture struct {
	child    process
	max      int64
	total    int64
	data     bytes.Buffer
	retain   bool
	exceeded bool
}

func (w *fixedCapture) Write(p []byte) (int, error) {
	if w.retain && w.total < w.max {
		n := min(int64(len(p)), w.max-w.total)
		_, _ = w.data.Write(p[:n])
	}
	w.total = min(w.max+1, w.total+int64(len(p)))
	if w.total > w.max && !w.exceeded {
		w.exceeded = true
		go w.child.Stop()
	}
	return len(p), nil
}
func captureProcess(child process, max int64, retain bool) ([]byte, bool, error) {
	w := &fixedCapture{child: child, max: max, retain: retain}
	_, err := io.Copy(w, child.Output())
	_ = child.Output().Close()
	if err != nil {
		_ = child.Stop()
	}
	return w.data.Bytes(), w.exceeded, err
}
