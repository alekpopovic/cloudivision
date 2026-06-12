package redact

import "strings"

const mask = "[REDACTED]"

type Redactor struct {
	values []string
}

func New(values ...string) Redactor {
	filtered := make([]string, 0, len(values))
	for _, value := range values {
		if value != "" && value != mask {
			filtered = append(filtered, value)
		}
	}
	return Redactor{values: filtered}
}

func FromEnv(env map[string]string) Redactor {
	values := []string{}
	for key, value := range env {
		if IsSecretKey(key) {
			values = append(values, value)
		}
	}
	return New(values...)
}

func MaskString(input string) string {
	return MaskKeyValues(input)
}

func (r Redactor) Mask(input string) string {
	output := MaskKeyValues(input)
	for _, value := range r.values {
		output = strings.ReplaceAll(output, value, mask)
	}
	return output
}

func MaskKeyValues(input string) string {
	fields := strings.FieldsFunc(input, func(r rune) bool {
		return r == '&' || r == ' ' || r == '\n' || r == '\t'
	})
	output := input
	for _, field := range fields {
		key, value, ok := splitKeyValue(field)
		if !ok || !IsSecretKey(key) || value == "" {
			continue
		}
		output = strings.ReplaceAll(output, value, mask)
	}
	return output
}

func IsSecretKey(key string) bool {
	normalized := strings.ToUpper(strings.NewReplacer("-", "_", " ", "_", ".", "_").Replace(key))
	return strings.Contains(normalized, "TOKEN") ||
		strings.Contains(normalized, "PASSWORD") ||
		strings.Contains(normalized, "SECRET") ||
		strings.Contains(normalized, "AUTHORIZATION") ||
		strings.Contains(normalized, "PRIVATE_KEY") ||
		strings.Contains(normalized, "CLIENT_SECRET")
}

func splitKeyValue(field string) (string, string, bool) {
	for _, separator := range []string{"=", ":"} {
		if before, after, ok := strings.Cut(field, separator); ok {
			before = strings.Trim(before, "\"'")
			after = strings.Trim(after, "\"'")
			return before, after, true
		}
	}
	return "", "", false
}
