---
schema_version: 1
id: "iss-2608301747001641"
slug: "a-malformed-grounds-bullet-on-an-issue"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-179-round-5-builder"
found_at: "internal/core/grounds"
resolution: "record_schema now reports every bullet under ## Grounds that the reader drops, on issue and intent records, from grounds.MalformedIn, the other half of the one walk ParseSection takes; the grammar is judged, not the floor, and prose paragraphs stay silent. The tree's one such bullet (iss-2609012039210402) is corrected. Proven by TestRecordSchemaReportsAMalformedGroundsBullet (watched red) and TestMalformedInIsTheReadersComplement."
impact: fix
resolved_by:
  commit: "bc9fca94"
---

a malformed grounds bullet on an issue record reads as prose so no gate can review it losing coverage the frontmatter rule had

Reported by the round-5 builder as a deliberate consequence of the shared
reader's contract, not an oversight. Recorded because it is COVERAGE THE
FRONTMATTER RULE HAD AND THE SECTION FORM DOES NOT.

A `grounds:` key with a malformed value was a value the schema gate could judge.
A malformed bullet under `## Grounds` is prose: the reader takes what parses and
the rest reads as body text, so nothing is wrong to any gate and nothing says
so. The entry is simply not there, silently.

That is the same shape as the class this branch has been closing all cycle -- a
reader declining to see something while nothing downstream knows -- arrived at
from the other direction, by moving data somewhere the gate does not look. Worth
weighing before the section form is copied to any other field.

## Grounds

- pursued: a hand-typed bullet the grounds reader drops is named by the record gate; a malformed bullet passing record-lint while no surface shows it would show it wrong
