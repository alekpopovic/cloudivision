# Slack notifications

Slack is represented by a provider skeleton so configuration, capability discovery and health reporting remain stable while delivery is implemented. Selecting `provider: slack` currently reports an unsupported delivery condition without interrupting BuildRun or Release reconciliation.

Keep incoming webhook URLs or tokens in a namespaced Kubernetes Secret. Do not place them directly in Project YAML. Microsoft Teams and email currently follow the same skeleton behavior.
