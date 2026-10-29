---
schema_version: 1
id: "iss-2610262035390841"
slug: "the-guard-chapter-says-the-guard-writes"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 587 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md:6"
remedy: "none (filed automatically)"
finding_ref: cfk-6579576e61d71fa8
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035387982. Same End B sentence; guard is one more verb it wrongly lists among read-only renders."
duplicates: [iss-2610262035387982]
---

The guard chapter says the guard writes a created-branch record, the waiting-on mode and an open-question marker; universal patterns lists `guard` among the read-only renders.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/17-guard.md:6` — "It writes nothing but the record of a branch an agent creates and, when it admits abcd's own question where the badge shows, the waiting-on mode and the marker saying a question is open"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:90` — "`lint` and its targets, `capture`, `intent`, `guard`, `--version`, `rules`, `spec`, `identity` and `banlist` are read-only renders, and `abcd lint` in particular is documented as performing zero writes"

17-guard says the guard writes the record of a branch an agent creates and, when it admits abcd's own question, the waiting-on mode and the `question_open` marker. 04-universal-patterns lists `guard` with the verbs it calls read-only renders. A verb that writes three kinds of state is not a read-only render.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 587.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262035387982. Same End B sentence; guard is one more verb it wrongly lists among read-only renders.
