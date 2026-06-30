package tools

import (
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// validateArgs
// ---------------------------------------------------------------------------

func TestValidateArgs_nilSchemaPassesAnyArgs(t *testing.T) {
	// nil schema passes valid JSON args (JSON is unmarshalled first)
	err := validateArgs(nil, `{"foo": "bar"}`)
	if err != nil {
		t.Fatalf("expected nil error for nil schema, got: %v", err)
	}

	// nil schema with a simple JSON value also passes
	err = validateArgs(nil, `"a string"`)
	if err != nil {
		t.Fatalf("expected nil error for nil schema with string value, got: %v", err)
	}

	// nil schema with a number also passes
	err = validateArgs(nil, `42`)
	if err != nil {
		t.Fatalf("expected nil error for nil schema with number value, got: %v", err)
	}
}

func TestValidateArgs_objectSchemaValidatesRequiredFields(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"required": []any{"name"},
	}
	err := validateArgs(schema, `{"name": "alice"}`)
	if err != nil {
		t.Fatalf("expected nil error for valid args, got: %v", err)
	}
}

func TestValidateArgs_objectSchemaRejectsMissingRequiredField(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"name": map[string]any{"type": "string"},
		},
		"required": []any{"name"},
	}
	err := validateArgs(schema, `{}`)
	if err == nil {
		t.Fatal("expected error for missing required field, got nil")
	}
	if err.Error() != `missing required property: "name"` {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateArgs_objectSchemaValidatesPropertyTypes(t *testing.T) {
	schema := map[string]any{
		"type": "object",
		"properties": map[string]any{
			"count": map[string]any{"type": "number"},
		},
	}
	// number property with number value passes
	err := validateArgs(schema, `{"count": 42}`)
	if err != nil {
		t.Fatalf("expected nil error for valid number, got: %v", err)
	}

	// string value for number property fails
	err = validateArgs(schema, `{"count": "not-a-number"}`)
	if err == nil {
		t.Fatal("expected error for wrong property type, got nil")
	}
}

func TestValidateArgs_invalidJSONReturnsError(t *testing.T) {
	schema := map[string]any{"type": "object"}
	err := validateArgs(schema, `{invalid}`)
	if err == nil {
		t.Fatal("expected error for invalid JSON, got nil")
	}
	// Verify it contains the expected error prefix
	if err.Error() != "invalid JSON: invalid character 'i' looking for beginning of object key string" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateArgs_nonObjectArgsWithObjectSchemaReturnsError(t *testing.T) {
	schema := map[string]any{"type": "object"}
	err := validateArgs(schema, `"just a string"`)
	if err == nil {
		t.Fatal("expected error for non-object args, got nil")
	}
	if err.Error() != "expected object, got string" {
		t.Fatalf("unexpected error message: %v", err)
	}
}

// ---------------------------------------------------------------------------
// validateProperty
// ---------------------------------------------------------------------------

func TestValidateProperty_stringPropertyWithStringValuePasses(t *testing.T) {
	schema := map[string]any{"type": "string"}
	err := validateProperty(schema, "name", "hello")
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

func TestValidateProperty_stringPropertyWithNumberValueFails(t *testing.T) {
	schema := map[string]any{"type": "string"}
	err := validateProperty(schema, "name", 42)
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != `property "name": expected string, got int` {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateProperty_numberPropertyWithFloat64Passes(t *testing.T) {
	schema := map[string]any{"type": "number"}
	err := validateProperty(schema, "val", float64(3.14))
	if err != nil {
		t.Fatalf("expected nil error for float64, got: %v", err)
	}
}

func TestValidateProperty_numberPropertyWithIntPasses(t *testing.T) {
	schema := map[string]any{"type": "number"}
	err := validateProperty(schema, "val", int(42))
	if err != nil {
		t.Fatalf("expected nil error for int, got: %v", err)
	}
}

func TestValidateProperty_numberPropertyWithInt64Passes(t *testing.T) {
	schema := map[string]any{"type": "number"}
	err := validateProperty(schema, "val", int64(99))
	if err != nil {
		t.Fatalf("expected nil error for int64, got: %v", err)
	}
}

func TestValidateProperty_booleanPropertyWithBoolPasses(t *testing.T) {
	schema := map[string]any{"type": "boolean"}
	err := validateProperty(schema, "flag", true)
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

func TestValidateProperty_booleanPropertyWithStringFails(t *testing.T) {
	schema := map[string]any{"type": "boolean"}
	err := validateProperty(schema, "flag", "true")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != `property "flag": expected boolean, got string` {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateProperty_arrayPropertyWithSlicePasses(t *testing.T) {
	schema := map[string]any{"type": "array"}
	err := validateProperty(schema, "items", []any{"a", "b"})
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

func TestValidateProperty_arrayPropertyWithStringFails(t *testing.T) {
	schema := map[string]any{"type": "array"}
	err := validateProperty(schema, "items", "not-an-array")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != `property "items": expected array, got string` {
		t.Fatalf("unexpected error message: %v", err)
	}
}

func TestValidateProperty_objectPropertyWithMapPasses(t *testing.T) {
	schema := map[string]any{"type": "object"}
	err := validateProperty(schema, "nested", map[string]any{"key": "val"})
	if err != nil {
		t.Fatalf("expected nil error, got: %v", err)
	}
}

func TestValidateProperty_missingTypeSkipsValidation(t *testing.T) {
	schema := map[string]any{} // no "type" key
	err := validateProperty(schema, "anything", 12345)
	if err != nil {
		t.Fatalf("expected nil error when type is missing, got: %v", err)
	}
}

// ---------------------------------------------------------------------------
// quote
// ---------------------------------------------------------------------------

func TestQuote_normalStringWrapsInSingleQuotes(t *testing.T) {
	result := quote("hello")
	expected := "'hello'"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestQuote_stringWithEmbeddedSingleQuoteEscapesCorrectly(t *testing.T) {
	result := quote("it's")
	// The shell convention: 'it'\''s'
	// Which is: close single-quote, escaped single-quote, reopen single-quote
	expected := "'it'\\''s'"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestQuote_emptyStringReturnsEmptyQuotes(t *testing.T) {
	result := quote("")
	expected := "''"
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestQuote_stringWithSpecialCharsIsWrapped(t *testing.T) {
	// spaces, $, backticks — the important thing is that the result is
	// wrapped in single quotes so the shell treats them literally.
	input := "hello world $HOME `backtick`"
	result := quote(input)
	// Must start and end with a single quote
	if len(result) < 2 || result[0] != '\'' || result[len(result)-1] != '\'' {
		t.Fatalf("expected result wrapped in single quotes, got %q", result)
	}
	// Must contain the original substring (with interior quotes possibly escaped)
	if len(result) <= len(input) {
		t.Fatalf("expected result to be at least as long as input, got %q", result)
	}
}

// ---------------------------------------------------------------------------
// escapeSed
// ---------------------------------------------------------------------------

func TestEscapeSed_backslashIsEscaped(t *testing.T) {
	result := escapeSed(`a\b`)
	expected := `a\\b`
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestEscapeSed_forwardSlashIsEscaped(t *testing.T) {
	result := escapeSed(`a/b`)
	expected := `a\/b`
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestEscapeSed_ampersandIsEscaped(t *testing.T) {
	result := escapeSed(`a&b`)
	expected := `a\&b`
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestEscapeSed_multipleOccurrencesAllEscape(t *testing.T) {
	result := escapeSed(`a\b/c&d/e\f&g`)
	expected := `a\\b\/c\&d\/e\\f\&g`
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestEscapeSed_emptyStringReturnsEmpty(t *testing.T) {
	result := escapeSed("")
	expected := ""
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

func TestEscapeSed_combinationString(t *testing.T) {
	// Mix of \, /, and & in one string
	result := escapeSed(`\/&`)
	expected := `\\\/\&`
	if result != expected {
		t.Fatalf("expected %q, got %q", expected, result)
	}
}

// ---------------------------------------------------------------------------
// ToolDefinitions order determinism
// ---------------------------------------------------------------------------

func TestToolDefinitionsAreSortedByName(t *testing.T) {
	reg := New(nil)
	// Register tools in reverse alphabetical order
	reg.Register(Tool{
		Name:        "z_tool",
		Description: "Last tool",
		Parameters:  map[string]any{"type": "object"},
	})
	reg.Register(Tool{
		Name:        "a_tool",
		Description: "First tool",
		Parameters:  map[string]any{"type": "object"},
	})
	reg.Register(Tool{
		Name:        "m_tool",
		Description: "Middle tool",
		Parameters:  map[string]any{"type": "object"},
	})

	defs := reg.ToolDefinitions()
	if len(defs) != 3 {
		t.Fatalf("expected 3 definitions, got %d", len(defs))
	}
	if defs[0].Name != "a_tool" {
		t.Errorf("expected first tool 'a_tool', got %q", defs[0].Name)
	}
	if defs[1].Name != "m_tool" {
		t.Errorf("expected second tool 'm_tool', got %q", defs[1].Name)
	}
	if defs[2].Name != "z_tool" {
		t.Errorf("expected third tool 'z_tool', got %q", defs[2].Name)
	}
}

func TestToolDefinitionsAreByteIdenticalAcrossCalls(t *testing.T) {
	reg := New(nil)
	reg.Register(Tool{
		Name:        "read_file",
		Description: "Read a file",
		Parameters:  map[string]any{"type": "object"},
	})
	reg.Register(Tool{
		Name:        "write_file",
		Description: "Write a file",
		Parameters:  map[string]any{"type": "object"},
	})

	first := reg.ToolDefinitions()
	second := reg.ToolDefinitions()

	buildPayload := func(defs []ToolDefinition) string {
		var b strings.Builder
		for _, d := range defs {
			b.WriteString("- **" + d.Name + "**: " + d.Description + "\n")
		}
		return b.String()
	}

	p1 := buildPayload(first)
	p2 := buildPayload(second)

	if len(p1) == 0 {
		t.Fatal("expected non-empty payload")
	}
	if p1 != p2 {
		t.Fatal("tool definitions payload differs between consecutive calls")
	}
}
