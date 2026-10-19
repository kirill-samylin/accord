package web

import "io"

// MultipartReader's standard header bound is larger than our form overhead.
// Keep it on a small read budget until validated file headers have been reached;
// only the actual file stream receives the larger budget.
type designBodyBudget struct {
	io.ReadCloser
	remaining, read int64
}

func (r *designBodyBudget) Read(p []byte) (int, error) {
	if r.remaining <= 0 {
		return 0, designFail(413, "FILE_TOO_LARGE", "Слишком большая форма загрузки")
	}
	if int64(len(p)) > r.remaining {
		p = p[:r.remaining]
	}
	n, err := r.ReadCloser.Read(p)
	r.remaining -= int64(n)
	r.read += int64(n)
	return n, err
}
