package conversion

import "testing"

func TestCurrentPlanDoesNotAdvertiseUnimplementedBeta(t *testing.T) {
	plan := CurrentPlan()
	if plan.Hub != VersionV1Alpha1 || plan.Storage != VersionV1Alpha1 {
		t.Fatalf("CurrentPlan() = %#v, want v1alpha1 hub and storage", plan)
	}
	if !plan.Supports(VersionV1Alpha1) {
		t.Fatal("CurrentPlan() must serve v1alpha1")
	}
	if plan.Supports(VersionV1Beta1) {
		t.Fatal("CurrentPlan() must not advertise the unimplemented v1beta1 API")
	}
}
