---
schema_version: 1
id: "iss-2610262041364605"
slug: "the-record-families-page-claims-to-be"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1012 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/record-families.md:36"
remedy: "none (filed automatically)"
related_intents: [itd-2609020625402518]
finding_ref: cfk-130f4160b20a70fe
---

The record-families page claims to be the one map of record families and lists seven; the reframe intent adds a reframe record family beside other reading families, none of which has a row

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/record-families.md:36` — "a name that has no row here is a name nobody has justified"
- End B: `.abcd/development/intents/shipped/itd-2609020625402518-a-reframe-occasioned-by-a-reading-is.md:42` — "A reframe record family"

The record-families page presents itself as the one page that maps abcd's record families (intent, spec, step, bundle, issue, release, status), and says a name with no row is a name nobody has justified. The shipped reframe intent creates a reframe record family in the working tier beside the other reading families (readings, dispositions, admissions, surprises), and none of them has a row. Either the map is incomplete or those families are unjustified. The two documents cannot both be right.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1012.
