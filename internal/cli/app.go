package cli

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
)

type CommandRunner interface {
	Run(context.Context, string, ...string) (string, error)
}

type execRunner struct{}

func (execRunner) Run(ctx context.Context, name string, args ...string) (string, error) {
	data, err := exec.CommandContext(ctx, name, args...).CombinedOutput()
	return strings.TrimSpace(string(data)), err
}

type App struct {
	Out, Err io.Writer
	HTTP     *http.Client
	Runner   CommandRunner
	Sleep    func(time.Duration)
	HomeDir  func() (string, error)
	Now      func() time.Time
	Version  string
}

func NewApp() *App {
	return &App{Out: os.Stdout, Err: os.Stderr, HTTP: &http.Client{Timeout: 30 * time.Second}, Runner: execRunner{}, Sleep: time.Sleep, HomeDir: os.UserHomeDir, Now: time.Now, Version: "dev"}
}

func (a *App) Run(args []string) int {
	options, args, err := extractGlobals(args)
	if err != nil {
		return a.fail(err)
	}
	home, err := a.HomeDir()
	if err != nil {
		return a.fail(err)
	}
	config, configPath, err := resolveConfig(options, home)
	if err != nil {
		return a.fail(err)
	}
	if len(args) == 0 {
		a.usage()
		return 2
	}
	client := apiClient{baseURL: config.APIURL, token: config.Token, http: a.HTTP}
	ctx := context.Background()
	switch args[0] {
	case "version":
		return a.printValue(map[string]string{"version": a.Version}, "cloudivision "+a.Version, options.Output)
	case "login":
		if config.Token == "" {
			return a.fail(fmt.Errorf("token is required through --token or CLOU_DIVISION_TOKEN"))
		}
		if err := saveConfig(configPath, config); err != nil {
			return a.fail(err)
		}
		fmt.Fprintf(a.Out, "Saved API URL and token to %s\n", configPath)
		return 0
	case "project":
		return a.project(ctx, client, config, options.Output, args[1:])
	case "repo":
		return a.repo(ctx, client, config, options.Output, args[1:])
	case "pipeline":
		return a.pipeline(ctx, client, config, options.Output, args[1:])
	case "build":
		return a.build(ctx, client, config, options.Output, args[1:])
	case "release":
		return a.release(ctx, client, config, options.Output, args[1:])
	case "doctor":
		return a.doctor(ctx, client, config, options.Output)
	case "help", "--help", "-h":
		a.usage()
		return 0
	default:
		return a.fail(fmt.Errorf("unknown command %q", args[0]))
	}
}

type resourceSummary struct {
	Name, Namespace string
	Spec            json.RawMessage
	Status          struct {
		Phase string `json:"phase"`
	}
}
type buildRun struct {
	Name, Namespace string
	Spec            cicdv1alpha1.BuildRunSpec
	Status          cicdv1alpha1.BuildRunStatus
}
type release struct {
	Name, Namespace string
	Spec            cicdv1alpha1.ReleaseSpec
	Status          cicdv1alpha1.ReleaseStatus
}
type page[T any] struct {
	Items         []T    `json:"items"`
	NextPageToken string `json:"nextPageToken,omitempty"`
	TotalCount    int    `json:"totalCount"`
	Limit         int    `json:"limit"`
}

func (a *App) project(ctx context.Context, client apiClient, config Config, output string, args []string) int {
	if len(args) == 0 {
		return a.fail(fmt.Errorf("project requires list or create"))
	}
	switch args[0] {
	case "list":
		var items []resourceSummary
		if err := client.do(ctx, http.MethodGet, query("/api/v1/projects", url.Values{"namespace": {config.Namespace}}), nil, &items); err != nil {
			return a.fail(err)
		}
		return a.printResources(items, output)
	case "create":
		fs := newFlags("project create", a.Err)
		name := fs.String("name", "", "name")
		display := fs.String("display-name", "", "display name")
		owner := fs.String("owner-team", "", "owner team")
		targetNS := fs.String("project-namespace", config.Namespace, "project workload namespace")
		registry := fs.String("default-registry", "", "default registry")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if *name == "" || *display == "" || *owner == "" || *registry == "" {
			return a.fail(fmt.Errorf("--name, --display-name, --owner-team and --default-registry are required"))
		}
		body := map[string]any{"name": *name, "namespace": config.Namespace, "spec": map[string]any{"displayName": *display, "ownerTeam": *owner, "namespace": *targetNS, "defaultRegistry": *registry, "defaultBranch": "main", "isolation": map[string]any{"createNamespace": false, "podSecurityLevel": "restricted", "networkPolicyMode": "disabled"}}}
		var created resourceSummary
		if err := client.do(ctx, http.MethodPost, "/api/v1/projects", body, &created); err != nil {
			return a.fail(err)
		}
		return a.printValue(created, created.Name, output)
	default:
		return a.fail(fmt.Errorf("unknown project command %q", args[0]))
	}
}

