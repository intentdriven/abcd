---
schema_version: 1
id: "iss-2608301908270807"
slug: "the-decision-log-entry-says-the-record-states-the-residual-s"
severity: "minor"
category: "bug"
source: "user-observation"
found_during: "itd-179-fix-delta-ruthless"
found_at: ".abcd/work/DECISIONS.md"
resolution: "The decision log carries a dated correction naming both errors, and the residual has its own open marker rather than a pointer into a record that does not mention it."
impact: internal
resolved_by:
  intent: "itd-179"
---

the decision log entry says the record states the residual survives and it does not and it cites a label that appears nowhere

Found by the itd-179 fix-delta ruthless review. Branch-introduced by `a1ce68e0`.

The entry reads: "body scope does NOT close consequence A, and the record says
so." Two things in one sentence are wrong.

The record enumerates its consequences as **1, 2, 3**. The string "consequence
A" appears nowhere in the repository outside this line -- it was the
ORCHESTRATOR'S briefing shorthand, which leaked into the durable record.

And the record does not say so. Its `resolution` field names the diagnosis and
the mint-ordering halves only, and its consequence 1 reads as fixed to anyone
arriving fresh. So the one place the residual was mentioned pointed at a record
that does not mention it, using a label nobody can resolve.

The residual now has its own marker (iss-2608301908270888). This entry should be
restated against what the record actually enumerates, and DECISIONS.md is
append-only, so the correction is a new dated line rather than an edit.

## Grounds

- pursued: We expect an appended correction to serve a later reader better than a silent edit would, because the log is append-only by design and the error worth seeing is that briefing shorthand reached the durable record at all.
