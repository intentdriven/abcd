---
id: itd-79
slug: persona-registry
kind: discipline
kind_notes: "Promoted from principles/personas-alice-bob-carol.md the day its registry lint shipped, per the principles→disciplines promotion path (enforced principle ⇒ discipline). Gate delivered by the record-lint persona_registry rule."
suggested_kind: null
spec_id: null
reclassification_history: []
severity: minor
blocked_by: []
builds_on: []
---

# Personas Come From the Registry; the Role Picks the Name

## Rule

Every persona in a user story, press-release quote, worked example, or scenario comes from the registry at `.abcd/development/personas.json` — the single source of truth for the roster. Selection is **by role, never by name**: the scenario's audience determines the role, and the role's registered name is used. Names form a fixed alphabetical sequence (Alice, Bob, Carol, Dave, …); when the roster grows, new personas continue the sequence. No invented names, no real people's names as stand-ins. Every persona is referred to as **they/them**; the real user, when referred to at all, is also they/them.

The name half of the rule is mechanically enforced: the `persona_registry` record-lint rule (blocker) flags any quote attribution of the form `said <Name>,` — Unicode-wide, including compound names — whose name is not in the registry. Attributions phrased any other way, the role-match, and the pronoun halves are review-enforced.

## Why

Persona names are load-bearing noise: a bespoke name per document makes readers wonder whether the name carries meaning, and a real-sounding name risks colliding with an actual person or project. Binding names to roles makes them carry exactly one bit of meaning — the audience — consistently across the whole corpus: Frank is always the DevOps voice, Kira always the maintainer's, Iris the product thinker's. The alphabetical sequence keeps the roster auditable and extension mechanical.

The 2026-07-08 dependency sweep demonstrated the failure mode this kills: twenty quote attributions had drifted to invented names (Theo, Mara, Priya, Fatima, "Dev") or role-mismatched roster names, hand-corrected in one pass. A registry with a lint gate prevents the drift from re-accumulating.

## What's In Scope

- The registry file (`personas.json`: roster, role hints, selection convention) as the roster's single source of truth.
- The `persona_registry` record-lint rule: quote-attribution names must be registry members; blocker severity; historical record (`superseded/`, `research/`) exempt via the standard content-drift exemption.
- Role-first selection and they/them reference as review-enforced convention wherever the mechanical gate cannot reach (role fit, pronouns, non-attribution mentions).

## What's Out of Scope

- Mechanical role-match enforcement (requires semantic judgement of the stated role against `role_hints`).
- Pronoun linting (they/them violations are a review concern; a pronoun regex cannot see referents).
- Real-name/PII detection generally — that is the itd-74 banlist family's job.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline._

- **Given** an intent whose press-release quote is attributed to a name not in `personas.json`, **when** record-lint runs, **then** a `persona_registry` blocker finding names the file, line, and offending name.
- **Given** a quote attributed to a registry persona, **when** record-lint runs, **then** no persona finding is raised for that line.
- **Given** a file under an exempt path (historical record), **when** record-lint runs, **then** persona attributions in it are not flagged.
- **Given** the registry gains a persona (continuing the alphabetical sequence), **when** a new intent quotes that persona, **then** the lint passes without any lint-config change — the roster file is the only thing edited.

## References

- Registry: `.abcd/development/personas.json` (schema_version 2 — role-first selection convention).
- Gate: `persona_registry` rule in `.abcd/record-lint.json`, implemented in `internal/core/lint/persona.go`.
- Lineage: promoted from the `principles/` layer per the promotion path recorded in `principles/README.md`; decision lines of 2026-07-08: [dcn-2607080000000002](../../../work/decisions/dcn-2607080000000002-personas-in-any-scenario-are-always.md), [dcn-2607080000000003](../../../work/decisions/dcn-2607080000000003-consume-model-interview-spec-id-is.md), [dcn-2607080000000004](../../../work/decisions/dcn-2607080000000004-persona-registry-lint-shipped.md), [dcn-2607080000000005](../../../work/decisions/dcn-2607080000000005-persona-ssots-reconciled-personas-json.md).
- Convention prior art: controlled-vocabulary field binding (DITA subject-scheme pattern) — see `ACKNOWLEDGEMENTS.md`.
