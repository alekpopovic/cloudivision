# CRD upgrade test skeleton

The upgrade gate will become executable when cloudivision has a published base release. `run.sh` currently fails explicitly rather than claiming an untested upgrade.

The eventual test must:

1. create a disposable kind cluster with explicit Kubernetes version;
2. install CRDs/controller from `UPGRADE_BASE_REF`;
3. create every v1alpha1 kind, including defaulted objects, terminal/non-terminal BuildRuns, approval states and status conditions;
4. save YAML and resource UIDs/resourceVersions for comparison;
5. install target CRDs and `UPGRADE_TARGET_IMAGE` without deleting resources;
6. wait for CRD Established and controller readiness with explicit timeouts;
7. assert every object remains readable and status/spec intent is preserved;
8. reconcile repeatedly and prove Jobs/Releases are not duplicated;
9. when multiple versions exist, read/write each served version and verify hub round trips;
10. inspect CRD `status.storedVersions`, run the documented storage migration and test rollback constraints;
11. print CRDs, objects, events, controller logs and child resources on failure.

The test must use released immutable artifacts for the base side. Building both sides from the current worktree would not test a real upgrade boundary.
