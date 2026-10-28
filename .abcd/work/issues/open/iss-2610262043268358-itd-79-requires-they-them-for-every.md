---
schema_version: 1
id: "iss-2610262043268358"
slug: "itd-79-requires-they-them-for-every"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1137 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-79-persona-registry.md:18"
remedy: "none (filed automatically)"
related_intents: [itd-79, itd-2610180912189125]
finding_ref: cfk-d03e74e2d34b13fa
---

itd-79 requires they/them for every persona; itd-2610180912189125 refers to its persona Alice as 'her'

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-79-persona-registry.md:18` — "the real user, when referred to at all, is also they/them"
- End B: `.abcd/development/intents/shipped/itd-2610180912189125-a-session-told-to-work-on-while-its.md:20` — "a push her standing instructions do not allow"

The persona discipline rules that every persona, and the real user, is referred to as they/them. The shipped away-mode press release uses Alice as its persona and writes 'a push her standing instructions do not allow', while the rest of the same paragraph uses they. The shipped text breaks the pronoun half of the discipline.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1137.
