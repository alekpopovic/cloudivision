package api

import (
	"encoding/csv"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/audit"
	"github.com/cloudivision/cloudivision/internal/redact"
)

type BuildReport struct {
	Total                  int            `json:"total"`
	Succeeded              int            `json:"succeeded"`
	Failed                 int            `json:"failed"`
	SuccessRate            float64        `json:"successRate"`
	FailureRate            float64        `json:"failureRate"`
	AverageDurationSeconds float64        `json:"averageDurationSeconds"`
	FailedByReason         map[string]int `json:"failedByReason"`
}

type ReleaseReport struct {
	Total              int            `json:"total"`
	ByEnvironment      map[string]int `json:"byEnvironment"`
	Approvals          int            `json:"approvals"`
	Rejections         int            `json:"rejections"`
	Rollbacks          int            `json:"rollbacks"`
	DeploymentFailures int            `json:"deploymentFailures"`
}

type SecurityReport struct {
	PolicyDenials               int `json:"policyDenials"`
	UnsignedReleasesBlocked     int `json:"unsignedReleasesBlocked"`
	WebhookRejections           int `json:"webhookRejections"`
	CriticalVulnerabilityBlocks int `json:"criticalVulnerabilityBlocks"`
}

func (s Server) auditExport(w http.ResponseWriter, r *http.Request) {
	filter, err := auditFilterFromRequest(r)
	if err != nil {
		s.writeError(w, badRequest(err.Error()))
		return
	}
	events, err := s.loadAuditEvents(r, filter)
	if err != nil {
		s.writeError(w, err)
		return
	}
	safe := make([]audit.Event, 0, len(events))
	for _, event := range events {
		safe = append(safe, safeAuditEvent(event))
	}
	if exportFormat(r) == "csv" {
		records := [][]string{{"id", "type", "actor", "organization", "project", "repository", "buildRun", "release", "eventId", "message", "metadata", "createdAt"}}
		for _, event := range safe {
			records = append(records, []string{event.ID, event.Type, event.Actor, event.Organization, event.Project, event.Repository, event.BuildRun, event.Release, event.EventID, event.Message, string(event.Metadata), event.CreatedAt.UTC().Format(time.RFC3339Nano)})
		}
		writeCSV(w, "audit-events.csv", records)
		return
	}
	items := make([]AuditEventResponse, 0, len(safe))
	for _, event := range safe {
		items = append(items, auditEventDTO(event))
	}
	writeJSON(w, http.StatusOK, items)
}

