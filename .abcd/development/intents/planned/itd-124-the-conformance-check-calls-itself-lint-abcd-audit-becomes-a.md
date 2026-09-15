---
id: itd-124
spec_id: spc-29
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
impact: breaking
slug: the-conformance-check-calls-itself-lint-abcd-audit-becomes-a
---

# The Conformance Check Calls Itself Lint

## Press Release

The conformance check calls itself what it is. `abcd lint` checks a repo
against the working conventions — deterministic rules, severities, exit codes
— and the word `audit` goes back to meaning "did we do what we said":
`/abcd:audit` returns to its reserved seat for the hash-chain fidelity checks.
"When a verb says lint I know it's checking form — no model, no judgement.
When one says audit I know it's checking a promise. I stopped having to
remember which one lied," says Bob, staff engineer.

## Why This Matters

itd-85 shipped deterministic rule-checking under the one name
`02-constraints/04-naming.md` reserves for itd-16's formal verification. Per
adr-40 the act is a lint, and the maintainer ruled the replacement name
`abcd lint` (2026-08-16, planning question 1): it matches the existing
`record-lint` / `docs-lint` / `lint-reviews` family, with `abcd docs lint`
read as the same word at a narrower scope. Clean break, no aliases —
pre-1.0.0, `--impact breaking` drives version derivation, users re-download.
About 57 references carry the current name; the scoped sweep is smaller (the
word "audit" in its family-2 sense — the intent audit — and third-party
senses never move).

## Scope Conditions

None stated.

## Acceptance Criteria

> _BDD format, per the itd-1 discipline. Walked and confirmed by the
> maintainer, 2026-08-16._

- **Given** the rename, **then** `abcd lint` replaces `abcd audit` with
  identical behaviour, flags, and the tri-state exit contract, and
  `abcd audit` fails as an unknown command — no alias, no shim.
- **Given** the plugin surface, **then** `commands/lint.md` replaces
  `commands/audit.md` (`/abcd:lint`), the `04-surfaces` registry row and
  surface file move, and `/abcd:audit` stands reserved for itd-16 again —
  its two `04-naming.md` listings intact.
- **Given** the sweep boundary, **then** historical and dated records keep
  the old name; the brief, commands, code, tests, `docs/`, and rules-loader
  references move. The scoped patterns are `abcd audit`, `/abcd:audit`, and
  the itd-85 surface's identifiers — never the intent audit or third-party
  senses of the word.
- **Given** anything abcd writes into other repos (managed-repo scaffolding,
  `prepare-this-repo` instructions), **then** it emits the new verb, and the
  breaking CHANGELOG entry names the re-download/re-scaffold step for
  managed repos.
- **Given** itd-122's extended `surface_coverage` armed, **then** the
  registry and sub-verb rows move in the same change — `record-lint` exit 0
  proves the migration complete; landing before the check is armed is
  forbidden.
- **Given** the task-class tokens, **then** conformance work carries `lint`,
  never `audit`, and the enum rows in `04-naming.md` are re-checked in the
  same change (adr-40's consequence).
- **Given** the release record, **then** the CHANGELOG entry is breaking.

## SOTA

Same family as itd-123: pre-1.0.0 breaking rename, no aliases (adr-40;
iss-171 precedent). Package ruling after adversarial review (2026-08-16):
the conformance code moves to `internal/core/repolint` — merging the two
lint engines was reviewed and deliberately deferred to iss-251, a future
consolidation intent, never smuggled into a rename. **Chosen path: bespoke
sweep**, proved complete by the armed itd-122 gate. No new dependency.

## Open Questions

_None gating. The engine-consolidation question is captured as iss-251._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
