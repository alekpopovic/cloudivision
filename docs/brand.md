# cloudivision brand system

The cloudivision identity should feel operational, calm, and precise. It uses a
modular **C** mark to connect the product's core ideas: Kubernetes resources,
pipeline stages, and forward delivery through GitOps.

## Primary mark

<p align="center">
  <img src="../web/public/assets/brand/cloudivision-mark.png" alt="cloudivision modular C logo" width="180">
</p>

The transparent PNG is the canonical generated mark for v0.1. Use it on dark,
white, or very light neutral backgrounds with clear space equal to roughly one
quarter of the mark's width. Do not rotate it, place it in a containing shape,
add effects, or recolor individual modules.

Asset: [`web/public/assets/brand/cloudivision-mark.png`](../web/public/assets/brand/cloudivision-mark.png)

## Wordmark

Use the lowercase product name `cloudivision` next to the mark. The web UI uses
the system sans-serif stack with a bold weight so it loads quickly and remains
consistent with the application. Keep the name as live text whenever possible;
this improves accessibility and avoids rasterized typography.

## Color tokens

| Token | Hex | Use |
| --- | --- | --- |
| Control plane | `#081426` | Primary dark surface, diagrams, navigation |
| Cluster blue | `#4F7CFF` | Primary action, build/source states |
| Delivery teal | `#2DD4BF` | Success, flow, GitOps/deployment accent |
| Cloud white | `#F8FAFC` | Main light surface and dark-surface text |
| Signal gray | `#9FB3CF` | Secondary text on dark surfaces |
| Border blue | `#31517C` | Dark-surface borders and separators |

Status colors retain their semantic meaning: emerald for healthy/succeeded,
amber for waiting/development, and rose for failed/destructive states. Do not use
brand blue or teal as a replacement for failure and warning signals.

## Icon system

Icons use a 64×64 rounded control-plane tile, a maximum of two brand accents, and
simple four-pixel strokes. They are functional navigation and documentation cues,
not illustrations.

| Platform | Pipeline | Source | Build | Deploy | Observe | Security |
| --- | --- | --- | --- | --- | --- | --- |
| ![Platform](../web/public/assets/brand/icons/platform.svg) | ![Pipeline](../web/public/assets/brand/icons/pipeline.svg) | ![Source](../web/public/assets/brand/icons/source.svg) | ![Build](../web/public/assets/brand/icons/build.svg) | ![Deploy](../web/public/assets/brand/icons/deploy.svg) | ![Observe](../web/public/assets/brand/icons/observe.svg) | ![Security](../web/public/assets/brand/icons/security.svg) |

Each SVG includes a title for standalone use. When an icon appears next to a
visible label in the application, treat it as decorative with an empty `alt`.

## Diagrams and charts

Use a left-to-right flow for build and release diagrams, rounded cards for
resources/providers, and a distinct control-plane band for reconciliation. Prefer
short labels and describe the full flow in nearby prose or SVG accessibility
metadata.

Canonical architecture chart:
[`docs/assets/brand/architecture.svg`](assets/brand/architecture.svg).

## Voice

- Lead with the operational outcome.
- Prefer concrete Kubernetes and GitOps terms over generic “cloud automation.”
- Be direct about alpha limitations and skipped integration evidence.
- Use **build** for CI artifact creation and **release** for GitOps promotion.
- Never imply that the runner directly deploys application manifests.

## Generation provenance

The primary raster mark was generated with the built-in OpenAI image generation
tool from an original logo brief. The repository-native icons and architecture
diagram were then authored as deterministic SVG assets using the same visual
language. This note documents asset provenance; it does not change the project's
software license or grant rights in third-party marks.
