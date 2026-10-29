---
schema_version: 1
id: "iss-2610262030459678"
slug: "the-press-release-lists-intent-grill-as"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 41 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:26"
remedy: "Remove the '/abcd:intent grill (per itd-27)' companion from the Forward-looking discipline bullet (line 26): itd-27 is superseded by shipped itd-94, whose planning interview is the grilling. The staged list in 02-constraints/04-naming.md still names it and needs the same edit."
finding_ref: cfk-0c2d6c5af44f467f
refines: [iss-2609091956001547]
---

The press release lists intent grill as a designed companion per itd-27; the intent chapter says itd-27 is superseded by itd-94

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:26` — "Two companions are designed and not yet built: `/abcd:intent grill` (per itd-27)"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:319` — "(per itd-27, `intents/superseded/`, superseded by itd-94; no grill sub-verb ships)"

01-press-release says /abcd:intent grill (per itd-27) is designed and not yet built, a live design target, while 05-intent says itd-27 sits in intents/superseded/, superseded by itd-94, and no grill sub-verb ships. A design target whose backing intent is superseded is no longer what the record says will be built.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 41.
