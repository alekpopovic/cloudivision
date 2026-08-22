package build

import (
	"strings"
	"testing"
	"time"
)

func TestResolveImageTagTemplates(t *testing.T) {
	input := TagInput{
		Branch: "Feature/Payments API", CommitSHA: "ABCDEF0123456789", Revision: "main",
		BuildRunName: "payments-42", Timestamp: time.Date(2026, 8, 22, 18, 5, 4, 0, time.UTC),
	}
	for _, test := range []struct {
		name     string
		template string
		want     string
	}{
		{name: "default", want: "feature-payments-api-abcdef012345"},
		{name: "build run", template: "{{ .BuildRunName }}", want: "payments-42"},
		{name: "commit", template: "{{ .CommitSHA }}", want: "abcdef0123456789"},
		{name: "timestamp", template: "{{ .Timestamp }}-{{ .ShortSHA }}", want: "20260822t180504z-abcdef012345"},
	} {
		t.Run(test.name, func(t *testing.T) {
			got, generated, err := ResolveImageTag("", test.template, input)
			if err != nil || !generated || got != test.want {
				t.Fatalf("ResolveImageTag() = %q, %t, %v; want %q, true", got, generated, err, test.want)
			}
		})
	}
}

func TestResolveImageTagHandlesEmptyCommitDeterministically(t *testing.T) {
	got, generated, err := ResolveImageTag("", "", TagInput{Branch: "main", BuildRunName: "build-1"})
	if err != nil || !generated || got != "main-unknown" {
		t.Fatalf("ResolveImageTag() = %q, %t, %v", got, generated, err)
	}
}

func TestResolveImageTagPreservesSafeManualSemver(t *testing.T) {
	got, generated, err := ResolveImageTag("v1.2.3-rc.1", "", TagInput{})
	if err != nil || generated || got != "v1.2.3-rc.1" {
		t.Fatalf("ResolveImageTag() = %q, %t, %v", got, generated, err)
	}
}

func TestSanitizeImageTagIsRegistrySafeAndBounded(t *testing.T) {
	got, err := SanitizeImageTag(" Feature/ABC + unsafe " + strings.Repeat("x", 140))
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(got, " /+") || len(got) > 128 || got != strings.ToLower(got) {
		t.Fatalf("SanitizeImageTag() = %q", got)
	}
	if _, err := SanitizeImageTag("///"); err == nil {
		t.Fatal("SanitizeImageTag() error = nil for empty result")
	}
}

func TestResolveImageTagRejectsInvalidTemplate(t *testing.T) {
	if _, _, err := ResolveImageTag("", "{{ .UnknownField }}", TagInput{}); err == nil {
		t.Fatal("ResolveImageTag() error = nil for unknown template field")
	}
}
