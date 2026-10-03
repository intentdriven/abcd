---
schema_version: 1
id: "iss-2610020726287561"
slug: "the-implement-loop-s-composed-commits-declare-assisted-by"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "autonomous run A resumed 2026-09-25"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/implement/loop/pickcommit.go"
remedy: "Per ruling PC1 (the technical facilitator, 2026-10-02), on the grounds that the loop's composed commits declare None, which AGENTS.md calls a false disclosure for tool-touched work: give them a third label naming abcd, Assisted-by: abcd:<version> (the binary's release version, or dev for a build with none), and amend AGENTS.md, .github/CONTRIBUTING.md and scripts/check-attribution.sh to accept it as one fixed form, refusing a malformed abcd label."
resolution: "The loop's pick commit and sync merge end with Assisted-by: abcd:<version> (release version, or dev), composed in one place; the attribution gate, the prepare-commit-msg hook, AGENTS.md and CONTRIBUTING.md name it as the third accepted form, and the gate refuses a malformed abcd label."
impact: additive
resolved_by:
  commit: "89ce4e39d"
---

The implement loop's composed commits declare Assisted-by: None, a false disclosure for tool-touched text: the pick's record-only commit (internal/core/implement/loop/pickcommit.go pickMessage) and the sync's merge commit (sync.go) carry text the abcd binary computes from the run's state and records, yet both end with Assisted-by: None, which AGENTS.md defines as the declaration that no tool touched the work and calls a false disclosure when a tool did. No model wrote that text either, so a vendor trailer would be false too: the convention has no truthful form for a commit abcd composes.

## Grounds

- pursued: we expect every commit the loop composes from record facts to carry exactly Assisted-by: abcd:<version> and the gate to accept it while refusing abcd:latest, abcd:0.12.0 and ABCD:v0.12.0; a pick or sync commit still ending in None, or a gate case passing a malformed abcd label, would show it wrong.
