---
schema_version: 1
id: "iss-2610262031419126"
slug: "the-naming-register-reserves-abcd-audit"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 164 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:78"
remedy: "Rename the verb in itd-85 to a lint-shaped name (or supersede it by the shipped `abcd lint`) and drop `abcd audit` and commands/audit.md from its scope; adr-40 ruled that itd-85 gives `/abcd:audit` back to itd-16 as the register reserves it."
related_intents: [itd-85]
finding_ref: cfk-3176179da5bbe5a0
deferred_after: v0.15.1
deferral_reason: "Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes."
---

The naming register reserves `/abcd:audit` for itd-16's formal-verification surface, but itd-85 takes `abcd audit` for a repo-conformance check.

A naming conflict (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:78` — "formal verification surface: hash-chain and Merkle audit trails, fidelity checks (itd-16, a later phase)"
- End B: `.abcd/development/intents/drafts/itd-85-audit-verb.md:26` — "A new read-only `abcd audit` verb, distinct from `ahoy doctor` (doctor = tool-setup health; audit = repo conformance)."

04-naming lists `/abcd:audit` among the reserved meta-development commands, 'named now to prevent collisions', as the formal verification surface (hash-chain and Merkle audit trails, fidelity checks, itd-16). itd-85 scopes a new `abcd audit` verb meaning repo conformance (three-tier layout, AGENTS.md router, docs currency, privacy hygiene). One verb name would cover two different surfaces, and whichever ships first takes the name the other was promised.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 164.

## Deferral 2026-10-27

Deferred past v0.15.1: Deferred by the product thinker on 2026-10-27 (dcn-2610271112330536): the brief's single-home rewrite (dcn-2610271112339516, dcn-2610271112330235) rewrites or removes most of the passages this record quotes.
