---
schema_version: 1
id: "iss-2610262039169067"
slug: "itd-61-says-abcd-already-carries-built"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 849 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-61-brief-change-derivation.md:24"
remedy: "In 04-universal-patterns.md (the doc-fidelity entry in the logs layout and the forward-source marker paragraph) replace 'draft intent itd-60 ... planned ... not yet built' with the shipped state: itd-60 sits in intents/shipped/ and the gate is internal/core/docfidelity; check the content-addressed layout and paired-stamp wording against what that store writes."
related_intents: [itd-61]
finding_ref: cfk-773c91fe27c0bf7d
---

itd-61 says abcd already carries built reality forward into the brief (itd-60), but universal patterns says that forward pass is planned and not yet built

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-61-brief-change-derivation.md:24` — "abcd already carries built reality forward into the brief and docs"
- End B: `.abcd/development/brief/05-internals/04-universal-patterns.md:144` — "the doc-fidelity anti-drift pass planned under **draft intent itd-60**"

itd-61 presents itself as the reverse of a forward doc-fidelity pass that already exists, and dedups against brief edits that pass auto-drafted. 04-universal-patterns describes the itd-60 forward pass, its content-addressed tier and its paired stamps as planned under a draft intent and not yet built. itd-61's dedup depends on stamps that, by that page, nothing writes yet.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 849.
