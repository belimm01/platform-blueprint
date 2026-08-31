# Operations runbook

## Claim rejected

Run `go run ./cmd/platformctl validate -file <claim>`. Correct every reported contract violation before generating manifests. Validation performs no external mutation.

## Argo CD reports OutOfSync

1. Compare the live and desired resource in Argo CD.
2. Confirm the difference is not produced by a mutating admission controller.
3. Re-render the claim and verify the committed output is current.
4. Sync only after reviewing destructive operations; do not disable pruning globally.

## Workload is deployed but unavailable

Inspect readiness failures, resource saturation, and application telemetry. GitOps convergence is not a health guarantee. Roll back the image tag through Git when the failure followed a release.

## Crossplane claim remains unready

Inspect claim conditions and composed-resource events. Verify provider configuration and quota without exposing credential material. Do not delete a production claim until its deletion policy and external-resource ownership are understood.
