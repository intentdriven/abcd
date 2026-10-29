---
schema_version: 1
id: "iss-2610262030331903"
slug: "the-mental-model-gives-modification"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 14 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:93"
remedy: "Edit 03-mental-model.md (Naurian gap paragraph): say a spec's `## Modification Grammar` section carries three sub-headings (Extends cleanly / Breaks the design / Why) plus the Ripple axis, four axes in all."
finding_ref: cfk-7386cca5921d40bf
---

The mental model gives Modification Grammar three sub-headings; the tradeoffs page records a four-axis section that includes Ripple

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:93` — "every spec carries a `## Modification Grammar` section with three sub-headings (`Extends cleanly` / `Breaks the design` / `Why`)"
- End B: `.abcd/development/brief/03-evidence/04-tradeoffs.md:45` — "a 4-axis `## Modification Grammar` section per spec (`Extends cleanly` / `Breaks the design` / `Why` / `Ripple`)"

The mental model says every spec's `## Modification Grammar` section has three sub-headings: Extends cleanly, Breaks the design and Why. The tradeoffs entry records the choice as a four-axis section with Ripple as the fourth, and the mental model itself cites a `## Modification Grammar > Ripple` sub-bullet elsewhere. A spec author cannot follow both counts.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 14.
