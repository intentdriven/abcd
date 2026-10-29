---
schema_version: 1
id: "iss-2610262040177541"
slug: "universal-patterns-call-itd-60-a-planned"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 921 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md:144"
remedy: "Edit 04-universal-patterns.md section 6: rewrite the doc-fidelity entry and the forward-doc-sync marker paragraph as shipped (itd-60 is in shipped/ and internal/core/docfidelity exists), checking the layout against the code."
related_intents: [itd-60]
finding_ref: cfk-1f8b21e2377b8586
---

Universal patterns call itd-60 a planned draft that is not yet built; itd-60 is shipped, and its gate refuses at spec close

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:144` — "(`intents/planned/itd-60-doc-fidelity-anti-drift.md`; not yet built)"
- End B: `.abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:124` — "`abcd spec close`, the verb that moves a planned intent to `shipped/`, refuses per intent while the brief lags the surface that intent delivered"

04-universal-patterns describes the doc-fidelity pass as planned under a draft itd-60 at intents/planned/ and not yet built, and plans its log-tier layout as a later addition. itd-60 sits in shipped/ and says `abcd spec close` refuses per intent while the brief lags the surface it delivered. The gate cannot be both unbuilt and enforcing at the close.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 921.
