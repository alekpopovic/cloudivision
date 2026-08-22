package controller

const (
	BuildRunProjectIndex    = "spec.projectRef"
	BuildRunRepositoryIndex = "spec.repositoryRef"
	BuildRunEventIDIndex    = "spec.triggeredBy.eventID"
	ReleaseBuildRunIndex    = "spec.buildRunRef"
)

func normalizedConcurrency(value int) int {
	if value < 1 {
		return 1
	}
	return value
}
