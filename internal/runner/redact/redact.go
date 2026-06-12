package redact

import central "github.com/cloudivision/cloudivision/internal/redact"

type Redactor = central.Redactor

func New(values ...string) Redactor {
	return central.New(values...)
}

func FromEnv(env map[string]string) Redactor {
	return central.FromEnv(env)
}
