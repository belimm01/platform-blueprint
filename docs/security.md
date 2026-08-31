# Security model

## Trust boundaries

- Service claims and pull requests are untrusted input.
- Git review is the authorization boundary for desired-state changes.
- Argo CD and Crossplane hold cluster/cloud privileges; application CI does not.
- Admission policy is the final cluster-side guardrail.

## Controls

- Contract validation rejects invalid names, environments, ports, and untagged images.
- Generated namespaces include quotas and default-deny ingress.
- Kyverno requires probes and resource requests/limits.
- Production should replace mutable tags with image digests and verify signatures before admission.
- Secrets are referenced from a secret manager; this repository contains no secret values.

## Explicit non-goals

The reference implementation does not install controllers or provision a real AWS account. Operators must review controller RBAC, provider credentials, network egress, deletion policies, and tenant isolation before production use.
