---
schema_version: 1
id: "iss-2610262044265126"
slug: "the-relocatable-home-draft-has-abcd-move"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1206 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-77-relocatable-user-home.md:17"
remedy: "none (filed automatically)"
related_intents: [itd-77, itd-2610030720038073]
finding_ref: cfk-4342c3d4c83679f1
---

The relocatable-home draft has abcd move its home on command, but the shipped rename ruled that abcd moves nothing and fixed the name at ~/.abcd.noindex

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-77-relocatable-user-home.md:17` — "a guided, wizard-style flow moves the home wherever the user's filing system wants it"
- End B: `.abcd/development/intents/shipped/itd-2610030720038073-abcd-keeps-its-home-folder-out-of.md:99` — "abcd moves nothing. An existing `~/.abcd` stops every abcd command and hook before it writes anything"

itd-77 promises a guided flow in which abcd moves the user-level home anywhere and re-points every consumer. itd-2610030720038073 dropped abcd's own move of the home ('abcd moves nothing') and carries no move code. It also relies on the `.noindex` name for the indexer to skip the folder, which a home moved by name elsewhere would lose. The two records disagree on whether abcd moves its home.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1206.
