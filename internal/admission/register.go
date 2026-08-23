package admission

import (
	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/manager"
	cradmission "sigs.k8s.io/controller-runtime/pkg/webhook/admission"
)

func Register(mgr manager.Manager) {
	register := func(name string, obj runtime.Object) {
		mgr.GetWebhookServer().Register("/mutate-cicd-cloudivision-io-v1alpha1-"+name, cradmission.WithCustomDefaulter(mgr.GetScheme(), obj, Defaulter{}))
		mgr.GetWebhookServer().Register("/validate-cicd-cloudivision-io-v1alpha1-"+name, cradmission.WithCustomValidator(mgr.GetScheme(), obj, Validator{Reader: mgr.GetClient()}))
	}
	register("project", &cicdv1alpha1.Project{})
	register("repository", &cicdv1alpha1.Repository{})
	register("pipelinetemplate", &cicdv1alpha1.PipelineTemplate{})
	register("buildrun", &cicdv1alpha1.BuildRun{})
	register("environment", &cicdv1alpha1.Environment{})
	register("release", &cicdv1alpha1.Release{})
}
