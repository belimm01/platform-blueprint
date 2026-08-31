# Architecture

Platform Blueprint models a paved road as a deliberately small contract rather than a collection of copied YAML.

```text
service claim (JSON)
        |
        v
 platformctl -- validate contract, apply defaults, render deterministic output
        |
        +--> namespace boundary + quota + default network policy
        |
        +--> Argo CD Application --> team-owned Helm chart --> Kubernetes
        |
        +--> Crossplane ServiceEnvironment claim --> cloud/runtime resources
```

## Control boundaries

Application teams own source code, their deployable image, and workload-level configuration. The platform team owns the service-claim contract, environment policy, GitOps reconciliation, and reusable infrastructure compositions. Cloud credentials stay in controllers; they are never handed to application pipelines.

## Reconciliation and failure model

Rendered configuration is committed and reviewed before Argo CD reconciles it. Reconciliation is idempotent: the same claim produces byte-stable output. Invalid claims fail before any cluster mutation. Argo CD reports drift and retries convergence; it does not conceal unhealthy workloads, which remain visible through Kubernetes health and SLO telemetry.

## Production evolution

The local renderer is intentionally dependency-light. A production implementation can expose the same contract through a Kubernetes API and reconcile it with a Go controller. Keeping the external contract separate from the implementation allows that migration without forcing application teams to rewrite service descriptors.
