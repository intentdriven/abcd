---
schema_version: 1
id: "iss-2610241549052683"
slug: "twelve-decisions-are-recorded-twice"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "hand-back triage of 2026-10-24, batch b13 reading iss-2608210737264758"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/work/decisions"
remedy: "Record one correcting decision, as commands/decide.md:50 requires since a record is never edited, naming each of the twelve pairs as one entry the old log held twice and the lower id as the one to cite, and add a record-lint rule that refuses two decision records with identical bodies, with the twelve known pairs in a shrink-only baseline; proven by the rule failing on a two-record fixture and passing on the tree."
---

Twelve decisions are recorded twice, under two ids with byte-identical bodies: dcn-2609290000000011 and dcn-2609290000000020 (the Group J rulings, J12 among them), dcn-2609290000000012 and dcn-2609290000000021 (ruling J13), and ten more pairs dated 2026-09-27 to 2026-09-30. The old .abcd/work/DECISIONS.md already held each of these entries twice before c2fdca9e1 migrated every entry into a record of its own, so each duplicate became a second record. None of the twelve higher ids is cited anywhere, but abcd decide lists each ruling twice and a reader cannot tell which id is the record.
