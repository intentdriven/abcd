---
schema_version: 1
id: "iss-182"
slug: "build-plumbing-comments-describe-an"
severity: "minor"
category: "documentation"
source: "impl-review"
found_during: "2026-08-05 iss-37 merge-gate review"
found_at: "Makefile"
resolution: "Three build-plumbing comments corrected to state what actually runs. The Makefile preflight comment names the three lint-gate prerequisites (lint-reviews, record-lint, docs-lint) ahead of build/vet/test/race and attributes the `gofmt -l .` format gate to CI's check job rather than to preflight. The .githooks/pre-push comment describes what the hook enforces — `make preflight`, i.e. the three lint gates then the four Go steps — instead of a check-job trio, and its Actions-only list now carries the smoke lane and notes that CI's reviews-charter job repeats what lint-reviews already ran locally. The ci.yml header comment names the check job's gofmt, record-lint and docs-lint steps (ubuntu leg alone) and the separate reviews-charter and smoke jobs alongside the secret scan and workflow audit. Comment-only: no recipe, hook logic, workflow step, or gate behaviour changed."
impact: fix
---

Three build-plumbing comments still describe the pre-push gate and CI by an older, smaller shape. Same class as iss-37 (claims must match reality), in living files its re-scope does not cover, and no ledger entry records them until now. The fix direction is the same bar: state what actually runs, verified against the recipe or workflow beside the comment.

1. **`Makefile:58-61`**, the `preflight` recipe comment — "Pre-push gate (invoked by `.githooks/pre-push`): the same steps CI's check job runs — build, vet, test, and race-enabled internal tests — natively, plus the reviews-charter discipline." Two claims fail. CI's check job also runs a `gofmt -l .` format gate, which `preflight` does not, so the step sets are not the same; and the recipe's own prerequisite line reads `preflight: lint-reviews record-lint docs-lint`, of which the comment names only the reviews-charter one. It also now contradicts `CONTRIBUTING.md`, which describes the real suite.
2. **`.githooks/pre-push:12-14`** — "The CI check-job trio (build/vet/test + internal race) must pass natively before a push leaves this machine. Secret-scan and workflow-audit lanes stay in Actions." The hook invokes `make preflight`, which runs three lint gates ahead of those four Go steps, so the "trio" undercounts what the hook enforces; and the Actions-only list omits the reviews-charter job and the smoke job.
3. **`.github/workflows/ci.yml:3-4`**, the file header comment — "Lightweight checks on every push and pull request: build, vet, and test on macOS and Linux, plus full-history secret scanning and a workflow audit." It omits the check job's `gofmt`, record-lint and docs-lint steps (all three gated to the ubuntu leg) and the reviews-charter and smoke jobs.

Scope note: comment-only corrections in all three files. No recipe, hook logic, workflow step, or gate behaviour changes.
