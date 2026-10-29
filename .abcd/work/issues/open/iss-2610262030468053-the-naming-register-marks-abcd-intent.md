---
schema_version: 1
id: "iss-2610262030468053"
slug: "the-naming-register-marks-abcd-intent"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 42 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:43"
remedy: "In 04-naming.md drop the 'staged (itd-27)' claim for `/abcd:intent grill` (line 43 and the staged sub-verb list near line 117); itd-27 is in intents/superseded/ (superseded_by itd-94) and the grilling is the shipped `abcd intent interview`."
finding_ref: cfk-7c3bb8c4b5c2c617
---

The naming register marks /abcd:intent grill staged under itd-27; the intent chapter says itd-27 is superseded

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:43` — "`/abcd:intent grill` is **staged** (itd-27), and no `intent grill` sub-verb is registered"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:319` — "(per itd-27, `intents/superseded/`, superseded by itd-94; no grill sub-verb ships)"

04-naming records /abcd:intent grill as staged (itd-27), which under the truth rule means a decided design target, while 05-intent records itd-27 as superseded by itd-94 with no grill sub-verb. The sub-verb cannot be staged on an intent the record has retired.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 42.
