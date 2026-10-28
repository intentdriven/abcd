---
schema_version: 1
id: "iss-2610262044004875"
slug: "itd-97-builds-solo-mode-on-the-run-seam"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1177 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-97-the-facilitator-is-a-mode-not-a-person.md:60"
remedy: "none (filed automatically)"
related_intents: [itd-97, itd-2609201916151817]
finding_ref: cfk-6aa4090c941c9153
---

itd-97 builds solo mode on the run seam of itd-29; itd-2609201916151817 says itd-29 is superseded

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-97-the-facilitator-is-a-mode-not-a-person.md:60` — "mode switch, over the pluggable run seam (itd-29)."
- End B: `.abcd/development/intents/planned/itd-2609201916151817-one-verb-takes-a-single-intent-from.md:69` — "`itd-29` is superseded by this intent."

itd-97 composes solo mode (scheduled sync, self-firing reviews, run initiation) over the pluggable run seam it names as itd-29. itd-2609201916151817 decision 7 supersedes itd-29, moves its budget check into the pacing intent and drops its other verbs. Solo mode's stated substrate is a record that has been retired.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1177.
