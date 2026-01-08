package entity

import (
	"database/sql/driver"
	"encoding/json"
	"errors"
)

// JSON is a type for storing JSON data in PostgreSQL JSONB columns.
type JSON map[string]any

// Value implements driver.Valuer for JSON.
func (j JSON) Value() (driver.Value, error) {
	if j == nil {
		return nil, nil
	}
	return json.Marshal(j)
}

// Scan implements sql.Scanner for JSON.
func (j *JSON) Scan(value interface{}) error {
	if value == nil {
		*j = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return errors.New("type assertion to []byte failed")
	}

	return json.Unmarshal(bytes, j)
}