func (a *App) repo(ctx context.Context, client apiClient, config Config, output string, args []string) int {
	if len(args) == 0 {
		return a.fail(fmt.Errorf("repo requires list or add"))
	}
	if args[0] == "list" {
		var items []resourceSummary
		if err := client.do(ctx, http.MethodGet, query("/api/v1/repositories", url.Values{"namespace": {config.Namespace}}), nil, &items); err != nil {
			return a.fail(err)
		}
		return a.printResources(items, output)
	}
	if args[0] != "add" {
		return a.fail(fmt.Errorf("unknown repo command %q", args[0]))
	}
	fs := newFlags("repo add", a.Err)
	name := fs.String("name", "", "name")
	project := fs.String("project", "", "project")
	provider := fs.String("provider", "generic", "provider")
	repositoryURL := fs.String("url", "", "URL")
	branch := fs.String("default-branch", "main", "default branch")
	pipeline := fs.String("pipeline-template", "", "pipeline template")
	if err := fs.Parse(args[1:]); err != nil {
		return 2
	}
	if *name == "" || *project == "" || *repositoryURL == "" || *pipeline == "" {
		return a.fail(fmt.Errorf("--name, --project, --url and --pipeline-template are required"))
	}
	body := map[string]any{"name": *name, "namespace": config.Namespace, "spec": map[string]any{"projectRef": *project, "provider": *provider, "url": *repositoryURL, "defaultBranch": *branch, "pipelineTemplateRef": *pipeline}}
	var created resourceSummary
	if err := client.do(ctx, http.MethodPost, "/api/v1/repositories", body, &created); err != nil {
		return a.fail(err)
	}
	return a.printValue(created, created.Name, output)
}

func (a *App) pipeline(ctx context.Context, client apiClient, config Config, output string, args []string) int {
	if len(args) != 1 || args[0] != "list" {
		return a.fail(fmt.Errorf("pipeline supports list"))
	}
	var items []resourceSummary
	if err := client.do(ctx, http.MethodGet, query("/api/v1/pipeline-templates", url.Values{"namespace": {config.Namespace}}), nil, &items); err != nil {
		return a.fail(err)
	}
	return a.printResources(items, output)
}

type stringMapFlag map[string]string

func (m *stringMapFlag) String() string { return fmt.Sprint(map[string]string(*m)) }
func (m *stringMapFlag) Set(value string) error {
	key, val, ok := strings.Cut(value, "=")
	if !ok || key == "" {
		return fmt.Errorf("param must be key=value")
	}
	(*m)[key] = val
	return nil
}

