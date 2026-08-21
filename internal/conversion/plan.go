package conversion

// Version identifies a version participating in CRD conversion.
type Version string

const (
	// VersionV1Alpha1 is the only currently served and stored API version.
	VersionV1Alpha1 Version = "v1alpha1"
	// VersionV1Beta1 is reserved for the planned beta spoke/storage hub.
	VersionV1Beta1 Version = "v1beta1"
)

// Plan describes conversion topology without enabling a Kubernetes webhook.
// Served and Storage must only change together with generated CRDs, webhook
// deployment/TLS and upgrade tests.
type Plan struct {
	Hub     Version
	Served  []Version
	Storage Version
}

// CurrentPlan returns the single-version topology used by all current installs.
func CurrentPlan() Plan {
	return Plan{
		Hub:     VersionV1Alpha1,
		Served:  []Version{VersionV1Alpha1},
		Storage: VersionV1Alpha1,
	}
}

// Supports reports whether the current topology serves a version.
func (p Plan) Supports(version Version) bool {
	for _, served := range p.Served {
		if served == version {
			return true
		}
	}
	return false
}
