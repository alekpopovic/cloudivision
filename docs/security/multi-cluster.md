# Multi-cluster security

Multi-cluster support is experimental and health-check-only. A kubeconfig grants
whatever authority its embedded credential carries, so use a dedicated,
short-lived identity with permission only for Kubernetes discovery until more
features are implemented. Store each kubeconfig in a Secret in the same
namespace as its `ClusterTarget`; never place it directly in the CR.

The controller reads only the named Secret and key. API responses expose the
reference, reachability and version, never Secret data. Restrict who may create
ClusterTargets or Secrets because choosing an arbitrary API endpoint can cause
server-side request forgery and disclose client identity metadata. Apply
egress allow-lists, private endpoints, certificate verification, audit Secret
access, rotate credentials, and delete unused targets promptly.

No build, repository credential, global secret, or deployment token is sent to
an external target in this release. Do not broaden the health-check identity in
anticipation of future scheduling.
