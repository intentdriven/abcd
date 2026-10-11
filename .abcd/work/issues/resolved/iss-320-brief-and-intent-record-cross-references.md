---
schema_version: 1
id: "iss-320"
slug: "brief-and-intent-record-cross-references"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "bughunt-round-1"
found_at: ".abcd/development/brief"
resolution: "eighteen dangling brief anchors repaired; each corrected target verified to resolve."
impact: internal
---

brief and intent record cross-references carry dangling section anchors including five that misdirect a reader to the wrong section
## Evidence
18 link sites across 13 record files point at anchors that do not resolve under GitHub's slug algorithm (calibrated against known-working `--` em-dash anchors in research/related-work.md). The five that actively misdirect (correct-looking visible label, wrong or non-existent target):
- `05-internals/06-lint.md:49` labels "§ 5" but acceptance-gates is §6 (§5 is Frontmatter fields).
- `00-meta.md:11` → `02-disembark.md#3-agent-context-budget` — no §3, no "context budget" heading ever existed (rule lives in `05-internals/03-configuration.md`).
- `01-product/01-press-release.md:32` → `01-ahoy.md#1-acceptance` — heading is unnumbered `## Acceptance` (slug `acceptance`); inside an acceptance criterion.
- `itd-1-acceptance-gates.md:75` → stale `intent-fidelity-reviewer` slug (agent renamed to `intent-auditor`).
- `05-internals/05-prompt-quality.md:13` → stale `#2-mcp-preferred-structural-fallback` (heading is `#2-host-delegated-by-default-oracle-adapters-opt-in`).
Plus 13 cosmetic slug misses (`#2-payload-manifest-default-deny`×3 → `#2-curated-release-artefact-default-deny`; the `voyage-layout` em-dash single-vs-double hyphen ×8; `#the-naurian-gap` ×2).

## Adversarial verdict: CONFIRMED, all 18 (minor)
`links_resolve` strips the fragment before os.Stat, so anchors are structurally invisible to the linter — a fix cannot regress-guard itself. A 2026-07-06 review receipt catalogued 4 of these and the remediation only half-drained them; no ledger issue tracks the remainder. .abcd/** never ships in binaries, so this is record integrity, which the repo treats as load-bearing. Fix: correct the hrefs (and the three visible labels §5→§6 etc.). Not prior art: iss-14 (resolved) fixed a section NUMBER on itd-1 before the agent rename re-staled its slug.
