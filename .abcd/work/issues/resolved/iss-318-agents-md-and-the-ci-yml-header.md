---
schema_version: 1
id: "iss-318"
slug: "agents-md-and-the-ci-yml-header"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: "AGENTS.md"
resolution: "AGENTS.md and the ci.yml header add the dependency-review and govulncheck jobs."
impact: internal
---

AGENTS.md and the ci.yml header enumerate CI separate jobs as a closed list that omits dependency-review and govulncheck
## Evidence
`AGENTS.md:78-80` (also CLAUDE.md, a symlink): "Separate jobs run the reviews-charter check, gitleaks, zizmor, and the smoke harness." `.github/workflows/ci.yml` declares eight jobs including `dependency-review` (`:364`) and `govulncheck` (`:378`), which `CONTRIBUTING.md:22-25` names correctly. `ci.yml:8-9`'s own header comment carries the same stale closed list (a recurrence of iss-182 item 3). Both jobs landed today in 61a15b4, which touched ci.yml only.

## Adversarial verdict: CONFIRMED (minor)
Not abridgement — the parallel CONTRIBUTING.md sentence is complete; the docs follow-up 3b17b6a updated CONTRIBUTING/SECURITY/pre-push but not AGENTS.md. dependency-review is deliberately non-required, so the merge queue is not misdescribed, but AGENTS.md tells an agent "new deps need sign-off" without noting CI now scans them. Fix: add "dependency review, govulncheck" to both AGENTS.md:78-80 and ci.yml:8-9. Not prior art: iss-182 (resolved) scoped three build-plumbing comments, AGENTS.md not among them.
