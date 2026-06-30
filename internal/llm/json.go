package llm

import (
	"encoding/json"
	"sort"
	"strings"
)

type orderedMap []mapEntry

type mapEntry struct {
	Key   string
	Value any
}

func (om orderedMap) MarshalJSON() ([]byte, error) {
	var b strings.Builder
	b.WriteByte('{')
	for i, e := range om {
		if i > 0 {
			b.WriteByte(',')
		}
		keyBytes, err := json.Marshal(e.Key)
		if err != nil {
			return nil, err
		}
		b.Write(keyBytes)
		b.WriteByte(':')
		valBytes, err := json.Marshal(e.Value)
		if err != nil {
			return nil, err
		}
		b.Write(valBytes)
	}
	b.WriteByte('}')
	return []byte(b.String()), nil
}

// DeterministicMarshal returns JSON bytes with all map[string]any keys sorted
// lexicographically, ensuring byte-identical output for the same data structure
// regardless of map iteration order.
func DeterministicMarshal(v any) ([]byte, error) {
	return json.Marshal(toOrdered(v))
}

func toOrdered(v any) any {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		om := make(orderedMap, 0, len(x))
		for _, k := range keys {
			om = append(om, mapEntry{Key: k, Value: toOrdered(x[k])})
		}
		return om
	case []any:
		for i, item := range x {
			x[i] = toOrdered(item)
		}
	}
	return v
}
