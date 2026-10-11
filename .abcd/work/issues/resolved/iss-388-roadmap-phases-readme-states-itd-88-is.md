---
schema_version: 1
id: "iss-388"
slug: "roadmap-phases-readme-states-itd-88-is"
severity: "minor"
category: "documentation"
source: "agent-finding"
found_during: "bughunt-round-3"
found_at: ".abcd/development/roadmap/phases/README.md"
resolution: "drop the stale planned/ bucket claim for the shipped itd-88, keeping the adr-34 orthogonality point"
impact: fix
---

roadmap phases README states itd-88 is in planned/ and unscheduled, but the intent has shipped — a false bucket claim in present-tense current-state prose
## Evidence

- `.abcd/development/roadmap/phases/README.md:109` — "itd-88 is in `planned/` and enters no phase's `## Scope` — scheduled and planned are orthogonal axes (per adr-34), and an intent committed-to but unsequenced is a valid state." The sentence sits under "A second cluster is now unscheduled on the same terms" — present-tense current-state prose, not historical narration.
- The intent lives at `.abcd/development/intents/shipped/itd-88-lifeboat-coverage-experiment.md`; adr-3 makes directory location the lifecycle source of truth.
- Refuter verdict: CONFIRMED (minor) — no lint covers prose bucket claims (`index_drift`'s regions do not include this), no prior art. The load-bearing half survives (itd-88 genuinely enters no phase Scope; the adr-34 orthogonality point stands) — only the bucket name and the committed-but-unbuilt implication are wrong, which is the misleading part for a triage pass.
