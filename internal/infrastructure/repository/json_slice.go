package repository

import "encoding/json"

// encodeStringSlice serializes a []string to a JSON string for storage in
// portable TEXT columns. Empty/nil slices serialize to "" to keep the column
// minimal — both Postgres TEXT and SQLite TEXT round-trip without issues.
func encodeStringSlice(values []string) string {
	if len(values) == 0 {
		return ""
	}
	b, err := json.Marshal(values)
	if err != nil {
		return ""
	}
	return string(b)
}

func decodeStringSlice(raw string) []string {
	if raw == "" {
		return nil
	}
	var out []string
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil
	}
	return out
}
