---
schema_version: 1
id: "iss-2610262031090459"
slug: "the-naming-register-relies-on-role-2-to"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 93 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/04-naming.md:155"
remedy: "none (filed automatically)"
finding_ref: cfk-7c42e1a37abb3fe3
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030481186. Same A end (Role 2 verifies registration on every plan-review) and same fix; the agent catalog's own verb, reading only the brief and intents, is the second witness."
duplicates: [iss-2610262030481186]
---

The naming register relies on Role 2 to verify spec vocabulary registration at every plan-review, but Role 2 reads only the brief and the intents, through its own verb

A sequencing impossibility (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/04-naming.md:155` — "The Role 2 cross-document audit verifies registration on every plan-review."
- End B: `.abcd/development/brief/05-internals/01-agents.md:163` — "It reads the brief and every intent outside `superseded/` and reports terminology drift"

The vocabulary-registration requirement applies to terms a spec introduces in its Modification Grammar > Ripple > Vocabulary delta, and the naming register says the Role 2 cross-document audit verifies registration on every plan-review. The agent catalog defines Role 2 as `abcd intent consistency`, which reads the brief and every intent outside `superseded/`. It reads no spec, and it runs as its own verb rather than at plan-review. As written, Role 2 cannot see the spec where the term is introduced, so it cannot meet the dependency the register puts on it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 93.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536) as a duplicate of iss-2610262030481186. Same A end (Role 2 verifies registration on every plan-review) and same fix; the agent catalog's own verb, reading only the brief and intents, is the second witness.