func (a *App) build(ctx context.Context, client apiClient, config Config, output string, args []string) int {
	if len(args) == 0 {
		return a.fail(fmt.Errorf("build requires trigger, list, get, logs, watch, cancel, retry, or rerun"))
	}
	switch args[0] {
	case "trigger":
		fs := newFlags("build trigger", a.Err)
		name := fs.String("name", "", "BuildRun name")
		project := fs.String("project", "", "project")
		repository := fs.String("repository", "", "repository")
		pipeline := fs.String("pipeline-template", "", "pipeline template")
		revision := fs.String("revision", "main", "revision")
		branch := fs.String("branch", "", "branch")
		image := fs.String("image", "example.invalid/cloudivision-cli", "output image repository")
		watch := fs.Bool("watch", false, "watch to terminal phase")
		timeout := fs.Duration("timeout", 30*time.Minute, "watch timeout")
		params := stringMapFlag{}
		fs.Var(&params, "param", "key=value (repeatable)")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		if *project == "" || *repository == "" || *pipeline == "" {
			return a.fail(fmt.Errorf("--project, --repository and --pipeline-template are required"))
		}
		if *name == "" {
			*name = "cli-" + strconv.FormatInt(a.Now().UnixNano(), 36)
		}
		request := map[string]any{"name": *name, "namespace": config.Namespace, "spec": cicdv1alpha1.BuildRunSpec{ProjectRef: *project, RepositoryRef: *repository, PipelineTemplateRef: *pipeline, Revision: *revision, Branch: *branch, TriggeredBy: cicdv1alpha1.TriggeredBy{Type: cicdv1alpha1.TriggerTypeManual, Actor: "cloudivision-cli"}, Image: cicdv1alpha1.ImageRef{Repository: *image, Tag: "manual"}, Params: params, Executor: cicdv1alpha1.ExecutorTypeJob}}
		var created buildRun
		if err := client.do(ctx, http.MethodPost, "/api/v1/build-runs", request, &created); err != nil {
			return a.fail(err)
		}
		if output == "json" {
			_ = writeJSON(a.Out, created)
		} else {
			fmt.Fprintln(a.Out, created.Name)
		}
		if *watch {
			return a.watchBuild(ctx, client, config.Namespace, created.Name, *timeout, time.Second, output)
		}
		return 0
	case "list":
		fs := newFlags("build list", a.Err)
		phase := fs.String("phase", "", "phase")
		project := fs.String("project", "", "project")
		repository := fs.String("repository", "", "repository")
		limit := fs.Int("limit", 50, "page size (maximum 200)")
		pageToken := fs.String("page-token", "", "opaque token returned by the previous page")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		values := url.Values{"namespace": {config.Namespace}, "limit": {strconv.Itoa(*limit)}}
		if *pageToken != "" {
			values.Set("pageToken", *pageToken)
		}
		if *phase != "" {
			values.Set("phase", *phase)
		}
		if *project != "" {
			values.Set("project", *project)
		}
		if *repository != "" {
			values.Set("repository", *repository)
		}
		var response page[buildRun]
		if err := client.do(ctx, http.MethodGet, query("/api/v1/build-runs", values), nil, &response); err != nil {
			return a.fail(err)
		}
		if output == "json" {
			_ = writeJSON(a.Out, response)
			return 0
		}
		rows := [][]string{}
		for _, item := range response.Items {
			rows = append(rows, []string{item.Namespace, item.Name, item.Spec.ProjectRef, item.Spec.RepositoryRef, string(item.Status.Phase)})
		}
		printTable(a.Out, []string{"NAMESPACE", "NAME", "PROJECT", "REPOSITORY", "PHASE"}, rows)
		if response.NextPageToken != "" {
			fmt.Fprintf(a.Out, "NEXT_PAGE_TOKEN\t%s\n", response.NextPageToken)
		}
		return 0
	case "get":
		if len(args) != 2 {
			return a.fail(fmt.Errorf("usage: cloudivision build get NAME"))
		}
		var item buildRun
		if err := client.do(ctx, http.MethodGet, "/api/v1/build-runs/"+url.PathEscape(config.Namespace)+"/"+url.PathEscape(args[1]), nil, &item); err != nil {
			return a.fail(err)
		}
		return a.printValue(item, fmt.Sprintf("%s\t%s\t%s\n", item.Name, item.Status.Phase, item.Status.Failure.Message), output)
	case "watch":
		watchArgs := args[1:]
		name := ""
		if len(watchArgs) > 0 && !strings.HasPrefix(watchArgs[0], "-") {
			name, watchArgs = watchArgs[0], watchArgs[1:]
		}
		fs := newFlags("build watch", a.Err)
		timeout := fs.Duration("timeout", 30*time.Minute, "timeout")
		interval := fs.Duration("interval", 2*time.Second, "poll interval")
		if err := fs.Parse(watchArgs); err != nil {
			return 2
		}
		if name == "" && fs.NArg() == 1 {
			name = fs.Arg(0)
		}
		if name == "" || fs.NArg() > 0 {
			return a.fail(fmt.Errorf("usage: cloudivision build watch NAME"))
		}
		return a.watchBuild(ctx, client, config.Namespace, name, *timeout, *interval, output)
	case "logs":
		return a.buildLogs(ctx, client, config, args[1:])
	case "cancel", "retry", "rerun":
		if len(args) != 2 {
			return a.fail(fmt.Errorf("usage: cloudivision build %s NAME", args[0]))
		}
		var item buildRun
		path := fmt.Sprintf("/api/v1/build-runs/%s/%s/%s", url.PathEscape(config.Namespace), url.PathEscape(args[1]), args[0])
		if err := client.do(ctx, http.MethodPost, path, nil, &item); err != nil {
			return a.fail(err)
		}
		return a.printValue(item, item.Name+" "+args[0]+" requested", output)
	default:
		return a.fail(fmt.Errorf("unknown build command %q", args[0]))
	}
}

