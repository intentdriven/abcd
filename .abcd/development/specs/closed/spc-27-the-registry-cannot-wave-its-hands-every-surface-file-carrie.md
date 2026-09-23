---
id: spc-27
slug: the-registry-cannot-wave-its-hands-every-surface-file-carrie
intent: itd-122
---
# the-registry-cannot-wave-its-hands-every-surface-file-carrie

## Summary

Delivers per-surface sub-verb tables (bucket + existence per verb) across
`04-surfaces/`, and extends the `surface_coverage` record-lint rule to check
every table row against the committed command-tree snapshot in both
directions. This is adr-40's detector (decision 6), the fix for iss-246, and
the arming step that must land before any rename intent ships.

## Scope

- `internal/core/lint/lint.go` — extend `checkSurfaceCoverage` (lint.go, the
  `surfaceRow` parser and the three existing invariants) with the sub-verb
  pass; new parsing/check helpers may split into `internal/core/lint/
  subverbs.go`.
- `internal/core/lint/config.go` — rule config gains: the snapshot path
  (default `.abcd/development/release/surface.json`), the host-delegated
  surface list (`consult`, `ingest`, `prepare-this-repo`), and the
  operator-internal verb list (`spec`, `rules`, `hook`, `completion`) — all
  explicit config beside the existing `BareCommand`, never hard-coded silent
  skips.
- `.abcd/development/brief/04-surfaces/*.md` (all twenty) — the population
  pass: one sub-verb table per file, fixed format (below), rows reflecting
  current pre-rename names.
- `.abcd/development/brief/04-surfaces/README.md` — the prose describing the
  machine-checked column extends to the sub-verb grain.
- `.abcd/development/brief/02-constraints/04-naming.md` — register the bucket
  enum `{lint, review, audit, gate}` under Reserved vocabulary (source:
  adr-40 + this spec), per the VR001 discipline.
- `CHANGELOG.md` — additive entry.

Out of scope: any rename (intents 5–7); any change to the surface-grain
`Status` semantics (stays two-valued); any new `surface.json` content —
the snapshot generator (`cmd/abcd-gen-surface`, `cli.GenerateSurface`) is
consumed, not modified, unless it omits sub-command data the check needs, in
which case extending its emitted shape is in scope but its drift-test
contract (byte-identical regeneration) must be preserved.

## Approach

**Table format** (parse target, one per surface file):

```markdown
## Sub-verbs

| Verb | Bucket | Status |
|---|---|---|
| `probe` | — | shipped |
| `oracle` | review | shipped |
```

Anchored by the exact `## Sub-verbs` heading. `Verb` is the sub-command name
as registered (code-span, one token; nested sub-verbs space-separated, e.g.
`review ingest`). `Bucket` is one of `lint` / `review` / `audit` / `gate`, or
`—` for a non-assessment verb. `Status` is `shipped` / `staged`. Any other
value in either column is a lint finding (mirrors the existing unknown-status
finding).

**The check**, per surface with a table:

1. Load `surface.json`; build the registered sub-command set for the
   surface's top-level verb (the snapshot is drift-checked against the cobra
   tree by `internal/surface/cli/surface_test.go`, so committed bytes are a
   trustworthy proxy for the tree — no import of the surface package from
   core/lint).
2. `shipped` row → its verb must be in the registered set; `staged` row → it
   must not; registered sub-command → a row must exist. Each violation is one
   finding at the row's (or file's) line, rule id `surface_coverage`.
3. Exemptions, all from config: host-delegated surfaces skip step 1–2's cobra
   comparison (their rows are format-checked only); operator-internal verbs
   are absent from the registry by design and excluded from the reverse
   check; the `bare_command` exemption stays as-is. Cobra's auto-added
   `help` sub-command is excluded structurally.
4. A surface file with **no** `## Sub-verbs` heading: fails the lint if the
   snapshot registers sub-commands for its verb (blindness is what this
   intent removes); passes if the verb has none.

**Population pass** (same change, binding rulings):

- Buckets follow adr-40's comparison test. Pre-ruled by the maintainer
  (2026-08-16, binding): `identity` bare/render = **audit**; the launch
  changelog guardrail = **gate**; `guard check` = **gate**. `intent ready` =
  gate (adr-40); `docs lint` = lint; `intent review` and `disembark oracle`
  keep their *current* names in rows (the renames update code and table
  together, gated by this check).
- Any other genuinely ambiguous bucket is a STOP for an unattended run —
  record the ambiguity, do not guess into the closed list.

## Acceptance-criteria mapping

- *Tables present, two facts per verb* — population pass + check step 4;
  test: a fixture surface file without a table for a sub-command-bearing verb
  fails.
- *Both-direction check* — step 2; tests: shipped-but-unregistered,
  staged-but-registered, registered-but-rowless, each one finding.
- *Explicit exemptions* — step 3 + config; tests: host-delegated surface
  passes without cobra backing; an unlisted silent skip is impossible (config
  absence means the check applies).
- *Same-change population, pre-rename names* — the twenty files land in this
  spec's change; `make record-lint` exit 0 on the branch is the proof.
- *Pre-ruled buckets* — recorded above; the population pass applies them.
- *Enum registered* — the `04-naming.md` Reserved-vocabulary row.
- *No `partial`* — no code change; the two-valued parse is unchanged and the
  README prose states the ruling.
- *Docs sweep* — Scope's file list; `record-lint` / `docs-lint` clean.

Every new behaviour lands with a test watched fail before the change.
