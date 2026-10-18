package core

import (
	"encoding/json"
	"testing"
)

func TestEvidenceTypedGettersRejectWrongTypesAndPreserveIntegers(t *testing.T) {
	var e Evidence
	if err := json.Unmarshal([]byte(`{"exact":9007199254740993,"exponent":1e3,"empty":null,"array":[],"string":"ok"}`), &e); err != nil {
		t.Fatal(err)
	}
	if v, ok, err := e.Integer("exact"); err != nil || !ok || v != 9007199254740993 {
		t.Fatal(v, ok, err)
	}
	if v, _, err := e.Integer("exponent"); err != nil || v != 1000 {
		t.Fatal(v, err)
	}
	if _, err := e.String("empty"); err == nil {
		t.Fatal("null became empty string")
	}
	if _, err := e.Strings("empty"); err == nil {
		t.Fatal("null became empty array")
	}
	if _, _, err := e.Integer("empty"); err == nil {
		t.Fatal("null became zero")
	}
	if err := json.Unmarshal([]byte(`{"a":1,"a":2}`), &e); err == nil {
		t.Fatal("duplicate raw field discarded")
	}
}
