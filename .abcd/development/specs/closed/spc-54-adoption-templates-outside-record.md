---
id: spc-54
slug: adoption-templates-outside-record
intent: itd-162
---
# adoption-templates-outside-record

## Summary

Makes prepare-this-repo's adopt phase self-contained in the abcd record. Phase 3
references two templates at a machine-local path
(`~/ABCDevelopment/.agents/templates/`), so a fresh clone of abcd-cli onboarding
a repo would not have them and the adoption silently degrades against
loud-staging. This spec resolves every adopt-phase asset from the committed
record or the embedded binary — never a `~` path — and adds an onboarding
self-containment check that no machine-local reference remains.

## Scope

In:

- `commands/prepare-this-repo.md` — the two machine-local references at lines 173
  (Phase 3 pre-commit config) and 177 (Phase 3 `prepare-commit-msg` hook),
  rewritten to resolve from the binary/record.
- A net-new `prepare-commit-msg` template embedded under
  `internal/core/ahoy/defaults/` via `//go:embed` in
  `internal/core/ahoy/banlist_scaffold.go` (no such template exists yet).
- An onboarding self-containment check asserting no `~`/machine-local path
  reference remains in the onboarding artefacts.
- Tests modelled on `internal/core/ahoy/banlist_scaffold_test.go`.

Out:

- AGENTS.md, DECISIONS.md and NEXT.md are described *inline* as prose in
  prepare-this-repo.md (the working-conventions section, lines 181–207), not as
  machine-local file-template references — no `~` path there, so nothing to move.
- No copying of abcd-record content downstream (prepare-this-repo.md's
  never-commit-downstream note forbids it); the embedded templates are generic
  scaffolding, not record files.

## Approach

**The two `~` references.** `commands/prepare-this-repo.md` invokes the binary as
`"${CLAUDE_PLUGIN_ROOT}/abcd"` (lines 82/142/161) but, at Phase 3 steps 5 and 6,
reaches for `~/ABCDevelopment/.agents/templates/pre-commit-config.yaml` (line 173)
and a `prepare-commit-msg` hook under the same path (line 177), both "if present"
— the silent-degradation source: absent on a fresh clone, the step does nothing
and loud-staging is defeated.

**Reuse the existing embed precedent.** `internal/core/ahoy/banlist_scaffold.go`
already `//go:embed`s `defaults/pre-commit` (line 65) and `defaults/pre-merge-commit`
(68), with the rationale (lines 59–63) stated verbatim: embedded "so the binary
is self-contained … so scaffolding cannot depend on a plugin-root file that a
broken install left behind." The pre-commit config the Phase-3 line reaches for is
thus already available embedded. The change:

1. Rewrite prepare-this-repo.md lines 173 and 177 to scaffold via the binary
   (`"${CLAUDE_PLUGIN_ROOT}/abcd" …`) so the templates resolve from the embedded
   defaults, never from `~`.
2. Add a `prepare-commit-msg` template under `internal/core/ahoy/defaults/` and
   `//go:embed` it beside the guard hooks in `banlist_scaffold.go` (it does not
   exist yet), so Phase-3 step 6 resolves it from the binary.

**Self-containment check.** No existing check scans for machine-local `~` path
references. Add an onboarding self-containment gate that scans the adopt-phase
artefacts (`commands/prepare-this-repo.md` and the assets it applies) and fails on
any `~`/`ABCDevelopment/.agents/templates` reference — asserting every asset the
adopt phase applies resolves from within the abcd record or the binary. The gate
is asserted for humans in prepare-this-repo.md's "Definition of done" (lines
215–227) and enforced by the net-new check.

## How it satisfies each acceptance criterion

- *A machine without `~/ABCDevelopment/.agents/templates/` still completes
  adoption* — the templates now resolve from the embedded binary
  (`banlist_scaffold.go` embeds), so the adopt phase applies them regardless of
  the home directory. Test: run the adopt scaffolding with `HOME` pointed at an
  empty dir and assert the templates are applied (mirrors
  `banlist_scaffold_test.go`).
- *Every adopt-phase asset resolves from the record or the binary, never a
  machine-local path* — the pre-commit config and `prepare-commit-msg` are both
  embedded; AGENTS.md/DECISIONS.md/NEXT.md are inline prose. Test: assert each
  named asset has an in-record or embedded source and no `~` resolution.
- *The self-containment check finds no `~` machine-local templates path remaining*
  — the new gate scans prepare-this-repo and its assets. Test: the gate passes on
  the rewritten file and fails on a fixture that reintroduces a `~` reference.

## Decisions

Embed the templates in the binary rather than commit them as loose record files,
following the established `banlist_scaffold.go` pattern: an embedded asset cannot
be defeated by a broken install that left the plugin root without a file, which is
exactly the failure mode a `~` path already exhibits. The `prepare-commit-msg`
template is net-new because none exists in the tree; the pre-commit config reuses
the already-embedded default. This keeps the adopt phase self-contained without
copying any abcd-record content downstream, honouring the file's own
never-commit-downstream rule.
