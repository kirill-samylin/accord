package client

import (
	"io"
	"strings"
	"testing"
)

func TestBodyLimitRejectsChunkedOverflowWithoutTruncationSuccess(t *testing.T) {
	for _, size := range []int{MaxResponseBytes - 1, MaxResponseBytes, MaxResponseBytes + 1} {
		body := &boundedBody{ReadCloser: io.NopCloser(strings.NewReader(strings.Repeat("x", size))), remaining: MaxResponseBytes}
		data, err := io.ReadAll(body)
		if size <= MaxResponseBytes {
			if err != nil || len(data) != size {
				t.Fatal("body within limit changed")
			}
		} else if err == nil || len(data) != MaxResponseBytes {
			t.Fatal("overflow became successful EOF")
		}
	}
}
