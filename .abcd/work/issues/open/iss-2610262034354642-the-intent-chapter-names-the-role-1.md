---
schema_version: 1
id: "iss-2610262034354642"
slug: "the-intent-chapter-names-the-role-1"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 507 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/05-intent.md:532"
remedy: "none (filed automatically)"
related_intents: [itd-50]
finding_ref: cfk-44eef02066b644d2
---

The intent chapter names the Role 1 fidelity agent intent-auditor; itd-50 calls the same actor intent-fidelity-reviewer.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/05-intent.md:532` — "`intent-auditor` is a single agent in the catalog"
- End B: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:25` — "the `intent-fidelity-reviewer` records a `NOT_MET` verdict"

05-intent states that intent-auditor is a single agent in the catalog owning three roles, Role 1 being the one that records per-criterion NOT_MET verdicts, and reserves the word review for family-1 change judgement under adr-40. itd-50's press release has the intent-fidelity-reviewer record the NOT_MET verdict. Two names for one agent, one of them in the register the chapter reserves for change review, leaves a builder of the loop unsure which agent it dispatches.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 507.
