---
schema_version: 1
id: "iss-2610262034151413"
slug: "itd-85-names-abcd-audit-as-the"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 482 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-85-audit-verb.md:16"
remedy: "none (filed automatically)"
related_intents: [itd-85]
finding_ref: cfk-bfe9c7aaf1e9b620
---

itd-85 names `abcd audit` as the repository-conformance check; the intent chapter says the conformance check is `abcd lint` and reserves `/abcd:audit` for hash-chain checks.

A naming conflict (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-85-audit-verb.md:16` — "a read-only check that tells you, in one command, whether a repository follows the working conventions"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:534` — "The top-level conformance check is `abcd lint`, with `/abcd:audit` reserved for itd-16's hash-chain fidelity checks."

itd-85 promises a new `abcd audit` verb that reports whether a repository follows the working conventions. 05-intent.md gives that job to `abcd lint` and reserves `/abcd:audit` for itd-16's hash-chain fidelity checks. One name cannot mean both a conformance report and the reserved hash-chain audit. Built as written, itd-85 would duplicate `abcd lint` under a name another surface already holds.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 482.
