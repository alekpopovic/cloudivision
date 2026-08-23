# Generic webhook notifications

Create a namespaced Kubernetes Secret containing the endpoint URL, then reference only its selected key:

```yaml
spec:
  notifications:
    enabled: true
    provider: webhook
    secretRef: {name: team-notifications, key: url}
    events: [BuildRunFailed, ReleaseAwaitingApproval, ReleaseFailed]
    filters: {environment: production}
```

The JSON payload contains event, project/resource identifiers, phase, message and occurrence time. It never contains the endpoint URL, token, Secret name or Secret data. Delivery retries transient server errors three times with bounded backoff; endpoint values are redacted from returned errors and logs.
