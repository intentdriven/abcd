---
schema_version: 1
id: "iss-2610262041338234"
slug: "itd-50-says-the-intent-fidelity-reviewer"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1008 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-50]
finding_ref: cfk-c88756a252429842
---

itd-50 says the `intent-fidelity-reviewer` records today's NOT_MET verdicts; the verification matrix names that role `intent-auditor`

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/planned/itd-50-loop-toward-acceptance.md:25` — "the `intent-fidelity-reviewer` records a `NOT_MET` verdict"
- End B: `.abcd/development/brief/06-delivery/02-verification-matrix.md:56` — "When `oracle.roles.intent-auditor` reaches a provider connection a hand-run close sends that request there and ingests the verdict"

itd-50 describes current behaviour: when a shipped intent fails a criterion, the `intent-fidelity-reviewer` records NOT_MET. The verification matrix routes that same audit through `oracle.roles.intent-auditor`, and the oracle glossary calls it the intent auditor. Two names for one agent leave a reader of the planned intent looking for a reviewer that is not in the roster.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1008.
