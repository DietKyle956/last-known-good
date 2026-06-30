package llm

import (
	"encoding/json"
	"testing"
)

func TestDeterministicMarshalSortsMapKeys(t *testing.T) {
	input := map[string]any{
		"z": "last",
		"a": "first",
		"m": "middle",
	}
	out, err := DeterministicMarshal(input)
	if err != nil {
		t.Fatalf("DeterministicMarshal error: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed["a"] != "first" || parsed["m"] != "middle" || parsed["z"] != "last" {
		t.Fatalf("unexpected values: %+v", parsed)
	}
	// Verify key order: "a", "m", "z"
	expected := `{"a":"first","m":"middle","z":"last"}`
	if string(out) != expected {
		t.Errorf("expected %q, got %q", expected, string(out))
	}
}

func TestDeterministicMarshalNestedMaps(t *testing.T) {
	input := map[string]any{
		"properties": map[string]any{
			"path":    map[string]any{"type": "string"},
			"content": map[string]any{"type": "string"},
		},
		"required": []any{"path", "content"},
		"type":     "object",
	}
	out, err := DeterministicMarshal(input)
	if err != nil {
		t.Fatalf("DeterministicMarshal error: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(out, &parsed); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if parsed["type"] != "object" {
		t.Errorf("expected type 'object', got %v", parsed["type"])
	}
	req, ok := parsed["required"].([]any)
	if !ok || len(req) != 2 {
		t.Fatalf("expected required with 2 items, got %v", parsed["required"])
	}
}

func TestDeterministicMarshalNil(t *testing.T) {
	out, err := DeterministicMarshal(nil)
	if err != nil {
		t.Fatalf("DeterministicMarshal(nil) error: %v", err)
	}
	if string(out) != "null" {
		t.Errorf("expected 'null', got %q", string(out))
	}
}

func TestDeterministicMarshalProducesIdenticalOutput(t *testing.T) {
	input := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"path":    map[string]any{"type": "string"},
			"content": map[string]any{"type": "string"},
		},
		"required": []any{"path", "content"},
	}

	first, _ := DeterministicMarshal(input)
	second, _ := DeterministicMarshal(input)

	if string(first) != string(second) {
		t.Fatal("DeterministicMarshal produced different output for the same input")
	}
}

func TestDeterministicMarshalPreservesArrayOrder(t *testing.T) {
	input := []any{"b", "a", "c"}
	out, err := DeterministicMarshal(input)
	if err != nil {
		t.Fatalf("DeterministicMarshal error: %v", err)
	}
	if string(out) != `["b","a","c"]` {
		t.Errorf("expected [\"b\",\"a\",\"c\"], got %q", string(out))
	}
}
