---
schema_version: 1
id: "iss-2610262031496655"
slug: "scope-says-itd-29-is-superseded-but"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 181 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/04-scope.md:19"
remedy: "Edit draft itd-97: cite the run machinery of itd-2609201916151817 (the build and implement loop) in place of the superseded itd-29 run seam, in the scope line and the related-intents line."
related_intents: [itd-97]
finding_ref: cfk-a65851931342f1ab
---

Scope says itd-29 is superseded, but itd-97 builds solo mode over the pluggable run seam it names as itd-29.

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/04-scope.md:19` — "itd-29 is superseded (by itd-2609201916151817, planned)"
- End B: `.abcd/development/intents/drafts/itd-97-the-facilitator-is-a-mode-not-a-person.md:60` — "mode switch, over the pluggable run seam (itd-29)."

04-scope records itd-29 as superseded by itd-2609201916151817, and says no `run` verb is on any shipped surface. itd-97 builds solo mode's composed 'when' (scheduled sync, self-firing reviews, run initiation) 'over the pluggable run seam (itd-29)'. As written, that is a dependency on a record that will never deliver.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 181.
