# External Secrets and Vault

The External Secrets provider is a capability skeleton. It detects `externalsecrets.external-secrets.io` and reports health, while cloudivision continues to consume only the resulting namespaced Kubernetes Secret through its selected-key resolver. It does not read external provider values directly.

The Vault provider currently accepts configuration shape (`address`, `authMethod`, `mount`) and reports whether configuration is present, but value resolution returns unsupported. Runtime environment variables are `CLOU_DIVISION_VAULT_ADDRESS`, `CLOU_DIVISION_VAULT_AUTH_METHOD`, and `CLOU_DIVISION_VAULT_MOUNT`.

For External Secrets, scope SecretStore/ClusterSecretStore permissions independently, write only required keys into the target Secret, keep target namespaces project-local, and verify provider health through `/api/v1/providers/health`.
