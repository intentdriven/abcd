---
schema_version: 1
id: "iss-2610262032465065"
slug: "the-build-sequence-says-ahoy-install"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 314 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/01-ahoy.md:848"
remedy: "In 06-delivery/01-build-sequence.md (the Install bullet near line 55) drop 'carrying its configuration as flags rather than a wizard': at a terminal the install asks each value question through the drawn door, and the flags pre-answer those questions."
finding_ref: cfk-d4d5717d70fa1e17
---

the build sequence says ahoy install carries its configuration as flags rather than a wizard; ahoy draws each value question and approval one at a time at a terminal

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/01-ahoy.md:848` — "the install asks through the drawn door instead of the line reader"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:55` — "carrying its configuration as flags rather than a wizard"

01-build-sequence describes abcd ahoy install as carrying its configuration as flags rather than a wizard. 01-ahoy specifies that at a terminal the install asks through the drawn door, putting each value question and each approval as a numbered Setup question through an answer loop, with flags only pre-answering those prompts. A question-by-question interactive setup is a wizard, so the two accounts of the install's shape disagree.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 314.
