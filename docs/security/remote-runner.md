# Remote runner security

A remote runner crosses the strongest platform trust boundary: untrusted
repository code executes outside the control-plane namespace and may be on an
operator-managed network. The mode must remain disabled until the following
controls are implemented and tested:

- one-time bootstrap followed by short-lived, audience-bound agent credentials;
- rotation, immediate revocation, replay protection and clock-skew handling;
- per-project runner authorization and deny-by-default pool matching;
- assignment leases, concurrency reservations and signed state transitions;
- minimal per-assignment secret envelopes, never global or unrelated secrets;
- outbound-only polling where possible, TLS verification and egress allow-lists;
- full registration, assignment, cancellation and completion audit history;
- redaction of tokens, repository credentials, variables and command output;
- sandboxing equivalent to the current non-root, non-privileged Job baseline.

Runner labels are authorization-adjacent input and cannot be trusted merely
because an agent asserted them. Pool membership and protected labels must be
issued by the control plane. Compromised runners must be quarantinable without
invalidating unrelated agents or projects.
