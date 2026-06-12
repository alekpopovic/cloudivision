package v1alpha1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"sigs.k8s.io/controller-runtime/pkg/scheme"
)

// GroupVersion is group version used to register these objects.
var GroupVersion = schema.GroupVersion{Group: "cicd.cloudivision.io", Version: "v1alpha1"}

// SchemeBuilder is used to add go types to the GroupVersionKind scheme.
var SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}

// AddToScheme adds the types in this group-version to the given scheme.
var AddToScheme = SchemeBuilder.AddToScheme

func init() {
	SchemeBuilder.Register(
		&Project{},
		&ProjectList{},
		&Repository{},
		&RepositoryList{},
		&PipelineTemplate{},
		&PipelineTemplateList{},
		&BuildRun{},
		&BuildRunList{},
		&Environment{},
		&EnvironmentList{},
		&Release{},
		&ReleaseList{},
	)
}

func addKnownTypes(scheme *runtime.Scheme) error {
	scheme.AddKnownTypes(
		GroupVersion,
		&Project{},
		&ProjectList{},
		&Repository{},
		&RepositoryList{},
		&PipelineTemplate{},
		&PipelineTemplateList{},
		&BuildRun{},
		&BuildRunList{},
		&Environment{},
		&EnvironmentList{},
		&Release{},
		&ReleaseList{},
	)
	return nil
}
