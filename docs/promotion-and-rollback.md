# Rehearse promotion and rollback locally

This exercise renders a staging candidate, promotes its image reference to a production claim, and restores the previous production claim. It requires Go 1.23+, Python 3, and a POSIX shell. Run the commands from the repository root. No cluster, registry, Git push, or cloud credentials are used.

The repository and image references are examples, not downloadable release artifacts. Successful rendering proves configuration behavior, not deployment health or image availability.

## Prepare isolated claims

Keep the same shell open throughout the exercise. All output goes into a temporary directory, leaving the checked-in example unchanged.

```bash
set -eu
work_dir="$(mktemp -d)"
export work_dir
trap 'rm -rf "$work_dir"' EXIT
python3 - <<'PY'
import json
import os
from pathlib import Path

work = Path(os.environ["work_dir"])
baseline = json.loads(Path("examples/catalog-service.json").read_text())
staging = dict(baseline, environment="staging", image="ghcr.io/example/catalog-service:1.2.4")
promoted = dict(baseline, image=staging["image"])
for name, claim in (("production-before", baseline), ("staging", staging), ("production-promoted", promoted)):
    (work / (name + ".json")).write_text(json.dumps(claim, indent=2) + "\n")
PY
for name in production-before staging production-promoted; do
  go run ./cmd/platformctl validate -file "$work_dir/$name.json"
  go run ./cmd/platformctl render -file "$work_dir/$name.json" -out "$work_dir/$name.yaml"
done
```

The staging namespace is `commerce-staging`; production remains `commerce-production`. Argo CD application names also differ by environment. Quotas and default-deny ingress policies are namespace-scoped. A namespace is shared by claims with the same owner and environment, so these guardrails are not per-service isolation.

## Review the promotion

```bash
python3 - <<'PY'
import difflib
import json
import os
from pathlib import Path

work = Path(os.environ["work_dir"])
before = json.loads((work / "production-before.json").read_text())
after = json.loads((work / "production-promoted.json").read_text())
staging = json.loads((work / "staging.json").read_text())
assert after["image"] == staging["image"]
assert {key for key in before if before[key] != after[key]} == {"image"}
old = (work / "production-before.yaml").read_text().splitlines(keepends=True)
new = (work / "production-promoted.yaml").read_text().splitlines(keepends=True)
print("".join(difflib.unified_diff(old, new, fromfile="production-before", tofile="production-promoted")), end="")
PY
```

The rendered production diff changes only the `image.tag` Helm parameter from `1.2.3` to `1.2.4`. Production namespace, application identity, replicas, and resource requests remain unchanged. In a real promotion, review and commit both the production claim and its rendered output to the operator-owned GitOps repository. Do not promote by changing staging's environment in place and removing its desired state: automated pruning could delete staging resources.

## Restore the previous desired state

```bash
cp "$work_dir/production-before.json" "$work_dir/production-rollback.json"
go run ./cmd/platformctl validate -file "$work_dir/production-rollback.json"
go run ./cmd/platformctl render -file "$work_dir/production-rollback.json" -out "$work_dir/production-rollback.yaml"
cmp "$work_dir/production-before.yaml" "$work_dir/production-rollback.yaml"
printf '%s\n' 'Rollback manifest matches the previous production manifest.'
```

A successful `cmp` proves byte-identical desired-state restoration with this renderer. In a real rollback, restore the previous production claim and matching rendered output through a reviewed Git change. Re-render using the same renderer revision as the previous release; renderer upgrades can change output independently of the claim. Avoid broad reverts that undo unrelated services or environment changes.

## Boundaries before a real rollout

- The rendered Argo CD source uses `targetRevision: main` and `path: deploy/chart`. Restoring a claim does not restore chart code. Record and restore a reviewed chart revision separately before treating a rollback as reproducible.
- Image tags can move. The CLI splits image references into `image.repository` and `image.tag`; it does not verify registry immutability or provide a dedicated digest parameter. Require registry-enforced immutable tags for this contract. Do not assume a digest reference is handled correctly by an arbitrary downstream chart.
- The example does not include the application chart. A real chart must consume the emitted Helm parameters and supply limits, probes, and appropriate ingress allowances; a rendered Application alone does not prove those requirements.
- Argo CD uses automated sync, pruning, and self-healing. Manual cluster edits can be reverted. Review namespace/application identity changes and deletions before committing desired state.
- Image rollback does not reverse database migrations, external side effects, or Crossplane-managed resources. Check backward compatibility and restoration procedures separately.
- After a real sync, verify the exact desired Git/chart revision, resolved image, Argo CD health, pod readiness, and application behavior. This local exercise performs none of those live checks.
