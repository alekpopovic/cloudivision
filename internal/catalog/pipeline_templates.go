package catalog

import (
	"sort"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
)

const Version = "1.0.0"

type PipelineTemplate struct {
	Name        string                            `json:"name"`
	Version     string                            `json:"version"`
	Description string                            `json:"description"`
	Parameters  []cicdv1alpha1.ParamSpec          `json:"parameters,omitempty"`
	Spec        cicdv1alpha1.PipelineTemplateSpec `json:"spec"`
}

func PipelineTemplates() []PipelineTemplate {
	definitions := []PipelineTemplate{
		stack("go", "Go test and rootless image build", "golang:1.26", []string{"go", "test", "./..."}),
		stack("node-npm", "Node.js npm CI", "node:24-alpine", []string{"npm", "ci", "&&", "npm", "test"}),
		stack("node-pnpm", "Node.js pnpm CI", "node:24-alpine", []string{"sh", "-c", "corepack enable && pnpm install --frozen-lockfile && pnpm test"}),
		stack("angular", "Angular test and production build", "node:24-alpine", []string{"sh", "-c", "npm ci && npm test -- --watch=false && npm run build"}),
		stack("react", "React test and production build", "node:24-alpine", []string{"sh", "-c", "npm ci && npm test -- --watch=false && npm run build"}),
		stack("python", "Python tests", "python:3.14-slim", []string{"sh", "-c", "pip install -r requirements.txt && pytest"}),
		stack("java-maven", "Java Maven verify", "maven:3.9-eclipse-temurin-21", []string{"mvn", "-B", "verify"}),
		stack("java-gradle", "Java Gradle check", "gradle:9-jdk21", []string{"gradle", "check"}),
		buildOnly("dockerfile", "Build and push an existing Dockerfile"),
		validateOnly("helm-chart", "Validate a Helm chart", "alpine/helm:3.18.4", []string{"helm", "lint", "."}),
		validateOnly("kubernetes-manifests", "Validate Kubernetes YAML", "ghcr.io/yannh/kubeconform:v0.7.0", []string{"-summary", "-strict", "manifests"}),
	}
	sort.Slice(definitions, func(i, j int) bool { return definitions[i].Name < definitions[j].Name })
	return definitions
}

func FindPipelineTemplate(name string) (PipelineTemplate, bool) {
	for _, item := range PipelineTemplates() {
		if item.Name == name {
			return item, true
		}
	}
	return PipelineTemplate{}, false
}

func stack(name, description, image string, command []string) PipelineTemplate {
	return PipelineTemplate{Name: name, Version: Version, Description: description, Parameters: commonParameters(), Spec: cicdv1alpha1.PipelineTemplateSpec{Description: description, Steps: []cicdv1alpha1.PipelineStep{{Name: "test", Image: image, Command: command}}, Build: cicdv1alpha1.PipelineBuildSpec{Enabled: true, Builder: cicdv1alpha1.BuildBuilderBuildKit, ContextDir: ".", Dockerfile: "Dockerfile", Push: true}, Security: safeSecurity()}}
}

func buildOnly(name, description string) PipelineTemplate {
	return PipelineTemplate{Name: name, Version: Version, Description: description, Parameters: commonParameters(), Spec: cicdv1alpha1.PipelineTemplateSpec{Description: description, Build: cicdv1alpha1.PipelineBuildSpec{Enabled: true, Builder: cicdv1alpha1.BuildBuilderBuildKit, ContextDir: ".", Dockerfile: "Dockerfile", Push: true}, Security: safeSecurity()}}
}

func validateOnly(name, description, image string, command []string) PipelineTemplate {
	return PipelineTemplate{Name: name, Version: Version, Description: description, Spec: cicdv1alpha1.PipelineTemplateSpec{Description: description, Steps: []cicdv1alpha1.PipelineStep{{Name: "validate", Image: image, Command: command}}, Build: cicdv1alpha1.PipelineBuildSpec{Builder: cicdv1alpha1.BuildBuilderNone}, Security: safeSecurity()}}
}

func commonParameters() []cicdv1alpha1.ParamSpec {
	return []cicdv1alpha1.ParamSpec{{Name: "image", Description: "Target OCI image repository", Required: true}}
}
func safeSecurity() cicdv1alpha1.PipelineSecuritySpec {
	return cicdv1alpha1.PipelineSecuritySpec{RunAsNonRoot: true}
}
