---
schema_version: 1
id: "iss-2610262040053417"
slug: "the-readme-ships-an-intent-only-when-no"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 908 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md:275"
remedy: "none (filed automatically)"
related_intents: [itd-80]
finding_ref: cfk-bafba7ff86574416
refines: [iss-2609100508566552]
---

The README ships an intent only when no open spec still names it, because an intent owns one or more specs; itd-80's reconcile ships the linked intent whenever its spec closes

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/README.md:275` — "only when no open spec is left naming it — an intent owns one or more specs"
- End B: `.abcd/development/intents/shipped/itd-80-intent-lifecycle-automation.md:34` — "on spec close, detect the linked intent via the immutable link, move it `planned → shipped` with an atomic same-filesystem rename"

The README's spec row says a spec close ships the linked intent only when no open spec is left naming it, since an intent owns one or more specs. itd-80 says that on spec close the reconcile finds the linked intent through the link and moves it from planned to shipped, which assumes one spec per intent. When an intent has two specs, closing the first ships it under itd-80 but not under the README.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 908.