func (s Server) buildReport(w http.ResponseWriter, r *http.Request) {
	from, to, err := reportRange(r)
	if err != nil {
		s.writeError(w, badRequest(err.Error()))
		return
	}
	var list cicdv1alpha1.BuildRunList
	if err := s.listBuildRuns(r.Context(), r, &list); err != nil {
		s.writeError(w, err)
		return
	}
	orgProjects, err := s.organizationProjects(r)
	if err != nil {
		s.writeError(w, err)
		return
	}
	report := BuildReport{FailedByReason: map[string]int{}}
	durationTotal := 0.0
	durationCount := 0
	for _, build := range list.Items {
		if !matchesBuildReport(build, r, from, to, orgProjects) {
			continue
		}
		report.Total++
		switch build.Status.Phase {
		case cicdv1alpha1.BuildRunPhaseSucceeded:
			report.Succeeded++
		case cicdv1alpha1.BuildRunPhaseFailed:
			report.Failed++
			reason := build.Status.Failure.Reason
			if reason == "" {
				reason = "Unknown"
			}
			report.FailedByReason[reason]++
		}
		if build.Status.StartedAt != nil && build.Status.CompletedAt != nil && !build.Status.CompletedAt.Before(build.Status.StartedAt) {
			durationTotal += build.Status.CompletedAt.Sub(build.Status.StartedAt.Time).Seconds()
			durationCount++
		}
	}
	if report.Total > 0 {
		report.SuccessRate = float64(report.Succeeded) / float64(report.Total)
		report.FailureRate = float64(report.Failed) / float64(report.Total)
	}
	if durationCount > 0 {
		report.AverageDurationSeconds = durationTotal / float64(durationCount)
	}
	if exportFormat(r) == "csv" {
		records := [][]string{{"metric", "value"}, {"total", strconv.Itoa(report.Total)}, {"succeeded", strconv.Itoa(report.Succeeded)}, {"failed", strconv.Itoa(report.Failed)}, {"successRate", strconv.FormatFloat(report.SuccessRate, 'f', 6, 64)}, {"failureRate", strconv.FormatFloat(report.FailureRate, 'f', 6, 64)}, {"averageDurationSeconds", strconv.FormatFloat(report.AverageDurationSeconds, 'f', 3, 64)}}
		keys := sortedKeys(report.FailedByReason)
		for _, key := range keys {
			records = append(records, []string{"failedByReason." + key, strconv.Itoa(report.FailedByReason[key])})
		}
		writeCSV(w, "build-report.csv", records)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s Server) releaseReport(w http.ResponseWriter, r *http.Request) {
	from, to, err := reportRange(r)
	if err != nil {
		s.writeError(w, badRequest(err.Error()))
		return
	}
	var list cicdv1alpha1.ReleaseList
	if err := s.list(r.Context(), r, &list); err != nil {
		s.writeError(w, err)
		return
	}
	orgProjects, err := s.organizationProjects(r)
	if err != nil {
		s.writeError(w, err)
		return
	}
	report := ReleaseReport{ByEnvironment: map[string]int{}}
	for _, release := range list.Items {
		if !matchesReleaseReport(release, r, from, to, orgProjects) {
			continue
		}
		report.Total++
		report.ByEnvironment[release.Spec.EnvironmentRef]++
		if release.Spec.RollbackOf != "" || release.Status.Phase == cicdv1alpha1.ReleasePhaseRolledBack {
			report.Rollbacks++
		}
		if releasePhaseFailed(release.Status.Phase) {
			report.DeploymentFailures++
		}
	}
	filter, _ := auditFilterFromRequest(r)
	events, _ := s.loadAuditEvents(r, filter)
	for _, event := range events {
		switch event.Type {
		case "ReleaseApproved":
			report.Approvals++
		case "ReleaseRejected":
			report.Rejections++
		}
	}
	if exportFormat(r) == "csv" {
		records := [][]string{{"metric", "value"}, {"total", strconv.Itoa(report.Total)}, {"approvals", strconv.Itoa(report.Approvals)}, {"rejections", strconv.Itoa(report.Rejections)}, {"rollbacks", strconv.Itoa(report.Rollbacks)}, {"deploymentFailures", strconv.Itoa(report.DeploymentFailures)}}
		for _, key := range sortedKeys(report.ByEnvironment) {
			records = append(records, []string{"byEnvironment." + key, strconv.Itoa(report.ByEnvironment[key])})
		}
		writeCSV(w, "release-report.csv", records)
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func (s Server) securityReport(w http.ResponseWriter, r *http.Request) {
	filter, err := auditFilterFromRequest(r)
	if err != nil {
		s.writeError(w, badRequest(err.Error()))
		return
	}
	events, err := s.loadAuditEvents(r, filter)
	if err != nil {
		s.writeError(w, err)
		return
	}
	report := SecurityReport{}
	for _, event := range events {
		kind := strings.ToLower(event.Type + " " + event.Message + " " + string(event.Metadata))
		if strings.Contains(kind, "policydenied") || strings.Contains(kind, "policy_denied") {
			report.PolicyDenials++
		}
		if event.Type == "WebhookRejected" {
			report.WebhookRejections++
		}
		if strings.Contains(kind, "unsigned") && strings.Contains(kind, "block") {
			report.UnsignedReleasesBlocked++
		}
		if strings.Contains(kind, "critical") && strings.Contains(kind, "vulnerab") && strings.Contains(kind, "block") {
			report.CriticalVulnerabilityBlocks++
		}
	}
	if exportFormat(r) == "csv" {
		writeCSV(w, "security-report.csv", [][]string{{"metric", "value"}, {"policyDenials", strconv.Itoa(report.PolicyDenials)}, {"unsignedReleasesBlocked", strconv.Itoa(report.UnsignedReleasesBlocked)}, {"webhookRejections", strconv.Itoa(report.WebhookRejections)}, {"criticalVulnerabilityBlocks", strconv.Itoa(report.CriticalVulnerabilityBlocks)}})
		return
	}
	writeJSON(w, http.StatusOK, report)
}

func auditFilterFromRequest(r *http.Request) (audit.EventFilter, error) {
	from, err := optionalTime(r.URL.Query().Get("from"))
	if err != nil {
		return audit.EventFilter{}, fmt.Errorf("from: %w", err)
	}
	to, err := optionalTime(r.URL.Query().Get("to"))
	if err != nil {
		return audit.EventFilter{}, fmt.Errorf("to: %w", err)
	}
	if from != nil && to != nil && from.After(*to) {
		return audit.EventFilter{}, fmt.Errorf("from must not be after to")
	}
	return audit.EventFilter{Organization: r.URL.Query().Get("organization"), Project: r.URL.Query().Get("project"), Repository: r.URL.Query().Get("repository"), BuildRun: r.URL.Query().Get("buildRun"), Release: r.URL.Query().Get("release"), Actor: r.URL.Query().Get("actor"), Type: firstNonEmpty(r.URL.Query().Get("eventType"), r.URL.Query().Get("type")), From: from, To: to}, nil
}
func (s Server) loadAuditEvents(r *http.Request, filter audit.EventFilter) ([]audit.Event, error) {
	lister := s.AuditEvents
	if lister == nil {
		if cast, ok := s.Audit.(audit.EventLister); ok {
			lister = cast
		}
	}
	if lister == nil {
		return []audit.Event{}, nil
	}
	events, err := lister.ListEvents(r.Context(), filter)
	if err != nil {
		return nil, err
	}
	out := []audit.Event{}
	for _, event := range events {
		if matchesAuditFilter(event, filter) {
			out = append(out, event)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].CreatedAt.After(out[j].CreatedAt) })
	return out, nil
}
func matchesAuditFilter(e audit.Event, f audit.EventFilter) bool {
	return (f.Organization == "" || e.Organization == f.Organization) && (f.Project == "" || e.Project == f.Project) && (f.Repository == "" || e.Repository == f.Repository) && (f.BuildRun == "" || e.BuildRun == f.BuildRun) && (f.Release == "" || e.Release == f.Release) && (f.Actor == "" || e.Actor == f.Actor) && (f.Type == "" || e.Type == f.Type) && withinRange(e.CreatedAt, f.From, f.To)
}
func safeAuditEvent(event audit.Event) audit.Event {
	event.Message = redact.MaskString(event.Message)
	event.Metadata = safeMetadata(event.Metadata)
	return event
}
func safeMetadata(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 {
		return nil
	}
	var value any
	if json.Unmarshal(raw, &value) != nil {
		return json.RawMessage(`{"redacted":true}`)
	}
	sanitizeJSON(value)
	data, _ := json.Marshal(value)
	return data
}
func sanitizeJSON(value any) {
	switch current := value.(type) {
	case map[string]any:
		for key, item := range current {
			if redact.IsSecretKey(key) {
				current[key] = "[REDACTED]"
			} else {
				sanitizeJSON(item)
			}
		}
	case []any:
		for _, item := range current {
			sanitizeJSON(item)
		}
	}
}
func exportFormat(r *http.Request) string {
	if strings.EqualFold(r.URL.Query().Get("format"), "csv") {
		return "csv"
	}
	return "json"
}
func reportRange(r *http.Request) (*time.Time, *time.Time, error) {
	filter, err := auditFilterFromRequest(r)
	return filter.From, filter.To, err
}
func writeCSV(w http.ResponseWriter, name string, records [][]string) {
	w.Header().Set("Content-Type", "text/csv; charset=utf-8")
	w.Header().Set("Content-Disposition", `attachment; filename="`+name+`"`)
	writer := csv.NewWriter(w)
	_ = writer.WriteAll(records)
}
func sortedKeys(values map[string]int) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
func (s Server) organizationProjects(r *http.Request) (map[string]bool, error) {
	organization := r.URL.Query().Get("organization")
	if organization == "" {
		return nil, nil
	}
	var projects cicdv1alpha1.ProjectList
	if err := s.list(r.Context(), r, &projects); err != nil {
		return nil, err
	}
	out := map[string]bool{}
	for _, project := range projects.Items {
		ref := project.Spec.OrganizationRef
		if ref == "" {
			ref = "default"
		}
		if ref == organization {
			out[project.Namespace+"/"+project.Name] = true
		}
	}
	return out, nil
}
func matchesBuildReport(item cicdv1alpha1.BuildRun, r *http.Request, from, to *time.Time, projects map[string]bool) bool {
	q := r.URL.Query()
	return (q.Get("project") == "" || item.Spec.ProjectRef == q.Get("project")) && (q.Get("repository") == "" || item.Spec.RepositoryRef == q.Get("repository")) && (q.Get("buildRun") == "" || item.Name == q.Get("buildRun")) && (projects == nil || projects[item.Namespace+"/"+item.Spec.ProjectRef]) && withinRange(item.CreationTimestamp.Time, from, to)
}
func matchesReleaseReport(item cicdv1alpha1.Release, r *http.Request, from, to *time.Time, projects map[string]bool) bool {
	q := r.URL.Query()
	return (q.Get("project") == "" || item.Spec.ProjectRef == q.Get("project")) && (q.Get("repository") == "" || item.Spec.Image.Repository == q.Get("repository")) && (q.Get("release") == "" || item.Name == q.Get("release")) && (projects == nil || projects[item.Namespace+"/"+item.Spec.ProjectRef]) && withinRange(item.CreationTimestamp.Time, from, to)
}
