---
schema_version: 1
id: "iss-2610262035399502"
slug: "the-identity-chapter-s-initialiser"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 588 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/19-identity.md:91"
remedy: "none (filed automatically)"
finding_ref: cfk-9869f5cb50bbb4b8
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035387982. Same End B sentence; identity's initialiser write path is another exception to the read-only claim."
duplicates: [iss-2610262035387982]
---

The identity chapter's initialiser records the block and its pointer; universal patterns lists `identity` among the read-only renders.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/19-identity.md:91` — "The initialiser records the block and the pointer to it."
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:90` — "`lint` and its targets, `capture`, `intent`, `guard`, `--version`, `rules`, `spec`, `identity` and `banlist` are read-only renders, and `abcd lint` in particular is documented as performing zero writes"

19-identity says the report and the render are read-only and that initialisation is a write path: the initialiser records the identity block and the pointer to it. 04-universal-patterns lists `identity` with the verbs it calls read-only renders, with no exception for that write.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 588.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035387982. Same End B sentence; identity's initialiser write path is another exception to the read-only claim.
