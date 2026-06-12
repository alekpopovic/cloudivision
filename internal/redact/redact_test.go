package redact

import (
	"strings"
	"testing"
)

func TestRedactorMasksSecretValues(t *testing.T) {
	cases := []struct {
		name  string
		key   string
		value string
	}{
		{name: "token", key: "GITHUB_TOKEN", value: "ghp_123"},
		{name: "lower token", key: "api_token", value: "tok_456"},
		{name: "password", key: "DB_PASSWORD", value: "pw"},
		{name: "secret", key: "WEBHOOK_SECRET", value: "hook"},
		{name: "authorization", key: "AUTHORIZATION", value: "Bearer abc"},
		{name: "private key", key: "SSH_PRIVATE_KEY", value: "private"},
		{name: "private key spaced", key: "private key", value: "private-spaced"},
		{name: "client secret", key: "OAUTH_CLIENT_SECRET", value: "client"},
		{name: "hyphenated client secret", key: "client-secret", value: "client-hyphen"},
		{name: "dotted token", key: "git.token", value: "dot-token"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			redactor := FromEnv(map[string]string{tc.key: tc.value, "NORMAL": "visible"})
			got := redactor.Mask("value=" + tc.value + " normal=visible")
			if strings.Contains(got, tc.value) {
				t.Fatalf("Mask() = %q, still contains secret %q", got, tc.value)
			}
			if !strings.Contains(got, "[REDACTED]") {
				t.Fatalf("Mask() = %q, want redaction marker", got)
			}
			if !strings.Contains(got, "visible") {
				t.Fatalf("Mask() = %q, want non-secret value preserved", got)
			}
		})
	}
}

func TestMaskStringMasksInlineSecretKeyValues(t *testing.T) {
	input := "token=abc&password=pw authorization:bearer normal=visible client-secret=client"
	got := MaskString(input)
	for _, leaked := range []string{"abc", "pw", "bearer", "client"} {
		if strings.Contains(got, leaked) {
			t.Fatalf("MaskString() = %q, leaked %q", got, leaked)
		}
	}
	if !strings.Contains(got, "normal=visible") {
		t.Fatalf("MaskString() = %q, want normal value preserved", got)
	}
}
