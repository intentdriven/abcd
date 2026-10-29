---
schema_version: 1
id: "iss-2610262033434648"
slug: "the-out-of-scope-page-says-itd-27-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 433 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/03-out-of-scope.md:166"
remedy: "Rewrite the 'Phased-in additions captured post-brief' paragraph: itd-27 is superseded by itd-94 (no grill sub-verb), itd-28 and itd-34 are shipped, and there is no phase to scope them into."
finding_ref: cfk-4c241131bd4cdba6
---

The out-of-scope page says itd-27 (the grill sub-verb) is scoped into the planned phases; the intent chapter says itd-27 is superseded by itd-94 and no grill sub-verb ships.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/03-out-of-scope.md:166` — "itd-27 (`/abcd:intent grill` sub-verb + glossary), itd-28 (spec-tied reviews in the native spec review store), and itd-34 (three intent kinds with three lifecycle paths) were captured after this brief was written and are scoped into the planned phases."
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:319` — "per itd-27, `intents/superseded/`, superseded by itd-94; no grill sub-verb ships"

03-out-of-scope lists itd-27 ('/abcd:intent grill' sub-verb + glossary) among additions that 'are scoped into the planned phases', which presents it as live, scheduled work. 05-intent records itd-27 as sitting in intents/superseded/, superseded by itd-94, with no grill sub-verb shipping. A superseded intent is not scheduled work, so the out-of-scope note cannot hold as written.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 433.
