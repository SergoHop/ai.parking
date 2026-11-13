package postgres

import "encoding/json"

// jsonArg converts a raw JSON value into something pgx sends as jsonb.
// Empty input becomes SQL NULL.
func jsonArg(raw json.RawMessage) any {
	if len(raw) == 0 {
		return nil
	}
	return string(raw)
}
