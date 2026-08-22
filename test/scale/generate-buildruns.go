package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"regexp"
	"strings"
)

func main() {
	count := flag.Int("count", 100, "number of BuildRuns")
	projects := flag.Int("projects", 1, "number of Project/Repository/PipelineTemplate sets")
	namespace := flag.String("namespace", "cloudivision-scale", "target namespace")
	runID := flag.String("run-id", "local", "cleanup label and name suffix")
	repositoryURL := flag.String("repository-url", "https://github.com/docker/getting-started-todo-app.git", "source repository URL")
	gitOps := flag.Bool("gitops", false, "generate GitOps-enabled BuildRuns and Environments")
	flag.Parse()
	if *count < 1 || *count > 10000 || *projects < 1 || *projects > 100 || *projects > *count {
		fmt.Fprintln(os.Stderr, "count must be 1..10000 and projects must be 1..min(100,count)")
		os.Exit(2)
	}
	cleanID := dnsLabel(*runID)
	labels := map[string]any{"cloudivision.io/scale-run": cleanID}
	items := make([]any, 0, *count+*projects*4)
	for i := 0; i < *projects; i++ {
		name := fmt.Sprintf("scale-%s-p%02d", cleanID, i)
		items = append(items,
			resource("Project", *namespace, name, labels, map[string]any{
				"displayName": "Scale test " + name, "ownerTeam": "scale-test", "namespace": *namespace,
				"defaultRegistry": "example.invalid", "defaultBranch": "main", "serviceAccountName": "cloudivision-runner",
				"isolation": map[string]any{"createNamespace": false, "podSecurityLevel": "restricted", "networkPolicyMode": "disabled"},
			}),
			resource("Repository", *namespace, name, labels, map[string]any{
				"projectRef": name, "provider": "generic", "url": *repositoryURL, "defaultBranch": "main", "pipelineTemplateRef": name,
			}),
			resource("PipelineTemplate", *namespace, name, labels, map[string]any{
				"projectRef": name,
				"steps":      []any{map[string]any{"name": "verify", "image": "alpine:3.22", "command": []string{"sh"}, "args": []string{"-c", "test -f client/package.json && test -f Dockerfile"}, "timeoutSeconds": 120}},
				"build":      map[string]any{"enabled": false, "builder": "none", "push": false},
				"resources":  map[string]any{"cpuRequest": "25m", "cpuLimit": "250m", "memoryRequest": "32Mi", "memoryLimit": "256Mi", "timeoutSeconds": 300},
				"security":   map[string]any{"allowPrivileged": false, "runAsNonRoot": true, "readOnlyRootFilesystem": false},
			}),
		)
		if *gitOps {
			items = append(items, resource("Environment", *namespace, name, labels, map[string]any{
				"projectRef": name, "displayName": name, "namespace": *namespace, "type": "dev", "requiresApproval": false,
				"gitOps": map[string]any{"provider": "generic"},
			}))
		}
	}
	for i := 0; i < *count; i++ {
		project := fmt.Sprintf("scale-%s-p%02d", cleanID, i%*projects)
		spec := map[string]any{
			"projectRef": project, "repositoryRef": project, "pipelineTemplateRef": project, "revision": "main", "branch": "main",
			"triggeredBy": map[string]any{"type": "manual", "actor": "scale-test"},
			"image":       map[string]any{"repository": "example.invalid/scale", "tag": fmt.Sprintf("run-%04d", i)}, "executor": "job",
			"gitOps": map[string]any{"enabled": false},
		}
		if *gitOps {
			spec["gitOps"] = map[string]any{
				"enabled": true, "repoURL": *repositoryURL, "branch": "main", "path": fmt.Sprintf("scale/%04d", i),
				"strategy": "raw-yaml", "environmentRef": project,
			}
		}
		items = append(items, resource("BuildRun", *namespace, fmt.Sprintf("scale-%s-%04d", cleanID, i), labels, spec))
	}
	if err := json.NewEncoder(os.Stdout).Encode(map[string]any{"apiVersion": "v1", "kind": "List", "items": items}); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func resource(kind, namespace, name string, labels map[string]any, spec map[string]any) map[string]any {
	return map[string]any{
		"apiVersion": "cicd.cloudivision.io/v1alpha1", "kind": kind,
		"metadata": map[string]any{"name": name, "namespace": namespace, "labels": labels}, "spec": spec,
	}
}

var invalidDNS = regexp.MustCompile(`[^a-z0-9-]+`)

func dnsLabel(value string) string {
	value = strings.Trim(invalidDNS.ReplaceAllString(strings.ToLower(value), "-"), "-")
	if value == "" {
		value = "run"
	}
	if len(value) > 20 {
		value = value[:20]
	}
	return strings.Trim(value, "-")
}
