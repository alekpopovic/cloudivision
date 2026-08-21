# Prompt 33 — Policy Engine Layer

Phase: Policy

```text
Continue working on the cloudivision project.

Task:
Add a policy engine layer for cloudivision.

Goal:
Centralize business, security and release policies instead of scattering checks across controllers and handlers.

Create:
- /internal/policy

Core types:

type Decision struct {
    Allowed bool
    Reason string
    Message string
    EvaluatedAt time.Time
    Violations []Violation
}

type Violation struct {
    Policy string
    Severity string
    Message string
    FieldPath string
}

type Evaluator interface {
    EvaluateBuildRun(ctx context.Context, input BuildRunPolicyInput) Decision
    EvaluateRelease(ctx context.Context, input ReleasePolicyInput) Decision
    EvaluatePipelineTemplate(ctx context.Context, input PipelineTemplatePolicyInput) Decision
}

Policy checks:
- CanTriggerBuild
- CanUseSecret
- CanDeployToEnvironment
- CanPromoteRelease
- ImageMustHaveDigest
- ImageMustBeSigned
- SBOMRequired
- CriticalVulnerabilitiesBlocked
- ProductionRequiresApproval
- RunnerPrivilegedForbidden
- DockerSocketForbidden
- HostPathForbidden
- UnknownProviderForbidden

Integration points:
1. API server evaluates policy before creating BuildRun and returns structured denial errors.
2. BuildRun controller evaluates runtime policy before creating Job and sets PolicyDenied condition if blocked.
3. Release controller evaluates release policy before GitOps update and sets PolicyDenied if blocked.
4. Angular UI shows policy decision and violations.

Configuration:
- Start with code-based default policy.
- Add ConfigMap-based policy later or now if feasible.
- Do not introduce OPA dependency in this task unless already planned.

Tests:
- production release without approval denied
- production release without digest denied when required
- unsigned image denied when required
- critical vulnerabilities denied when configured
- privileged runner denied by default
- safe BuildRun allowed

Docs:
- docs/concepts/policy.md
- docs/security/policy-examples.md

Acceptance criteria:
- go test ./... passes.
- npm run build passes in /web.
- Policy decisions are centralized.
- Controllers do not duplicate policy logic unnecessarily.
- API returns structured policy denial errors.
- UI shows policy violations.
```
