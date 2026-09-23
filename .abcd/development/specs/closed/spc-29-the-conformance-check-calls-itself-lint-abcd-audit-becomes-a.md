---
id: spc-29
slug: the-conformance-check-calls-itself-lint-abcd-audit-becomes-a
intent: itd-124
---
# the-conformance-check-calls-itself-lint-abcd-audit-becomes-a

## Summary

Delivers adr-40's second named rename, with the name the maintainer ruled in
planning: `abcd audit` → `abcd lint`, `/abcd:audit` → `/abcd:lint`, and the
reserved `/abcd:audit` seat returned to itd-16. Clean break, no aliases.
Ships only after spc-27's extended `surface_coverage` is armed. Package
ruling: the conformance core moves to `internal/core/repolint` (adversarial
review outcome; the engine merge is iss-251, out of scope).

## Scope — the live sweep (verified inventory, 2026-08-16)

Code and surface:

- `internal/core/audit/` → `internal/core/repolint/` (package `repolint`);
  exported API unchanged in shape (`Rule`, `Evaluate`, `DefaultRules`,
  `Context`, tri-state exit mapping). The `rule_docs.go` wrap of
  `core/lint` stays as-is.
- `internal/surface/cli/audit.go` (+ `audit_surface_test.go`,
  `audit_render_test.go`) → `lint.go` etc.; the `audit` verb in `cli.go` →
  `lint`, help text, JSON envelope key names carrying `audit`.
- `internal/surface/cli/identity.go` and `commands/identity.md` — the
  identity-positioning rule references (`runs on every audit` prose).
- `commands/audit.md` → `commands/lint.md`; `commands/prepare-this-repo.md`
  (the audit step it orchestrates).
- Tests referencing the verb string: `internal/termsafe/termsafe_test.go`,
  `internal/core/ahoy/gitignore_test.go`,
  `internal/core/lint/deliverystate_test.go`,
  `internal/core/launch/commandladder_test.go`,
  `internal/core/positioning/containment_test.go`,
  `internal/core/intent/create_test.go`.
- `docs/reference/cli/commands.md` — regenerate.

Record (current-state surfaces only):

- Brief: `01-product/03-mental-model.md`, `02-constraints/04-naming.md`
  (the metaphor-exemption and reserved rows re-point; `/abcd:audit` stays
  listed as reserved-for-itd-16 in BOTH its listings; task-class enum rows
  re-checked — conformance carries `lint`), `04-surfaces/README.md` row,
  `04-surfaces/16-audit.md` → `16-lint.md` (ordinal kept), `05-intent.md`,
  `08-abcd.md`, `15-prepare-this-repo.md`, `19-identity.md`,
  `05-internals/01-agents.md`, `04-universal-patterns.md`, `08-skills.md`,
  `06-delivery/03-out-of-scope.md`.
- The sub-verb table rows (spc-27 format) move in the same change.
- `CHANGELOG.md` — **breaking** entry that names the managed-repo step:
  re-download the binary and re-run scaffolding/`prepare-this-repo` where a
  managed repo's instructions referenced `abcd audit`.

**Not swept**: historical/dated records (ADRs, dated notes and plans,
resolved issues, shipped/superseded intents, `DECISIONS.md`), the word
"audit" in its family-2 sense (`intent audit`, itd-16's reserved surface,
oracle/lifeboat prose owned by itd-125), and third-party senses (zizmor's
workflow audit, `history_audit`). Scoped patterns: `abcd audit`,
`/abcd:audit`, `core/audit`, and the itd-85 surface identifiers.

## Approach

1. **Precondition**: spc-27's sub-verb check armed, else STOP (plan
   ordering).
2. Move the package (`git mv internal/core/audit internal/core/repolint`,
   package clause + importer updates), watching renamed tests fail under the
   old wiring first.
3. Rename the CLI verb and surface files; `abcd audit` must fail as unknown
   (surface test pins it); the tri-state exit contract of `abcd lint` is
   asserted unchanged by the existing render/surface tests moved over.
4. Sweep the brief and command surfaces; flip registry + sub-verb rows;
   `16-audit.md` renames with its ordinal kept (`16-lint.md`) so the
   registry file-link column stays dense.
5. Regenerate `surface.json` and the CLI reference; `record-lint`,
   `docs-lint`, surface drift test, and `make preflight` exit 0 — the proof
   of completeness. PR classifies the residual `audit` grep into the
   not-swept classes.

## Acceptance-criteria mapping

- *Identical behaviour under `lint`, `audit` unknown* — step 3's moved tests
  + the unknown-command pin.
- *Surface file moves; reservation intact* — step 4; `docs-lint` link check
  + a grep asserting both `04-naming.md` listings still name `/abcd:audit`
  reserved for itd-16.
- *Sweep boundary honoured* — step 5's classified residual.
- *Managed-repo emissions* — `prepare-this-repo` + scaffold sweep; CHANGELOG
  names the re-scaffold step.
- *Armed-gate ordering* — step 1.
- *Token re-check* — the `04-naming.md` enum rows; the binary's task-class
  cross-check test updated in the same commit.
- *Breaking CHANGELOG* — Scope.

Every behaviour change lands with a test watched fail before the change.
