---
schema_version: 1
id: "iss-2610262040273336"
slug: "the-loop-glossary-says-abcd-never-owns"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 932 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/core/loop.md:38"
remedy: "none (filed automatically)"
finding_ref: cfk-14a34222fb852a7f
---

The loop glossary says abcd never owns the run loop; the build sequence says abcd's own loop ships through abcd build without the seam.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/core/loop.md:38` — "abcd never owns the run loop."
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:143` — "whose loop ships on the intent key without the seam, through `abcd build`"

The loop entry says abcd only defines the contract and a pluggable adapter supplies the run loop. The build sequence says a native Go loop is the fallback, and that the loop which shipped runs on the intent key without the seam, through abcd build. A loop that ships inside abcd is a run loop abcd owns.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 932.
