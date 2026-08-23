# PipelineTemplate catalog

The built-in, versioned catalog provides Go, npm, pnpm, Angular, React, Python, Maven, Gradle, Dockerfile, Helm and Kubernetes validation starting points. List definitions with `GET /api/v1/catalog/pipeline-templates`; install one with `POST /api/v1/catalog/pipeline-templates/{name}/install` and `{namespace, projectRef, name}`. The UI can install a definition directly or import it into the visual editor.

Catalog version `1.0.0` uses rootless BuildKit for image-producing templates and non-privileged validation steps. Review image versions, commands and project policy before installation.
