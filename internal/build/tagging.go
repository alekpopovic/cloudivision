package build

import (
	"bytes"
	"fmt"
	"regexp"
	"strings"
	"text/template"
	"time"
	"unicode"
)

const DefaultTagTemplate = "{{ .Branch }}-{{ .ShortSHA }}"

var commitPattern = regexp.MustCompile(`^[a-fA-F0-9]{7,64}$`)

type TagInput struct {
	Branch       string
	CommitSHA    string
	Revision     string
	BuildRunName string
	Timestamp    time.Time
}

type TagTemplateData struct {
	Branch       string
	CommitSHA    string
	ShortSHA     string
	BuildRunName string
	Timestamp    string
}

func ResolveImageTag(explicitTag, tagTemplate string, input TagInput) (string, bool, error) {
	if explicitTag != "" {
		tag, err := SanitizeImageTag(explicitTag)
		return tag, false, err
	}
	if tagTemplate == "" {
		tagTemplate = DefaultTagTemplate
	}
	data := tagTemplateData(input)
	parsed, err := template.New("image-tag").Option("missingkey=error").Parse(tagTemplate)
	if err != nil {
		return "", true, fmt.Errorf("parse image tag template: %w", err)
	}
	var rendered bytes.Buffer
	if err := parsed.Execute(&rendered, data); err != nil {
		return "", true, fmt.Errorf("render image tag template: %w", err)
	}
	tag, err := SanitizeImageTag(rendered.String())
	if err != nil {
		return "", true, fmt.Errorf("sanitize generated image tag: %w", err)
	}
	return tag, true, nil
}

func tagTemplateData(input TagInput) TagTemplateData {
	branch := strings.TrimSpace(input.Branch)
	if branch == "" {
		branch = "detached"
	}
	commit := strings.TrimSpace(input.CommitSHA)
	if commit == "" && commitPattern.MatchString(strings.TrimSpace(input.Revision)) {
		commit = strings.TrimSpace(input.Revision)
	}
	commit = strings.ToLower(commit)
	short := commit
	if len(short) > 12 {
		short = short[:12]
	}
	if commit == "" {
		commit = "unknown"
		short = "unknown"
	}
	name := strings.TrimSpace(input.BuildRunName)
	if name == "" {
		name = "build"
	}
	timestamp := input.Timestamp.UTC()
	if timestamp.IsZero() {
		timestamp = time.Unix(0, 0).UTC()
	}
	return TagTemplateData{
		Branch:       branch,
		CommitSHA:    commit,
		ShortSHA:     short,
		BuildRunName: name,
		Timestamp:    timestamp.Format("20060102T150405Z"),
	}
}

func SanitizeImageTag(value string) (string, error) {
	value = strings.TrimSpace(value)
	var result strings.Builder
	lastReplacement := false
	for _, current := range value {
		allowed := current < unicode.MaxASCII && (unicode.IsLetter(current) || unicode.IsDigit(current) || current == '_' || current == '.' || current == '-')
		if allowed {
			result.WriteRune(unicode.ToLower(current))
			lastReplacement = false
			continue
		}
		if !lastReplacement {
			result.WriteByte('-')
			lastReplacement = true
		}
	}
	tag := strings.Trim(result.String(), ".-")
	if len(tag) > 128 {
		tag = strings.TrimRight(tag[:128], ".-")
	}
	if tag == "" {
		return "", fmt.Errorf("image tag is empty after sanitization")
	}
	return tag, nil
}
