package conversion

import (
	"reflect"
	"testing"

	alpha "github.com/cloudivision/cloudivision/api/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestV1Alpha1V1Beta1RoundTrips(t *testing.T) {
	now := metav1.Now()
	tests := []struct {
		name string
		run  func() (any, any, error)
	}{
		{"Project", func() (any, any, error) {
			in := &alpha.Project{TypeMeta: metav1.TypeMeta{APIVersion: alpha.GroupVersion.String(), Kind: "Project"}, ObjectMeta: metav1.ObjectMeta{Name: "p", Labels: map[string]string{"x": "y"}}, Spec: alpha.ProjectSpec{DisplayName: "P", OwnerTeam: "team", Namespace: "work", DefaultRegistry: "registry", DefaultBranch: "trunk", Registry: &alpha.ProjectRegistrySpec{ImagePrefix: "org"}}}
			mid, e := ProjectToV1Beta1(in)
			if e != nil {
				return in, nil, e
			}
			out, e := ProjectToV1Alpha1(mid)
			return in, out, e
		}},
		{"Repository", func() (any, any, error) {
			in := &alpha.Repository{TypeMeta: metav1.TypeMeta{APIVersion: alpha.GroupVersion.String(), Kind: "Repository"}, ObjectMeta: metav1.ObjectMeta{Name: "repo"}, Spec: alpha.RepositorySpec{ProjectRef: "p", Provider: alpha.RepositoryProviderGitHub, URL: "https://example.test/repo", DefaultBranch: "main", CredentialSecretRef: alpha.SecretKeyRef{Name: "git"}, Webhook: alpha.RepositoryWebhook{Enabled: true, SecretRef: alpha.RequiredSecretKeyRef{Name: "hook", Key: "token"}, Events: []string{"push"}}, PipelineTemplateRef: "pipe"}}
			mid, e := RepositoryToV1Beta1(in)
			if e != nil {
				return in, nil, e
			}
			out, e := RepositoryToV1Alpha1(mid)
			return in, out, e
		}},
		{"PipelineTemplate", func() (any, any, error) {
			in := &alpha.PipelineTemplate{TypeMeta: metav1.TypeMeta{APIVersion: alpha.GroupVersion.String(), Kind: "PipelineTemplate"}, ObjectMeta: metav1.ObjectMeta{Name: "pipe"}, Spec: alpha.PipelineTemplateSpec{ProjectRef: "p", Params: []alpha.ParamSpec{{Name: "mode", Default: "safe"}}, Steps: []alpha.PipelineStep{{Name: "test", Image: "go"}}, Build: alpha.PipelineBuildSpec{Enabled: true, Builder: alpha.BuildBuilderBuildKit}, Security: alpha.PipelineSecuritySpec{RunAsNonRoot: true}}}
			mid, e := PipelineTemplateToV1Beta1(in)
			if e != nil {
				return in, nil, e
			}
			out, e := PipelineTemplateToV1Alpha1(mid)
			return in, out, e
		}},
		{"BuildRun", func() (any, any, error) {
			in := &alpha.BuildRun{TypeMeta: metav1.TypeMeta{APIVersion: alpha.GroupVersion.String(), Kind: "BuildRun"}, ObjectMeta: metav1.ObjectMeta{Name: "build"}, Spec: alpha.BuildRunSpec{ProjectRef: "p", RepositoryRef: "r", PipelineTemplateRef: "pipe", Revision: "main", Branch: "main", CommitSHA: "abc", TriggeredBy: alpha.TriggeredBy{Type: alpha.TriggerTypeManual, Actor: "me"}, Image: alpha.ImageRef{Repository: "image", Tag: "one"}, Params: map[string]string{"mode": "safe"}, Executor: alpha.ExecutorTypeJob}}
			mid, e := BuildRunToV1Beta1(in)
			if e != nil {
				return in, nil, e
			}
			out, e := BuildRunToV1Alpha1(mid)
			return in, out, e
		}},
		{"Environment", func() (any, any, error) {
			in := &alpha.Environment{TypeMeta: metav1.TypeMeta{APIVersion: alpha.GroupVersion.String(), Kind: "Environment"}, ObjectMeta: metav1.ObjectMeta{Name: "prod"}, Spec: alpha.EnvironmentSpec{ProjectRef: "p", DisplayName: "Production", Namespace: "prod", Type: alpha.EnvironmentTypeProduction, RequiresApproval: true, GitOps: alpha.EnvironmentGitOpsSpec{Provider: alpha.GitOpsProviderArgoCD}}}
			mid, e := EnvironmentToV1Beta1(in)
			if e != nil {
				return in, nil, e
			}
			out, e := EnvironmentToV1Alpha1(mid)
			return in, out, e
		}},
		{"Release", func() (any, any, error) {
			in := &alpha.Release{TypeMeta: metav1.TypeMeta{APIVersion: alpha.GroupVersion.String(), Kind: "Release"}, ObjectMeta: metav1.ObjectMeta{Name: "rel"}, Spec: alpha.ReleaseSpec{ProjectRef: "p", EnvironmentRef: "prod", BuildRunRef: "build", Image: alpha.ImageRef{Repository: "image", Digest: "sha256:abc"}, Approval: alpha.ReleaseApprovalSpec{Required: true, ApprovedBy: "me", ApprovedAt: &now, Comment: "ok"}, Strategy: alpha.ReleaseStrategyGitOps, PromotionMode: alpha.PromotionModeDirectCommit}}
			mid, e := ReleaseToV1Beta1(in)
			if e != nil {
				return in, nil, e
			}
			out, e := ReleaseToV1Alpha1(mid)
			return in, out, e
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			want, got, err := test.run()
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(want, got) {
				t.Fatalf("round trip mismatch\nwant: %#v\n got: %#v", want, got)
			}
		})
	}
}
