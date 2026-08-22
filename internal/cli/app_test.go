package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
)

func TestResolveConfigPrecedence(t *testing.T) {
	home := t.TempDir()
	path := filepath.Join(home, ".cloudivision", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("apiURL: https://file.example\ntoken: file-token\nnamespace: file-ns\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLOU_DIVISION_API_URL", "https://env.example")
	t.Setenv("CLOU_DIVISION_TOKEN", "env-token")
	config, gotPath, err := resolveConfig(globalOptions{APIURL: "https://flag.example", Namespace: "flag-ns"}, home)
	if err != nil {
		t.Fatal(err)
	}
	if gotPath != path || config.APIURL != "https://flag.example" || config.Token != "env-token" || config.Namespace != "flag-ns" {
		t.Fatalf("path=%q config=%#v", gotPath, config)
	}
}

func TestBuildTriggerGeneratesAPIRequest(t *testing.T) {
	var received struct {
		Name, Namespace string
		Spec            cicdv1alpha1.BuildRunSpec
	}
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/build-runs" {
			t.Fatalf("request = %s %s", r.Method, r.URL.Path)
		}
		if err := json.NewDecoder(r.Body).Decode(&received); err != nil {
			t.Fatal(err)
		}
		data, _ := json.Marshal(map[string]any{"name": received.Name, "namespace": received.Namespace, "spec": received.Spec})
		return jsonResponse(http.StatusCreated, string(data)), nil
	})
	app, stdout, stderr := testApp(t)
	app.HTTP.Transport = transport
	code := app.Run([]string{"--api-url", "https://api.test", "--namespace", "ci", "build", "trigger", "--name", "build-1", "--project", "project", "--repository", "repo", "--pipeline-template", "pipeline", "--revision", "abc123", "--branch", "main", "--param", "target=test"})
	if code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	if strings.TrimSpace(stdout.String()) != "build-1" || received.Namespace != "ci" || received.Spec.ProjectRef != "project" || received.Spec.Params["target"] != "test" {
		t.Fatalf("stdout=%q request=%#v", stdout.String(), received)
	}
}

func TestBuildWatchExitsForSuccessAndFailure(t *testing.T) {
	for _, test := range []struct {
		name  string
		final cicdv1alpha1.BuildRunPhase
		want  int
	}{{"success", cicdv1alpha1.BuildRunPhaseSucceeded, 0}, {"failure", cicdv1alpha1.BuildRunPhaseFailed, 1}} {
		t.Run(test.name, func(t *testing.T) {
			var calls atomic.Int32
			transport := roundTripFunc(func(_ *http.Request) (*http.Response, error) {
				phase := cicdv1alpha1.BuildRunPhaseRunning
				if calls.Add(1) > 1 {
					phase = test.final
				}
				data, _ := json.Marshal(map[string]any{"name": "build-1", "namespace": "ci", "status": map[string]any{"phase": phase}})
				return jsonResponse(http.StatusOK, string(data)), nil
			})
			app, _, _ := testApp(t)
			app.Sleep = func(time.Duration) {}
			app.HTTP.Transport = transport
			if code := app.Run([]string{"--api-url", "https://api.test", "--namespace", "ci", "build", "watch", "build-1", "--timeout", "1s", "--interval", "1ms"}); code != test.want {
				t.Fatalf("code=%d want=%d", code, test.want)
			}
		})
	}
}

func TestDoctorReportsActionableChecksWithFakeAPI(t *testing.T) {
	transport := roundTripFunc(func(r *http.Request) (*http.Response, error) {
		switch r.URL.Path {
		case "/healthz":
			return jsonResponse(http.StatusOK, `{"status":"ok"}`), nil
		case "/api/v1/auth/me":
			return jsonResponse(http.StatusOK, `{"subject":"developer"}`), nil
		case "/api/v1/providers/health":
			return jsonResponse(http.StatusOK, `[{"name":"argocd","type":"gitops","health":{"healthy":true,"message":"ready"}}]`), nil
		default:
			return jsonResponse(http.StatusNotFound, `{"code":"not_found","message":"not found"}`), nil
		}
	})
	app, stdout, stderr := testApp(t)
	app.Runner = fakeCommandRunner{}
	app.HTTP.Transport = transport
	if code := app.Run([]string{"--api-url", "https://api.test", "--namespace", "cloudivision", "doctor"}); code != 0 {
		t.Fatalf("code=%d stderr=%s", code, stderr.String())
	}
	for _, expected := range []string{"API health", "Authentication", "Provider health", "CRDs", "Runner RBAC"} {
		if !strings.Contains(stdout.String(), expected) {
			t.Fatalf("output missing %q: %s", expected, stdout.String())
		}
	}
}

type fakeCommandRunner struct{}

func (fakeCommandRunner) Run(context.Context, string, ...string) (string, error) { return "ok", nil }

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) { return f(request) }
func jsonResponse(status int, body string) *http.Response {
	return &http.Response{StatusCode: status, Status: http.StatusText(status), Header: http.Header{"Content-Type": {"application/json"}}, Body: io.NopCloser(strings.NewReader(body))}
}

func testApp(t *testing.T) (*App, *bytes.Buffer, *bytes.Buffer) {
	t.Helper()
	t.Setenv("CLOU_DIVISION_API_URL", "")
	t.Setenv("CLOU_DIVISION_TOKEN", "")
	t.Setenv("CLOU_DIVISION_NAMESPACE", "")
	out, errOut := &bytes.Buffer{}, &bytes.Buffer{}
	app := NewApp()
	app.Out = out
	app.Err = errOut
	app.HomeDir = func() (string, error) { return t.TempDir(), nil }
	return app, out, errOut
}
