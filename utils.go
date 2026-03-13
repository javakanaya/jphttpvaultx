package jphttpvaultx

import "fmt"

// stringField extracts a string value from a data map, giving a clear error
// when the key is missing or has the wrong type.
func stringField(data map[string]interface{}, key string) (string, error) {
	raw, ok := data[key]
	if !ok {
		return "", fmt.Errorf("field %q not found in secret", key)
	}
	s, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("field %q is not a string (got %T)", key, raw)
	}
	return s, nil
}

// unwrapSecrets optionally unwraps the "secrets" envelope.
// If data contains a "secrets" key whose value is a map[string]interface{},
// that inner map is returned. Otherwise, data itself is returned unchanged.
// This makes the "secrets" wrapper optional — both flat and nested Vault
// secret layouts are accepted.
func unwrapSecrets(data map[string]interface{}) map[string]interface{} {
	if inner, ok := data["secrets"].(map[string]interface{}); ok {
		return inner
	}
	return data
}
