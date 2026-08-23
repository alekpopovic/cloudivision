---
title: Kubernetes-native delivery, reconciled
description: Build immutable artifacts in isolated Kubernetes Jobs and promote them safely through GitOps.
home: true
---

<section class="hero">
  <div class="hero-copy">
    <span class="hero-badge">v0.2 alpha · Kubernetes native</span>
    <h1>Build in Kubernetes.<br><span>Deliver through GitOps.</span></h1>
    <p>cloudivision turns source events into observable BuildRuns, isolated runner Jobs, immutable artifacts, and auditable GitOps releases.</p>
    <div class="hero-actions">
      <a class="button button-primary" href="{{ '/getting-started/quickstart-kind.html' | relative_url }}">Start the kind quickstart</a>
      <a class="button" href="{{ '/development/architecture.html' | relative_url }}">Explore the architecture</a>
    </div>
  </div>
  <div class="hero-visual">
    <img src="{{ '/assets/brand/cloudivision-mark.png' | relative_url }}" alt="cloudivision modular C mark">
  </div>
</section>

## From source to release

Choose the path that matches what you need to do next.

<div class="feature-grid">
  <a class="feature-card" href="{{ '/getting-started/first-build.html' | relative_url }}">
    <img src="{{ '/assets/icons/build.svg' | relative_url }}" alt="">
    <h3>Create an artifact</h3>
    <p>Run your first BuildRun as a resource-bounded, non-root Kubernetes Job.</p>
    <strong>First build →</strong>
  </a>
  <a class="feature-card" href="{{ '/getting-started/first-release.html' | relative_url }}">
    <img src="{{ '/assets/icons/deploy.svg' | relative_url }}" alt="">
    <h3>Promote with GitOps</h3>
    <p>Update a delivery repository and let Argo CD or Flux own cluster convergence.</p>
    <strong>First release →</strong>
  </a>
  <a class="feature-card" href="{{ '/operations/install-helm.html' | relative_url }}">
    <img src="{{ '/assets/icons/platform.svg' | relative_url }}" alt="">
    <h3>Operate the platform</h3>
    <p>Install the control plane, configure providers, and review production hardening.</p>
    <strong>Install with Helm →</strong>
  </a>
</div>

## One control plane, explicit boundaries

The Kubernetes API stores desired and runtime state. Controllers reconcile that state idempotently; the runner creates artifacts, never application deployments.

<div class="architecture-panel">
  <img src="{{ '/assets/brand/architecture.svg' | relative_url }}" alt="A Git event creates a BuildRun, which starts an isolated runner Job and produces an OCI image. A Release updates a GitOps repository, then Argo CD or Flux deploys it.">
</div>

## Designed for trustworthy delivery

<div class="principles">
  <div class="principle"><strong>Kubernetes is the runtime truth</strong><span>CR status records build, artifact, policy, and release evidence.</span></div>
  <div class="principle"><strong>Repository code is untrusted</strong><span>No Docker socket, privileged default, or unbounded build workload.</span></div>
  <div class="principle"><strong>CI creates artifacts</strong><span>Builds produce immutable images and attestable supply-chain evidence.</span></div>
  <div class="principle"><strong>CD converges through Git</strong><span>Release automation commits or opens a pull request; GitOps applies.</span></div>
</div>

## Go deeper

- Learn the resource model: [Project](concepts/project.md), [Repository](concepts/repository.md), [PipelineTemplate](concepts/pipeline-template.md), [BuildRun](concepts/buildrun.md), [Environment](concepts/environment.md), and [Release](concepts/release.md).
- Operate confidently with [observability](operations/observability.md), [troubleshooting](operations/troubleshooting.md), and [security hardening](operations/security-hardening.md).
- Extend the platform through the [executor and provider interfaces](development/architecture.md) and [contributor guide](development/contributing.md).
