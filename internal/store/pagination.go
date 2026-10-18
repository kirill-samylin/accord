package store

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"strconv"
	"strings"

	"agent-board/internal/core"
)

type listCursor struct {
	After  int64  `json:"a"`
	High   int64  `json:"h"`
	Filter string `json:"f"`
}

func numericKey(key string) int64 {
	_, s, _ := strings.Cut(key, "-")
	n, _ := strconv.ParseInt(s, 10, 64)
	return n
}
func pageCursor(kind string, input any, cursor string, limit int, high int64) (listCursor, int, error) {
	if limit == 0 {
		limit = 100
	}
	if limit < 1 || limit > 200 {
		return listCursor{}, 0, core.Fail("invalid_input", "limit must be 1..200")
	}
	raw, err := json.Marshal(input)
	if err != nil {
		return listCursor{}, 0, err
	}
	digest := sha256.Sum256(append([]byte(kind+"\n"), raw...))
	filter := hex.EncodeToString(digest[:])
	c := listCursor{High: high, Filter: filter}
	if cursor != "" {
		if len(cursor) > 1024 {
			return c, 0, core.Fail("invalid_input", "invalid list cursor")
		}
		raw, err = base64.RawURLEncoding.DecodeString(cursor)
		if err != nil {
			return c, 0, core.Fail("invalid_input", "invalid list cursor")
		}
		decoder := json.NewDecoder(strings.NewReader(string(raw)))
		decoder.DisallowUnknownFields()
		if err = decoder.Decode(&c); err != nil || c.Filter != filter || c.After < 0 || c.High < 0 || c.After > c.High || c.High > high {
			return c, 0, core.Fail("invalid_input", "cursor does not match this list pass")
		}
		if decoder.Decode(new(any)) != io.EOF {
			return c, 0, core.Fail("invalid_input", "invalid list cursor")
		}
	}
	return c, limit, nil
}
func finishPage(c listCursor, last int64, more bool) core.PageInfo {
	p := core.PageInfo{HighWater: c.High, HasMore: more}
	if more {
		c.After = last
		raw, _ := json.Marshal(c)
		p.NextCursor = base64.RawURLEncoding.EncodeToString(raw)
	}
	return p
}
