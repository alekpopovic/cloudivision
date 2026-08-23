package api

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/audit"
)

const (
	defaultPageLimit = 50
	maximumPageLimit = 200
)

type pageOptions struct {
	limit  int
	offset int
	sort   string
	order  string
	from   *time.Time
	to     *time.Time
}

type pageToken struct {
	Version int `json:"v"`
	Offset  int `json:"o"`
}

func parsePageOptions(r *http.Request, allowedSorts map[string]bool) (pageOptions, error) {
	query := r.URL.Query()
	limit, err := boundedInt(query.Get("limit"), defaultPageLimit, 1, maximumPageLimit)
	if err != nil {
		return pageOptions{}, fmt.Errorf("limit: %w", err)
	}
	tokenValue := query.Get("pageToken")
	if alternate := query.Get("continue"); alternate != "" {
		if tokenValue != "" && tokenValue != alternate {
			return pageOptions{}, fmt.Errorf("continue and pageToken must match when both are supplied")
		}
		tokenValue = alternate
	}
	offset := 0
	if tokenValue != "" {
		decoded, decodeErr := base64.RawURLEncoding.DecodeString(tokenValue)
		if decodeErr != nil {
			return pageOptions{}, fmt.Errorf("pageToken: invalid token")
		}
		var token pageToken
		if json.Unmarshal(decoded, &token) != nil || token.Version != 1 || token.Offset < 0 {
			return pageOptions{}, fmt.Errorf("pageToken: invalid token")
		}
		offset = token.Offset
	} else if query.Has("offset") { // Compatibility with the v0.1 BuildRun API.
		offset, err = boundedInt(query.Get("offset"), 0, 0, 1_000_000_000)
		if err != nil {
			return pageOptions{}, fmt.Errorf("offset: %w", err)
		}
	}
	sortField := strings.TrimSpace(query.Get("sort"))
	if sortField == "" {
		sortField = "createdAt"
	}
	if !allowedSorts[sortField] {
		return pageOptions{}, fmt.Errorf("sort: unsupported field %q", sortField)
	}
	order := strings.ToLower(strings.TrimSpace(query.Get("order")))
	if order == "" {
		order = "desc"
	}
	if order != "asc" && order != "desc" {
		return pageOptions{}, fmt.Errorf("order: must be asc or desc")
	}
	from, err := optionalTime(firstNonEmpty(query.Get("from"), query.Get("createdAfter")))
	if err != nil {
		return pageOptions{}, fmt.Errorf("from: %w", err)
	}
	to, err := optionalTime(firstNonEmpty(query.Get("to"), query.Get("createdBefore")))
	if err != nil {
		return pageOptions{}, fmt.Errorf("to: %w", err)
	}
	if from != nil && to != nil && from.After(*to) {
		return pageOptions{}, fmt.Errorf("from must not be after to")
	}
	return pageOptions{limit: limit, offset: offset, sort: sortField, order: order, from: from, to: to}, nil
}

func paginate[T any](items []T, options pageOptions) ([]T, string) {
	if options.offset >= len(items) {
		return []T{}, ""
	}
	end := options.offset + options.limit
	if end > len(items) {
		end = len(items)
	}
	next := ""
	if end < len(items) {
		encoded, _ := json.Marshal(pageToken{Version: 1, Offset: end})
		next = base64.RawURLEncoding.EncodeToString(encoded)
	}
	return items[options.offset:end], next
}

func filterSortBuildRuns(r *http.Request, source []cicdv1alpha1.BuildRun, options pageOptions) []cicdv1alpha1.BuildRun {
	query := r.URL.Query()
	filtered := make([]cicdv1alpha1.BuildRun, 0, len(source))
	for _, item := range source {
		if phase := query.Get("phase"); phase != "" && !strings.EqualFold(string(item.Status.Phase), phase) {
			continue
		}
		if project := query.Get("project"); project != "" && item.Spec.ProjectRef != project {
			continue
		}
		if repository := query.Get("repository"); repository != "" && item.Spec.RepositoryRef != repository {
			continue
		}
		if !withinRange(item.CreationTimestamp.Time, options.from, options.to) {
			continue
		}
		filtered = append(filtered, item)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		comparison := compareBuildRuns(filtered[i], filtered[j], options.sort)
		if options.order == "desc" {
			return comparison > 0
		}
		return comparison < 0
	})
	return filtered
}

func compareBuildRuns(left, right cicdv1alpha1.BuildRun, field string) int {
	switch field {
	case "name":
		return strings.Compare(left.Namespace+"/"+left.Name, right.Namespace+"/"+right.Name)
	case "phase":
		if value := strings.Compare(string(left.Status.Phase), string(right.Status.Phase)); value != 0 {
			return value
		}
	default:
		if value := left.CreationTimestamp.Time.Compare(right.CreationTimestamp.Time); value != 0 {
			return value
		}
	}
	return strings.Compare(left.Namespace+"/"+left.Name, right.Namespace+"/"+right.Name)
}

func filterSortReleases(r *http.Request, source []cicdv1alpha1.Release, options pageOptions) []cicdv1alpha1.Release {
	query := r.URL.Query()
	filtered := make([]cicdv1alpha1.Release, 0, len(source))
	for _, item := range source {
		if phase := query.Get("phase"); phase != "" && !strings.EqualFold(string(item.Status.Phase), phase) {
			continue
		}
		if project := query.Get("project"); project != "" && item.Spec.ProjectRef != project {
			continue
		}
		if repository := query.Get("repository"); repository != "" && item.Spec.Image.Repository != repository {
			continue
		}
		if !withinRange(item.CreationTimestamp.Time, options.from, options.to) {
			continue
		}
		filtered = append(filtered, item)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		var comparison int
		switch options.sort {
		case "name":
			comparison = strings.Compare(filtered[i].Namespace+"/"+filtered[i].Name, filtered[j].Namespace+"/"+filtered[j].Name)
		case "phase":
			comparison = strings.Compare(string(filtered[i].Status.Phase), string(filtered[j].Status.Phase))
		default:
			comparison = filtered[i].CreationTimestamp.Time.Compare(filtered[j].CreationTimestamp.Time)
		}
		if comparison == 0 {
			comparison = strings.Compare(filtered[i].Namespace+"/"+filtered[i].Name, filtered[j].Namespace+"/"+filtered[j].Name)
		}
		return options.order == "desc" && comparison > 0 || options.order == "asc" && comparison < 0
	})
	return filtered
}

func filterSortAuditEvents(source []audit.Event, options pageOptions) []audit.Event {
	filtered := make([]audit.Event, 0, len(source))
	for _, item := range source {
		if !withinRange(item.CreatedAt, options.from, options.to) {
			continue
		}
		filtered = append(filtered, item)
	}
	sort.SliceStable(filtered, func(i, j int) bool {
		var comparison int
		if options.sort == "type" {
			comparison = strings.Compare(filtered[i].Type, filtered[j].Type)
		} else {
			comparison = filtered[i].CreatedAt.Compare(filtered[j].CreatedAt)
		}
		if comparison == 0 {
			comparison = strings.Compare(filtered[i].ID, filtered[j].ID)
		}
		return options.order == "desc" && comparison > 0 || options.order == "asc" && comparison < 0
	})
	return filtered
}

func withinRange(value time.Time, from, to *time.Time) bool {
	return (from == nil || !value.Before(*from)) && (to == nil || !value.After(*to))
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
