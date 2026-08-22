# Upgrade lifecycle test

`make upgrade-test` always performs an offline chart, fixture, and CRD-retention check. Live assertions are opt-in because they install and uninstall cluster resources:

```sh
UPGRADE_TEST_LIVE=true \
UPGRADE_TEST_CONTEXT=kind-cloudivision \
UPGRADE_TEST_IMAGE_REGISTRY=ghcr.io/cloudivision \
UPGRADE_TEST_BASE_TAG=0.1.0 \
UPGRADE_TEST_TARGET_TAG=dev \
make upgrade-test
```

Use only a disposable cluster. The images must contain compatible controller, API, web, and runner binaries, and the cluster must reach the fixture repository and step image. The test installs the base chart, runs a BuildRun, applies target CRDs, upgrades workloads, compares resource UIDs, runs another BuildRun, uninstalls Helm, and verifies CRDs and custom resources remain.