func (a *App) watchBuild(ctx context.Context, client apiClient, namespace, name string, timeout, interval time.Duration, output string) int {
	deadline := a.Now().Add(timeout)
	last := ""
	for !a.Now().After(deadline) {
		var item buildRun
		if err := client.do(ctx, http.MethodGet, "/api/v1/build-runs/"+url.PathEscape(namespace)+"/"+url.PathEscape(name), nil, &item); err != nil {
			return a.fail(err)
		}
		phase := string(item.Status.Phase)
		if phase != last && output != "json" {
			fmt.Fprintf(a.Out, "%s\t%s\n", name, first(phase, "Pending"))
			last = phase
		}
		switch item.Status.Phase {
		case cicdv1alpha1.BuildRunPhaseSucceeded:
			if output == "json" {
				_ = writeJSON(a.Out, item)
			}
			return 0
		case cicdv1alpha1.BuildRunPhaseFailed, cicdv1alpha1.BuildRunPhaseCancelled:
			if output == "json" {
				_ = writeJSON(a.Out, item)
			}
			return 1
		}
		a.Sleep(interval)
	}
	fmt.Fprintf(a.Err, "timed out waiting for BuildRun %s\n", name)
	return 1
}

func (a *App) buildLogs(ctx context.Context, client apiClient, config Config, args []string) int {
	name := ""
	if len(args) > 0 && !strings.HasPrefix(args[0], "-") {
		name, args = args[0], args[1:]
	}
	fs := newFlags("build logs", a.Err)
	tail := fs.Int("tail", 200, "tail lines")
	follow := fs.Bool("follow", false, "poll logs")
	interval := fs.Duration("interval", 2*time.Second, "poll interval")
	if err := fs.Parse(args); err != nil {
		return 2
	}
	if name == "" && fs.NArg() == 1 {
		name = fs.Arg(0)
	}
	if name == "" || fs.NArg() > 0 {
		return a.fail(fmt.Errorf("usage: cloudivision build logs NAME"))
	}
	printed := 0
	for {
		var response struct {
			Lines []string `json:"lines"`
		}
		path := query("/api/v1/build-runs/"+url.PathEscape(config.Namespace)+"/"+url.PathEscape(name)+"/logs", url.Values{"tailLines": {strconv.Itoa(*tail)}})
		if err := client.do(ctx, http.MethodGet, path, nil, &response); err != nil {
			return a.fail(err)
		}
		if printed > len(response.Lines) {
			printed = 0
		}
		for _, line := range response.Lines[printed:] {
			fmt.Fprintln(a.Out, line)
		}
		printed = len(response.Lines)
		if !*follow {
			return 0
		}
		var item buildRun
		if err := client.do(ctx, http.MethodGet, "/api/v1/build-runs/"+url.PathEscape(config.Namespace)+"/"+url.PathEscape(name), nil, &item); err != nil {
			return a.fail(err)
		}
		if item.Status.Phase == cicdv1alpha1.BuildRunPhaseSucceeded || item.Status.Phase == cicdv1alpha1.BuildRunPhaseFailed || item.Status.Phase == cicdv1alpha1.BuildRunPhaseCancelled {
			return 0
		}
		a.Sleep(*interval)
	}
}

