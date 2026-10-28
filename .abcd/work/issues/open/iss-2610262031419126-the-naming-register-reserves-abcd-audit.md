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
remedy: "none (filed automatically)"
related_intents: [itd-85]
finding_ref: cfk-3176179da5bbe5a0
---

The naming register reserves `/abcd:audit` for itd-16's formal-verification surface, but itd-85 takes `abcd audit` for a repo-conformance check.

A naming conflict (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:78` — "formal verification surface: hash-chain and Merkle audit trails, fidelity checks (itd-16, a later phase)"
- End B: `.abcd/development/intents/drafts/itd-85-audit-verb.md:26` — "A new read-only `abcd audit` verb, distinct from `ahoy doctor` (doctor = tool-setup health; audit = repo conformance)."

04-naming lists `/abcd:audit` among the reserved meta-development commands, 'named now to prevent collisions', as the formal verification surface (hash-chain and Merkle audit trails, fidelity checks, itd-16). itd-85 scopes a new `abcd audit` verb meaning repo conformance (three-tier layout, AGENTS.md router, docs currency, privacy hygiene). One verb name would cover two different surfaces, and whichever ships first takes the name the other was promised.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 164.
