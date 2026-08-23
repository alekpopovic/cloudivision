package api

import (
	"net/http"

	cicdv1alpha1 "github.com/cloudivision/cloudivision/api/v1alpha1"
	"github.com/cloudivision/cloudivision/internal/catalog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (s Server) catalogPipelineTemplates(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, catalog.PipelineTemplates())
}

func (s Server) installCatalogPipelineTemplate(w http.ResponseWriter, r *http.Request) {
	definition, ok := catalog.FindPipelineTemplate(r.PathValue("name"))
	if !ok {
		s.writeError(w, apiError{status: http.StatusNotFound, code: "catalog_template_not_found", message: "catalog pipeline template was not found"})
		return
	}
	var request CatalogInstallRequest
	if !s.decode(w, r, &request) {
		return
	}
	name := request.Name
	if name == "" {
		name = definition.Name
	}
	if err := validateName(name); err != nil {
		s.writeError(w, badRequest(err.Error()))
		return
	}
	spec := definition.Spec.DeepCopy()
	spec.ProjectRef = request.ProjectRef
	object := &cicdv1alpha1.PipelineTemplate{ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: s.namespace(request.Namespace), Labels: map[string]string{"cicd.cloudivision.io/catalog-template": definition.Name}, Annotations: map[string]string{"cicd.cloudivision.io/catalog-version": definition.Version}}, Spec: *spec}
	if err := s.Client.Create(r.Context(), object); err != nil {
		s.writeError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, pipelineTemplateDTO(*object))
}
