---
schema_version: 1
id: "iss-2610262032022112"
slug: "the-intent-s-back-link-to-its-spec-is"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 213 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/02-constraints/03-invariants.md:171"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-599995dba6c24f23
wontfix_reason: "Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The glossary's spec and bundle entries list epic as a forbidden synonym, and internal/core/intent reads spec_id. A shipped record is not rewritten for vocabulary."
---

The intent's back-link to its spec is the scalar `spec_id` in invariant 17 and `epic_id` in itd-34.

A naming conflict (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/02-constraints/03-invariants.md:171` — "the intent's scalar `spec_id` names the spec it was planned with"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:52` — "each intent's `epic_id: spc-N` and `bundle: <bundle-id>`"

Invariant 17 names the intent field that points at the spec it was planned with `spec_id`. itd-34 writes the same link onto each bundle member as `epic_id: spc-N`. That is two names for one frontmatter field, and a reader building the plan path cannot tell which key to write.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 213.

## Grounds

- declined: Closed in the 2026-10-27 triage (dcn-2610271112330536): a later record already settles which end holds. The glossary's spec and bundle entries list epic as a forbidden synonym, and internal/core/intent reads spec_id. A shipped record is not rewritten for vocabulary.
