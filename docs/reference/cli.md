# cloudivision CLI

See [CLI installation](../operations/install-cli.md) for release binaries, the verified install script, Windows instructions, and shell completion setup.

Build the CLI with `make build` or `go build -o bin/cloudivision ./cmd/cloudivision`. The API URL is resolved from `--api-url`, `CLOU_DIVISION_API_URL`, `~/.cloudivision/config.yaml`, then `http://localhost:8080`. Token and namespace use the same flag → environment → config precedence; the default namespace is `default`.

`cloudivision version` reports the semantic version, source commit, and UTC build date. Use `--output json` for machine-readable metadata.

```sh
cloudivision --api-url http://localhost:8080 --token "$TOKEN" login
cloudivision --namespace ci project list
cloudivision repo add --name app --project storefront --url https://github.com/acme/app.git --pipeline-template node
cloudivision pipeline list
```

Trigger and watch a build:

```sh
cloudivision -n ci build trigger \
  --project storefront --repository app --pipeline-template node \
  --revision main --branch main --param target=production --watch

cloudivision -n ci build list --phase Failed --limit 50
cloudivision -n ci build get BUILD_NAME --output json
cloudivision -n ci build logs BUILD_NAME --tail 200 --follow
cloudivision -n ci build watch BUILD_NAME --timeout 20m
```

`build watch` exits 0 for `Succeeded` and non-zero for `Failed`, `Cancelled`, API failure, or timeout. `build logs --follow` polls the bounded log endpoint; it is not an unbounded streaming transport.

Release operations:

```sh
cloudivision -n ci release list
cloudivision -n ci release get RELEASE_NAME --output json
cloudivision -n ci release approve RELEASE_NAME --actor alice --comment "change approved"
cloudivision -n ci release reject RELEASE_NAME --actor alice --comment "rollback required"
cloudivision -n ci release rollback FAILED_RELEASE \
  --target-release PREVIOUS_DEPLOYED_RELEASE \
  --actor alice --reason "health regression"
```

Rollback creates a new auditable Release that restores the selected previously
deployed image. It remains subject to environment approval, digest, and signature
policy; the CLI never applies application manifests directly.

Run `cloudivision doctor` to check API health, authentication, provider/GitOps health and, when `kubectl` is available, CRDs, deployments, runner image configuration, and Job RBAC. `WARN` means an optional Kubernetes check could not run; `FAIL` makes doctor exit non-zero.

Use `--output json` for automation. Never put tokens in shell history; prefer `CLOU_DIVISION_TOKEN` from a secure process environment or a mode-0600 config file. `login` never prints the token.
