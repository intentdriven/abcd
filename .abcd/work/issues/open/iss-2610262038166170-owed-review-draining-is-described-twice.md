---
schema_version: 1
id: "iss-2610262038166170"
slug: "owed-review-draining-is-described-twice"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 783 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:409"
remedy: "none (filed automatically)"
finding_ref: cfk-f85df95d7a683e50
---

Owed-review draining is described twice: staged run-seam gating that drains nothing today, and a shipped `intent audit --owed` that drains the owed set.

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:409` — "seam ships, so nothing drains today. When it does, owed intent audits drain at"
- End B: `.abcd/development/brief/05-internals/01-agents.md:154` — "drains the owed set). Either way the close stands"

03-configuration assigns draining owed intent audits to the run seam's receipt gating and says no run seam ships, so nothing drains today. 01-agents says `abcd intent audit --owed` drains the owed set. The two chapters claim the same job and contradict each other on whether anything does it now.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 783.
