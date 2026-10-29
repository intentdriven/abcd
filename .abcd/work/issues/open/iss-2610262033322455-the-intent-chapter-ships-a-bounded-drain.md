---
schema_version: 1
id: "iss-2610262033322455"
slug: "the-intent-chapter-ships-a-bounded-drain"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 417 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:328"
remedy: "Rewrite the Owed-review draining paragraph at 03-configuration.md lines 405-412: the on-demand bounded drain ships as the audit verb's owed form (itd-53) and the build's landing takes the audit; keep as staged only seam-iteration gating, if still intended."
finding_ref: cfk-de5f62fdc784540c
---

The intent chapter ships a bounded drain for owed audits, while the configuration chapter says nothing drains today.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:328` — "The bounded command that pays the review debt"
- End B: `.abcd/development/brief/05-internals/03-configuration.md:408` — "No `run` seam ships, so nothing drains today."

05-intent describes the audit sub-verb's owed form as the shipped bounded command that pays the review debt: oldest shipped first, up to a cap. 03-configuration says that because no run seam ships, nothing drains today. Owed fidelity reviews either have a shipped drain or have none.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 417.
