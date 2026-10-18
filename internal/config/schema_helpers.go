package config

import (
	"encoding/json"
	"math/big"
	"net/url"
	"strconv"
	"strings"
)

func number(v any) (*big.Rat, bool) {
	n, ok := v.(json.Number)
	if !ok {
		return nil, false
	}
	text := string(n)
	// Bound decimal exponent expansion before big.Rat allocates. The cap exceeds
	// the entire permitted evidence byte budget and is not a numeric float cast.
	if i := strings.LastIndexAny(text, "eE"); i >= 0 {
		exponent, err := strconv.ParseInt(text[i+1:], 10, 32)
		if err != nil || exponent > MaxEvidenceBytes || exponent < -MaxEvidenceBytes {
			return nil, false
		}
	}
	rat, ok := new(big.Rat).SetString(text)
	return rat, ok
}
func resolveRef(root map[string]any, ref string) (map[string]any, bool) {
	if !strings.HasPrefix(ref, "#/$defs/") {
		return nil, false
	}
	fragment, err := url.PathUnescape(strings.TrimPrefix(ref, "#/"))
	if err != nil {
		return nil, false
	}
	var value any = root
	for _, part := range strings.Split(fragment, "/") {
		for i := 0; i < len(part); i++ {
			if part[i] == '~' {
				if i+1 == len(part) || (part[i+1] != '0' && part[i+1] != '1') {
					return nil, false
				}
				i++
			}
		}
		part = strings.ReplaceAll(strings.ReplaceAll(part, "~1", "/"), "~0", "~")
		m, ok := value.(map[string]any)
		if !ok {
			return nil, false
		}
		value = m[part]
	}
	schema, ok := value.(map[string]any)
	return schema, ok
}
