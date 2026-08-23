# Plugin system v1

cloudivision plugins are trusted Go implementations registered in process when the API server starts. The v1 registry exposes one consistent inventory for built-in provider adapters and platform extensions without changing the existing provider interfaces.

Supported plugin types are `git`, `registry`, `build`, `gitops`, `notification`, `policy`, and `supply-chain`. Every plugin publishes a name, type, version, capabilities, JSON configuration schema, configuration status, optional documentation URL, and a health check.

Use `GET /api/v1/plugins` for metadata, `GET /api/v1/plugins/{type}/{name}` for one plugin, and `GET /api/v1/plugins/health` for live health. The Plugins page presents the same information.

## Runtime boundary

Registration is static and compile-time only. Plugins execute with the API server's privileges and therefore must be reviewed as trusted product code. The registry does not load shared libraries, execute downloaded binaries, discover arbitrary containers, or accept remote registration.

External and dynamically installed plugins remain future work. A future protocol must add process isolation, authentication, version negotiation, timeouts, resource controls, and a failure boundary before third-party code can run safely.

## Provider compatibility

The existing provider registry remains the source of concrete Git, registry, build, GitOps, notification, secrets and supply-chain adapters. Supported providers are wrapped by a small plugin adapter during startup, so controller and domain code can continue using the established narrow provider interfaces.
