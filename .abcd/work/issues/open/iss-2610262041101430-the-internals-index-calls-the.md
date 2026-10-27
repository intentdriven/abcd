---
schema_version: 1
id: "iss-2610262041101430"
slug: "the-internals-index-calls-the"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 981 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/README.md:29"
remedy: "In the itd-30 draft rename intent-fidelity-reviewer to intent-auditor (the rename is recorded in the agents changelog), when the draft is next edited or planned."
related_intents: [itd-30]
finding_ref: cfk-16510a9e3988ade6
---

The internals index calls the intent-vs-implementation reviewer `intent-auditor`; itd-30 calls it `intent-fidelity-reviewer`.

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/README.md:29` — "When `intent-auditor` matures, brief-skeleton presence checks would land alongside its existing intent-vs-implementation checks."
- End B: `.abcd/development/intents/drafts/itd-30-design-fictions-as-intent-format.md:18` — "`intent-fidelity-reviewer` reads either dialect against the same delivered reality"

05-internals/README.md names `intent-auditor` as the agent that runs the intent-vs-implementation checks. itd-30 gives that same job, reading an intent against delivered reality, to an agent it calls `intent-fidelity-reviewer`, so one agent has two names.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 981.
