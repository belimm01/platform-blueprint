# ADR 0001: Keep deployment mutation behind a GitOps boundary

- Status: accepted
- Date: 2026-08-31

## Context

A self-service platform needs fast feedback without giving each application pipeline broad cluster credentials. Direct `kubectl apply` is simple but makes review, drift detection, and rollback inconsistent.

## Decision

`platformctl` renders deterministic manifests. Teams submit the generated change to Git, and Argo CD is the only component that mutates workload resources in shared clusters. Crossplane controllers separately reconcile infrastructure claims.

## Consequences

- Every desired-state change is reviewable and attributable.
- Application CI needs no cluster or cloud credential.
- Rollback is a Git revert and remains observable through reconciliation status.
- Delivery now depends on Git and Argo CD availability; emergency procedures must be explicit rather than silently bypassing the boundary.