func (a *App) release(ctx context.Context, client apiClient, config Config, output string, args []string) int {
	if len(args) == 0 {
		return a.fail(fmt.Errorf("release requires list, get, approve, reject, or rollback"))
	}
	list := func(limit int, token string) (page[release], error) {
		var response page[release]
		values := url.Values{"namespace": {config.Namespace}, "limit": {strconv.Itoa(limit)}}
		if token != "" {
			values.Set("pageToken", token)
		}
		err := client.do(ctx, http.MethodGet, query("/api/v1/releases", values), nil, &response)
		return response, err
	}
	switch args[0] {
	case "list":
		fs := newFlags("release list", a.Err)
		limit := fs.Int("limit", 50, "page size (maximum 200)")
		pageToken := fs.String("page-token", "", "opaque token returned by the previous page")
		if err := fs.Parse(args[1:]); err != nil {
			return 2
		}
		response, err := list(*limit, *pageToken)
		if err != nil {
			return a.fail(err)
		}
		if output == "json" {
			_ = writeJSON(a.Out, response)
			return 0
		}
		rows := [][]string{}
		for _, item := range response.Items {
			rows = append(rows, []string{item.Namespace, item.Name, item.Spec.EnvironmentRef, string(item.Status.Phase)})
		}
		printTable(a.Out, []string{"NAMESPACE", "NAME", "ENVIRONMENT", "PHASE"}, rows)
		if response.NextPageToken != "" {
			fmt.Fprintf(a.Out, "NEXT_PAGE_TOKEN\t%s\n", response.NextPageToken)
		}
		return 0
	case "get":
		if len(args) != 2 {
			return a.fail(fmt.Errorf("usage: cloudivision release get NAME"))
		}
		token := ""
		for {
			response, err := list(200, token)
			if err != nil {
				return a.fail(err)
			}
			for _, item := range response.Items {
				if item.Name == args[1] {
					return a.printValue(item, fmt.Sprintf("%s\t%s\n", item.Name, item.Status.Phase), output)
				}
			}
			if response.NextPageToken == "" {
				break
			}
			token = response.NextPageToken
		}
		return a.fail(fmt.Errorf("release %q not found", args[1]))
	case "approve", "reject":
		actionArgs := args[1:]
		name := ""
		if len(actionArgs) > 0 && !strings.HasPrefix(actionArgs[0], "-") {
			name, actionArgs = actionArgs[0], actionArgs[1:]
		}
		fs := newFlags("release "+args[0], a.Err)
		actor := fs.String("actor", "cloudivision-cli", "actor")
		comment := fs.String("comment", "", "comment")
		if err := fs.Parse(actionArgs); err != nil {
			return 2
		}
		if name == "" && fs.NArg() == 1 {
			name = fs.Arg(0)
		}
		if name == "" || fs.NArg() > 0 {
			return a.fail(fmt.Errorf("release name is required"))
		}
		var item release
		path := fmt.Sprintf("/api/v1/releases/%s/%s/%s", url.PathEscape(config.Namespace), url.PathEscape(name), args[0])
		if err := client.do(ctx, http.MethodPost, path, map[string]string{"actor": *actor, "comment": *comment}, &item); err != nil {
			return a.fail(err)
		}
		return a.printValue(item, item.Name+" "+args[0]+"d", output)
	case "rollback":
		return a.fail(fmt.Errorf("rollback is not available in the v0.1 API; revert the GitOps change and create an auditable replacement Release"))
	default:
		return a.fail(fmt.Errorf("unknown release command %q", args[0]))
	}
}

type doctorCheck struct{ Name, Status, Message string }

