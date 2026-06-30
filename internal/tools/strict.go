package tools

// SchemaSupportsStrict reports whether the given JSON Schema qualifies for
// DeepSeek's strict schema mode. Only a restricted subset of JSON Schema is
// supported in strict mode; schemas using unsupported features must use the
// non-strict fallback.
func SchemaSupportsStrict(schema map[string]any) bool {
	t, _ := schema["type"].(string)
	if t != "object" {
		return false
	}

	props, hasProps := schema["properties"].(map[string]any)
	if !hasProps {
		return false
	}

	if addl, ok := schema["additionalProperties"]; ok {
		if b, ok := addl.(bool); ok && b {
			return false
		}
	}

	for _, prop := range props {
		propSchema, ok := prop.(map[string]any)
		if !ok {
			return false
		}
		if !strictProperty(propSchema) {
			return false
		}
	}
	return true
}

func strictProperty(schema map[string]any) bool {
	if _, ok := schema["enum"]; ok {
		return false
	}
	if _, ok := schema["anyOf"]; ok {
		return false
	}
	if _, ok := schema["oneOf"]; ok {
		return false
	}
	if _, ok := schema["allOf"]; ok {
		return false
	}
	if _, ok := schema["not"]; ok {
		return false
	}
	if _, ok := schema["$ref"]; ok {
		return false
	}
	if _, ok := schema["const"]; ok {
		return false
	}
	if _, ok := schema["nullable"]; ok {
		return false
	}

	t, ok := schema["type"].(string)
	if !ok {
		return false
	}
	switch t {
	case "string", "number", "boolean":
		return true
	case "array":
		items, ok := schema["items"].(map[string]any)
		if !ok {
			return false
		}
		return strictProperty(items)
	case "object":
		props, ok := schema["properties"].(map[string]any)
		if !ok {
			return false
		}
		for _, prop := range props {
			propSchema, ok := prop.(map[string]any)
			if !ok {
				return false
			}
			if !strictProperty(propSchema) {
				return false
			}
		}
		return true
	default:
		return false
	}
}
