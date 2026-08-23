package v1beta1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

var GroupVersion = schema.GroupVersion{Group: "cicd.cloudivision.io", Version: "v1beta1"}

var SchemeBuilder = runtime.NewSchemeBuilder(addKnownTypes)

var AddToScheme = SchemeBuilder.AddToScheme

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(GroupVersion,
		&Project{}, &ProjectList{},
		&Repository{}, &RepositoryList{},
		&PipelineTemplate{}, &PipelineTemplateList{},
		&BuildRun{}, &BuildRunList{},
		&Environment{}, &EnvironmentList{},
		&Release{}, &ReleaseList{},
	)
	return nil
}
