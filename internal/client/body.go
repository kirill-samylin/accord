package client

import (
	"errors"
	"io"
)

// Return an error on overflow instead of turning truncated JSON into EOF.
type boundedBody struct {
	io.ReadCloser
	remaining int
}

func (b *boundedBody) Read(data []byte) (int, error) {
	if len(data) == 0 {
		return 0, nil
	}
	if b.remaining == 0 {
		var probe [1]byte
		n, err := b.ReadCloser.Read(probe[:])
		if n > 0 {
			return 0, errors.New("response exceeds limit")
		}
		return 0, err
	}
	if len(data) > b.remaining {
		data = data[:b.remaining]
	}
	n, err := b.ReadCloser.Read(data)
	b.remaining -= n
	return n, err
}
