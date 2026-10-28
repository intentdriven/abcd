---
schema_version: 1
id: "iss-2610262035089991"
slug: "itd-46-calls-bare-command-as-status-a"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 550 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-46]
finding_ref: cfk-65b28dcb4a852949
---

itd-46 calls bare-command-as-status a universal abcd convention; the intent chapter says it is a common convention, not a universal one

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:18` — "The bare command (zero arguments) keeps doing what the universal abcd convention says it must: shows status and help, never mutates state."
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:344` — "Bare-command-as-help is a common abcd convention, not a universal one"

itd-46 rests its bare-form behaviour on what the universal abcd convention says it must do. The intent chapter says outright that bare-command-as-help is a common abcd convention, not a universal one, and names commands that print help or a version block instead of a board. One document asserts a rule with no exceptions and the other asserts that exceptions exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 550.
