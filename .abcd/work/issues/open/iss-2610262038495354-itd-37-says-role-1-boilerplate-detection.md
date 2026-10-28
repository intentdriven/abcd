---
schema_version: 1
id: "iss-2610262038495354"
slug: "itd-37-says-role-1-boilerplate-detection"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 821 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-37-modification-grammar.md:69"
remedy: "none (filed automatically)"
related_intents: [itd-37]
finding_ref: cfk-06cab784fc133ace
---

itd-37 says Role 1 boilerplate detection runs at plan-review and ship time; 01-agents says the auditor's modification-grammar output is deferred and in no prompt or lint

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-37-modification-grammar.md:69` — "Role 1 boilerplate detection runs at plan-review and ship time"
- End B: `.abcd/development/brief/05-internals/01-agents.md:160` — "term-drift, PRD-fidelity and modification-grammar outputs the role was drawn with are **deferred**: none is in the shipped prompt or in any lint."

itd-37 says every spec inherits Role 1 boilerplate detection, run at plan-review and at ship time. 01-agents says the modification-grammar output the auditor role was drawn with is deferred, and is in neither the shipped prompt nor any lint. A spec author would rely on a gate that does not run.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 821.
