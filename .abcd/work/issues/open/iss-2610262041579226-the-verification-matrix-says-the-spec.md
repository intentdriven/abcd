---
schema_version: 1
id: "iss-2610262041579226"
slug: "the-verification-matrix-says-the-spec"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1036 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/06-delivery/02-verification-matrix.md:79"
remedy: "none (filed automatically)"
related_intents: [itd-28]
finding_ref: cfk-6adf70ba083b4357
---

The verification matrix says the spec core has no review store; shipped itd-28's press release says every spec-tied review lands in the native review store.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/06-delivery/02-verification-matrix.md:79` — "The spec core carries no review store, and the review that ships is intent-grained"
- End B: `.abcd/development/intents/shipped/itd-28-rp-reviews-into-flow.md:33` — "abcd lands every spec-tied review next to the spec it reviewed, in the native review store, pinned to the commit it reviewed"

The matrix's spec-tied-reviews row for itd-28 is staged. It says the spec core carries no review store and the review that ships is intent-grained. itd-28 sits in shipped/, but its press release still announces that every spec-tied review lands next to its spec in the native review store. Its own decision 1 dropped that store in favour of pin-and-staleness, so its press release and the matrix disagree about what shipped. The matrix also has no row for the pin-and-staleness behaviour that was delivered.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1036.
