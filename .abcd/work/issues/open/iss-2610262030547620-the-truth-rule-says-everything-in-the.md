---
schema_version: 1
id: "iss-2610262030547620"
slug: "the-truth-rule-says-everything-in-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 59 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/00-meta.md:7"
remedy: "Delete the six-section board design target ('The board itself is not built' and the sections that follow) from 08-abcd.md and keep 'What ships today'; itd-20 is superseded by shipped itd-121 and no spec builds it. Related open records name the same section: iss-2610262030511283 and iss-2610262033526755."
finding_ref: cfk-4e08cd88034dd244
---

The truth rule says everything in the brief is decided; the board chapter's design target rests on a superseded intent with no spec

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/00-meta.md:7` — "Everything in the brief is decided."
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:519` — "Design target (itd-20, `intents/superseded/`, superseded by itd-121, `spec_id: null`)."

00-meta says everything in the brief is decided and a staged section is a target a future change builds toward, while 08-abcd presents a six-section board as a design target whose intent itd-20 is superseded by itd-121 with spec_id null, and leaves open whether one of its sections is dropped. No recorded future change builds toward it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 59.
