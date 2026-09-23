---
id: spc-30
slug: the-lifeboat-verdict-says-what-it-is-disembark-oracle-become
intent: itd-125
---
# the-lifeboat-verdict-says-what-it-is-disembark-oracle-become

## Summary

Delivers the third rename: `disembark oracle` → `disembark review`,
`--oracle-json` → `--review-json`, the lifeboat synthesis artefact
`audit/oracle-<manifest12>.*` → `review/review-<manifest12>.*`, and the
`lifeboat-oracle` agent → `lifeboat-reviewer`. Reverses adr-40 §5 per the
maintainer's 2026-08-16 ruling (recorded as a dated in-place amendment in
adr-40 and a `DECISIONS.md` line, landed with the planning records). Clean
break, no aliases. Ships only after spc-27's extended `surface_coverage` is
armed.

## Scope — the live sweep (verified inventory, 2026-08-16)

Code and surface:

- `internal/core/lifeboat/synthesis_oracle.go` (+ `_test.go`) →
  `synthesis_review.go`; `AuditOracle` → `ReviewLifeboat` (or `Review`),
  `OracleResult` → `ReviewResult`; `synthesis_types.go` and
  `embark_types.go` references follow. The dual-mode contract, enum gate,
  cite-or-be-dropped, and attestation stamping are behaviour-frozen — the
  moved tests assert byte-identical verdict logic.
- Artefact writer: output directory `audit/` → `review/`, basename
  `oracle-<manifest12>` → `review-<manifest12>`; still excluded from
  `manifest_sha256`. **Clean-replacement across the rename**: on a re-run,
  after writing `review/review-<manifest12>.json|.md`, remove
  `audit/oracle-<manifest12>.json|.md` for the same manifest12 when present
  (and remove `audit/` if left empty); never touch other manifests' files.
- `internal/surface/cli/cli.go` + `disembark_synthesis_test.go` — the
  `oracle` sub-command → `review`, flag `--oracle-json` → `--review-json`,
  help text, JSON envelope keys carrying `oracle`.
- `agents/lifeboat-oracle.md` → `agents/lifeboat-reviewer.md` (name,
  frontmatter, prompt, canary fixture dir `agents/lifeboat-oracle/` →
  `agents/lifeboat-reviewer/`); `agents/README.md` roster;
  `agents/CHANGELOG.md` gets a new entry (append, never rewrite).
- `commands/disembark.md` — the "Oracle audit" section → "Review (content
  fidelity + verdict)".
- `docs/reference/cli/commands.md` — regenerate.

Record (current-state surfaces only):

- Brief: `01-product/01-press-release.md`, `04-surfaces/02-disembark.md`,
  `04-surfaces/03-embark.md`, `05-internals/01-agents.md` (the "oracle
  audit" prose → "review"; the seam prose keeps `oracle` where it names
  adr-25's seam), `02-adapters.md`, `03-configuration.md`,
  `04-universal-patterns.md`, `05-prompt-quality.md`,
  `06-delivery/01-build-sequence.md`, `02-verification-matrix.md`.
- `02-constraints/04-naming.md` — task-class enum row `oracle_review`
  re-checked: the token stays (it honestly names review work reaching a
  model through the oracle seam); the row's source prose updates.
- The `disembark` sub-verb table row (spc-27 format) flips in the same
  change.
- `CHANGELOG.md` — **breaking** entry (verb, flag, and artefact path all
  named).

**Not swept**: historical/dated records; adr-25 (the seam is untouched —
`oracle` remains the seam's name and `/abcd:oracle ask` remains the reserved
seam surface); voyage/history ledger entries already written.

## Approach

1. **Precondition**: spc-27 armed, else STOP.
2. Rename the core entrypoint and types, watching moved tests fail under
   old names first; add the old-artefact removal step with tests: (a) fresh
   pack → only `review/` written; (b) old pack with `audit/oracle-*` →
   re-run writes `review/`, removes the stale pair, `audit/` pruned when
   empty; (c) mixed manifests → only the matching manifest's files removed.
3. Rename the CLI sub-command and flag; surface tests pin `disembark
   oracle` and `--oracle-json` as unknown.
4. Agent + canary fixture rename; injection-canary test paths follow.
5. Sweep prose; flip the sub-verb row; regenerate `surface.json` and the
   CLI reference. `record-lint`, `docs-lint`, drift tests, `make preflight`
   exit 0. PR classifies the residual `oracle` grep (seam uses stay).

## Acceptance-criteria mapping

- *New spellings, old unknown* — step 3's pins.
- *Behaviour frozen* — step 2's moved tests (deterministic mapping table,
  delegated gates, attestation stamping, exit codes).
- *Artefact move + clean replacement* — step 2's three artefact tests.
- *Sweep + token re-check + boundary* — steps 4–5; the naming-registry row.
- *Reversal recorded* — the adr-40 amendment and DECISIONS line land with
  the planning records (this PR), before any build.
- *Armed-gate ordering* — step 1.
- *Breaking CHANGELOG* — Scope.

Every behaviour change lands with a test watched fail before the change.
