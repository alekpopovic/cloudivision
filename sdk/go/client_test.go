package client

import "testing"

func TestGeneratedEndpointsContainCoreActions(t *testing.T) {
	for _, name := range []string{"GetApiV1AuthMe", "PostApiV1BuildRunsNamespaceNameRetry", "PostApiV1ReleasesNamespaceNameRollback", "GetApiV1AuditEventsExport"} {
		if _, ok := Endpoints[name]; !ok {
			t.Fatalf("missing generated operation %s", name)
		}
	}
}
