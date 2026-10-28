---
schema_version: 1
id: "iss-2610262039195057"
slug: "itd-85-makes-abcd-audit-a-repository"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 853 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-85-audit-verb.md:26"
remedy: "none (filed automatically)"
related_intents: [itd-85]
finding_ref: cfk-d694517758725555
---

itd-85 makes `abcd audit` a repository conformance check, but the agent catalogue reserves the top-level `/abcd:audit` for itd-16's hash-chain surface

A naming conflict (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-85-audit-verb.md:26` — "A new read-only `abcd audit` verb, distinct from `ahoy doctor`"
- End B: `.abcd/development/brief/05-internals/01-agents.md:146` — "the top-level `/abcd:audit` stays reserved"

itd-85 adds `abcd audit` as a top-level, read-only conformance verb. The agent catalogue says adr-40 reserves the `audit` name for promise-versus-reality verdicts, and that the top-level `/abcd:audit` stays reserved for itd-16's hash-chain surface. One name would cover two concepts: building both puts the hash-chain sub-verbs and the conformance report under one verb with incompatible meanings.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 853.
