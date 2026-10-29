---
schema_version: 1
id: "iss-2610262041112989"
slug: "the-manual-hold-draft-relies-on-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 983 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/01-build-sequence.md:141"
remedy: "Supersede or re-scope itd-57: a manual hold already ships as abcd intent hold and unhold (a held line on a draft or planned record), while the run seam, abcd configure and a spec dependency graph do not ship, so the sentinel-spec design has nothing to ride on."
related_intents: [itd-57]
finding_ref: cfk-a57b23115609cb09
---

The manual-hold draft relies on a run-seam selector that already skips blocked specs; the build sequence says the run seam does not ship.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/01-build-sequence.md:141` — "The seam does not ship: the binary registers no `run` verb"
- End B: `.abcd/development/intents/drafts/itd-57-manual-hold-sentinel.md:17` — "the run seam's selector already skips specs blocked by a non-done dependency, which does the rest"

itd-57 says it needs no new logic because the run seam's selector already skips specs with a non-done dependency. 01-build-sequence.md says the seam does not ship: no `run` verb, the run adapter only planned. The draft's hold depends on a selector that does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 983.
