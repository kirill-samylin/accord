package worker

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"os"
	"strings"

	"agent-board/internal/transport"
)

func readPreviewConfig(path string) (Config, error) {
	var config Config
	if validateLocalPath(path) != nil {
		return config, previewError()
	}
	file, err := privateRead(path)
	if err != nil {
		return config, previewError()
	}
	defer file.Close()
	data, err := io.ReadAll(io.LimitReader(file, 65537))
	if err != nil || len(data) > 65536 || transport.ValidateJSONFields(data) != nil {
		return config, previewError()
	}
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if decoder.Decode(&config) != nil || decoder.Decode(new(any)) != io.EOF {
		return Config{}, previewError()
	}
	return config, nil
}

func previewExecutableHash(ctx context.Context, path, expected string) (string, error) {
	if validateLocalPath(path) != nil {
		return "", previewError()
	}
	initial, err := os.Lstat(path)
	if err != nil || !initial.Mode().IsRegular() || initial.Size() > 1<<30 {
		return "", previewError()
	}
	file, err := os.Open(path)
	if err != nil {
		return "", previewError()
	}
	defer file.Close()
	info, err := file.Stat()
	if err != nil || !info.Mode().IsRegular() || info.Size() > 1<<30 || !os.SameFile(initial, info) {
		return "", previewError()
	}
	h := sha256.New()
	reader := io.LimitReader(file, 1<<30+1)
	buffer := make([]byte, 32<<10)
	var count int64
	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		n, err := reader.Read(buffer)
		count += int64(n)
		if count > 1<<30 {
			return "", previewError()
		}
		_, _ = h.Write(buffer[:n])
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", previewError()
		}
	}
	after, err := file.Stat()
	current, pathErr := os.Lstat(path)
	if err != nil || pathErr != nil || !os.SameFile(info, current) ||
		!os.SameFile(info, after) || info.Size() != after.Size() || !info.ModTime().Equal(after.ModTime()) {
		return "", previewError()
	}
	digest := hex.EncodeToString(h.Sum(nil))
	if expected != "" && expected != digest {
		return "", previewError()
	}
	return digest, nil
}
