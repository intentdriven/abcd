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
remedy: "Supersede itd-85 by shipped itd-124 with `abcd intent reclassify itd-85 --kind superseded --by itd-124 --reason ...` after checking that none of its checks (conventions router, decision durability, docs currency) is missing from the `abcd lint` targets, or rewrite its verb to a lint-shaped name; `/abcd:audit` stays reserved for itd-16's hash-chain surface (adr-40)."
related_intents: [itd-85]
finding_ref: cfk-bfe9c7aaf1e9b620
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

itd-85 names `abcd audit` as the repository-conformance check; the intent chapter says the conformance check is `abcd lint` and reserves `/abcd:audit` for hash-chain checks.

A naming conflict (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-85-audit-verb.md:16` — "a read-only check that tells you, in one command, whether a repository follows the working conventions"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:534` — "The top-level conformance check is `abcd lint`, with `/abcd:audit` reserved for itd-16's hash-chain fidelity checks."

itd-85 promises a new `abcd audit` verb that reports whether a repository follows the working conventions. 05-intent.md gives that job to `abcd lint` and reserves `/abcd:audit` for itd-16's hash-chain fidelity checks. One name cannot mean both a conformance report and the reserved hash-chain audit. Built as written, itd-85 would duplicate `abcd lint` under a name another surface already holds.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 482.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
