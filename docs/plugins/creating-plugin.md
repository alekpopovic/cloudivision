# Creating an in-process plugin

Implement `plugin.Plugin`, return immutable metadata from `Metadata`, and make `HealthCheck(context.Context)` bounded and safe to call repeatedly. Register the implementation in `configurePluginRegistry` during API startup.

```go
type Plugin interface {
    Metadata() Metadata
    HealthCheck(context.Context) provider.ProviderHealth
}
```

Use a stable lowercase name, one supported type, and a meaningful version. Capabilities should describe callable behavior rather than configuration fields. Supply a JSON Schema object even when the plugin accepts no settings. Never place secret values in metadata, configuration status, errors, or health messages.

Registration rejects missing metadata, unsupported types, and duplicate `type/name` keys. Add registry tests, API behavior tests, configuration documentation, and a failure-mode health test with each plugin.

This mechanism is for code compiled into cloudivision. Do not use it to load untrusted or user-supplied binaries.