func (a *App) doctor(ctx context.Context, client apiClient, config Config, output string) int {
	checks := []doctorCheck{}
	var health map[string]any
	if err := client.do(ctx, http.MethodGet, "/healthz", nil, &health); err != nil {
		checks = append(checks, doctorCheck{"API health", "FAIL", err.Error()})
	} else {
		checks = append(checks, doctorCheck{"API health", "PASS", client.baseURL})
	}
	var principal map[string]any
	if err := client.do(ctx, http.MethodGet, "/api/v1/auth/me", nil, &principal); err != nil {
		checks = append(checks, doctorCheck{"Authentication", "FAIL", err.Error()})
	} else {
		checks = append(checks, doctorCheck{"Authentication", "PASS", "credentials accepted"})
	}
	var providers []struct {
		Name, Type string
		Health     struct {
			Healthy bool
			Message string
		}
	}
	if err := client.do(ctx, http.MethodGet, "/api/v1/providers/health", nil, &providers); err != nil {
		checks = append(checks, doctorCheck{"Provider health", "WARN", err.Error()})
	} else {
		unhealthy := []string{}
		for _, p := range providers {
			if !p.Health.Healthy {
				unhealthy = append(unhealthy, p.Type+"/"+p.Name)
			}
		}
		if len(unhealthy) == 0 {
			checks = append(checks, doctorCheck{"Provider health", "PASS", fmt.Sprintf("%d providers healthy", len(providers))})
		} else {
			checks = append(checks, doctorCheck{"Provider health", "WARN", "unhealthy: " + strings.Join(unhealthy, ", ")})
		}
	}
	commandChecks := []struct {
		name string
		args []string
	}{
		{"CRDs", []string{"get", "crd", "buildruns.cicd.cloudivision.io"}},
		{"Deployments", []string{"-n", config.Namespace, "get", "deploy", "-l", "app.kubernetes.io/name=cloudivision"}},
		{"Runner image", []string{"-n", config.Namespace, "get", "configmap", "-l", "app.kubernetes.io/name=cloudivision", "-o", "jsonpath={.items[0].data.CLOU_DIVISION_RUNNER_IMAGE}"}},
		{"Runner RBAC", []string{"auth", "can-i", "create", "jobs", "-n", config.Namespace}},
	}
	for _, check := range commandChecks {
		commandCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
		text, err := a.Runner.Run(commandCtx, "kubectl", check.args...)
		cancel()
		if err != nil {
			checks = append(checks, doctorCheck{check.name, "WARN", "kubectl unavailable or check failed: " + first(text, err.Error())})
		} else {
			checks = append(checks, doctorCheck{check.name, "PASS", first(text, "available")})
		}
	}
	if output == "json" {
		_ = writeJSON(a.Out, checks)
	} else {
		rows := [][]string{}
		for _, check := range checks {
			rows = append(rows, []string{check.Status, check.Name, check.Message})
		}
		printTable(a.Out, []string{"STATUS", "CHECK", "DETAIL"}, rows)
	}
	for _, check := range checks {
		if check.Status == "FAIL" {
			return 1
		}
	}
	return 0
}

func (a *App) printResources(items []resourceSummary, output string) int {
	if output == "json" {
		_ = writeJSON(a.Out, items)
		return 0
	}
	rows := [][]string{}
	for _, item := range items {
		rows = append(rows, []string{item.Namespace, item.Name, item.Status.Phase})
	}
	printTable(a.Out, []string{"NAMESPACE", "NAME", "PHASE"}, rows)
	return 0
}
func (a *App) printValue(value any, human, output string) int {
	if output == "json" {
		if err := writeJSON(a.Out, value); err != nil {
			return a.fail(err)
		}
	} else {
		fmt.Fprintln(a.Out, strings.TrimSuffix(human, "\n"))
	}
	return 0
}
func (a *App) fail(err error) int { fmt.Fprintln(a.Err, "error:", err); return 1 }
func (a *App) usage() {
	fmt.Fprintln(a.Out, "Usage: cloudivision [--api-url URL] [--token TOKEN] [--namespace NS] [--output table|json] COMMAND\nCommands: version, login, project, repo, pipeline, build, release, doctor")
}
func newFlags(name string, output io.Writer) *flag.FlagSet {
	fs := flag.NewFlagSet(name, flag.ContinueOnError)
	fs.SetOutput(output)
	return fs
}
func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}
func printTable(output io.Writer, headers []string, rows [][]string) {
	writer := tabwriter.NewWriter(output, 0, 4, 2, ' ', 0)
	fmt.Fprintln(writer, strings.Join(headers, "\t"))
	for _, row := range rows {
		fmt.Fprintln(writer, strings.Join(row, "\t"))
	}
	_ = writer.Flush()
}
