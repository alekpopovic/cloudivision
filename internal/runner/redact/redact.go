package redact

import "strings"

type Redactor struct {
	values []string
}

func New(values ...string) Redactor {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" {
			filtered = append(filtered, value)
		}
	}
	return Redactor{values: filtered}
}

func FromEnv(env map[string]string) Redactor {
	values := []string{}
	for key, value := range env {
		if isSecretKey(key) {
			values = append(values, value)
		}
	}
	return New(values...)
}

func (r Redactor) Mask(input string) string {
	output := input
	for _, value := range r.values {
		output = strings.ReplaceAll(output, value, "[REDACTED]")
	}
	return output
}

func isSecretKey(key string) bool {
	upper := strings.ToUpper(key)
	return strings.Contains(upper, "SECRET") ||
		strings.Contains(upper, "TOKEN") ||
		strings.Contains(upper, "PASSWORD") ||
		strings.Contains(upper, "PRIVATE_KEY")
}
