package redact

import "testing"

func TestRedactorMasksKnownSecretValues(t *testing.T) {
	redactor := FromEnv(map[string]string{
		"GITHUB_TOKEN": "super-secret",
		"NORMAL":       "visible",
	})
	got := redactor.Mask("token=super-secret normal=visible")
	want := "token=[REDACTED] normal=visible"
	if got != want {
		t.Fatalf("Mask() = %q, want %q", got, want)
	}
}
