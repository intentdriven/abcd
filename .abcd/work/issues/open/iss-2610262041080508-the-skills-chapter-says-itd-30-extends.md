---
schema_version: 1
id: "iss-2610262041080508"
slug: "the-skills-chapter-says-itd-30-extends"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 979 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/08-skills.md:127"
remedy: "Edit itd-30 (scope, interview and acceptance bullets): replace `/abcd:intent new --format=fiction` and the `refine --switch-format` surface with the canonical create `/abcd:intent` plus a format flag, as 08-skills.md describes."
related_intents: [itd-30]
finding_ref: cfk-bf3c89fa0f0fc182
---

The skills chapter says itd-30 extends the bare create `/abcd:intent "<text>"`; itd-30 adds an `/abcd:intent new` sub-verb instead.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/08-skills.md:127` — "extends the canonical create `/abcd:intent "<text>"` with a format"
- End B: `.abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18` — "`/abcd:intent new --format=fiction` opens a fiction-shaped interview"

08-skills.md describes itd-30 as a format flag on the canonical create, the bare quoted-text `/abcd:intent "<text>"`. itd-30 itself is built on `/abcd:intent new --format=fiction` and a `refine --switch-format` sub-verb. These are two different spellings for the same creation surface.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 979.
