---
schema_version: 1
id: "iss-2608300929274006"
slug: "report-and-gate-filename-grammars-diverge-for-rdi-and-dsp"
severity: "minor"
category: "inconsistency"
source: "impl-review"
found_during: "itd-189 build review, 2026-08-30"
found_at: "internal/core/lint/readingoutstanding.go, internal/core/issueschema/disposition.go"
---

The outstanding report's filename grammar for reading items and dispositions (readingItemFileRe, DispositionFileID) is stricter than the record_schema gate's FilenameNumRe, so a hand-written rdi-N-slug.md or dsp-N-slug.md passes the gate and is then invisible to the report; for those two families the divergence fails toward silence rather than a false claim. One grammar, the resolver's, for every family the report walks.

Extended 2026-08-30 by itd-189 round 4, which had to work around this rather
than fix it. The divergence is the ROOT the round's new spelling leg is built
on: `admittedProposals` and `readingItemFileRe` key on the filename stem and
never open the item's `id`, so the round's padding check had to compare a
join's value against the TARGET'S FILENAME rather than against its declared id.

That leaves one shape deliberately silent. A target whose filename is not
itself a bare handle, `rdi-2-widen-the-frame.md`, admits under no spelling at
all, because the family's reader never reads that file. The new leg does not
report it, and reporting it would be a claim about a file the reader does not
consume. The silence belongs to this record, not to the spelling leg: close
this and the shape stops existing.

