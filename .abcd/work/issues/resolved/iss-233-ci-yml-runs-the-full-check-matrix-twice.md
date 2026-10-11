---
schema_version: 1
id: "iss-233"
slug: "ci-yml-runs-the-full-check-matrix-twice"
severity: "minor"
category: "observation"
source: "user-observation"
found_during: "manual-capture"
resolution: "Scoped ci.yml's push trigger to main; PRs run once via pull_request, concurrency expressions unchanged and still correct"
impact: internal
---

ci.yml runs the full check matrix twice per PR push — push and pull_request triggers overlap
## Evidence (2026-08-16, PR #239's branch)

Two pushes to `docs/adr-39-host-tier-policy` produced four ci runs: 2 ×
`push` + 2 × `pull_request` (verified via `gh run list`). Cause: `ci.yml`
triggers on both `push: branches: ['**']` and `pull_request`, and the
concurrency group keys on `github.ref`, so `refs/heads/<branch>` and
`refs/pull/N/merge` are distinct groups — neither run cancels the other.
Every PR push therefore pays the full matrix twice (each ~2.5 min wall;
the macOS leg alone is 2m31s).

## Fix sketch

Scope the push trigger to `main` (PRs stay covered by `pull_request`;
`main` pushes stay covered for the auto-release path). Trade-off to
confirm at fix time: branches pushed without an open PR would no longer
get CI — acceptable under the branch→PR-promptly convention, but it is a
behaviour change a maintainer must sign off (CI config gate).

`ci.yml` is NOT scaffold-coupled: `TestSelfScaffoldParity` pins only
`release.yml` + `auto-release.yml`, so this edit carries no template
mirror.
