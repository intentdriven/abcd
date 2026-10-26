---
schema_version: 1
id: "iss-2610262031280238"
slug: "the-press-release-counts-abcd-run-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 134 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:27"
remedy: "none (filed automatically)"
finding_ref: cfk-f821798d0703f103
---

The press release counts /abcd:run, the itd-29 operator surface, among the plumbing in scope; the build sequence says the run seam does not ship, no run verb is registered and itd-29 is superseded.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:27` — "e.g. `/abcd:run`, the itd-29 autonomous-run operator surface"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:141` — "The seam does not ship: the binary registers no `run` verb"

The press release's scope lists operator-internal command wiring 'e.g. `/abcd:run`, the itd-29 autonomous-run operator surface' over the pluggable run seam, and names /abcd:intent grill per itd-27 as a designed companion. The build sequence says 'The seam does not ship: the binary registers no `run` verb', that the itd-29 surface is superseded by itd-2609201916151817, and that itd-27's grilling is the planning interview under itd-94. The press release presents as in scope two surfaces the build sequence records as superseded.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 134.
