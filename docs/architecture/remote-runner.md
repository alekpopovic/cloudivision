# Remote runner / agent architecture

Remote runner mode is an experimental design only. The current and only
enabled default remains the Kubernetes Job executor in the control-plane
cluster. `internal/executor/remoteagent` contains contracts and a provider
skeleton that always returns a disabled error.

The proposed model has `RunnerPool` inventory (`kubernetes-job` or
`remote-agent`, labels, concurrency limit and status), runner registration,
project-scoped authorization, short-lived assignments and pull-based polling.
The control plane matches a BuildRun's required labels to both a pool and a
registered agent, reserves capacity atomically, and returns only a leased
assignment. The agent reports bounded state transitions and the control plane
writes canonical BuildRun status and audit events.

Registration should exchange an operator-created one-time bootstrap credential
for a rotated agent identity. Poll responses must be idempotent, leases must
expire, and completion must carry the assignment identity and monotonic attempt
number. Offline runners release capacity only after lease expiry. RunnerPool is
not a served CRD yet because registration ownership, storage and revocation are
still open design decisions.

No controller registers the remote executor today, no BuildRun enum selects it,
and no workload or secret is sent off-cluster.
