---
id: spc-25
slug: a-resolved-issue-points-at-what-fixed-it-abcd-capture-resolv
intent: itd-120
---
# a-resolved-issue-points-at-what-fixed-it-abcd-capture-resolv

## Summary

Delivers the `--intent` / `--spec` / `--commit` provenance flags on
`abcd capture resolve`, writing the structured `resolved_by` object the schema
already parses. Closes the `resolved_by` half of iss-245. Provenance is
optional: a flagless resolve stays byte-identical to today.

## Scope

- `internal/core/capture/workflow.go` — `ResolveRequest` gains
  `ByIntent, BySpec, ByCommit string`; `Resolve` (workflow.go, currently
  writing only `resolution` + `impact`) validates and threads the object into
  the existing `transition` write.
- `internal/core/capture/serialize.go` — a deterministic nested-map writer
  (`setMapField`) beside `setScalarField`: `resolved_by` is an object, and the
  current `kv` extras write scalars only. Fixed member order
  `intent, spec, commit`; omitted members are omitted, never empty strings.
- `internal/core/capture/promote.go` / a small shared helper — the record-id
  existence probe (see Approach); place it wherever spc-24's implementation
  put its link-mode check so both verbs share one probe.
- `internal/surface/cli/cli.go` — three flags on the `resolve` sub-command in
  `newCaptureCommand`; JSON result carries the written `resolved_by`.
- Docs sweep: `commands/capture.md` (resolve section + `argument-hint`),
  `.abcd/work/issues/README.md` (the `resolved_by` field is now written by the
  verb), `CHANGELOG.md` (additive entry).

Out of scope: backfilling already-resolved issues (ruled out at the grill —
resolve still transitions open issues only); any change to `wontfix`; any
change to `promoted_to` (itd-119/spc-24).

## Approach

**Validation, before any write.**

- `--intent`: must match `^itd-[0-9]+$` **and** exist in the intent store —
  a filename probe over `.abcd/development/intents/{drafts,planned,shipped,
  disciplines,superseded}/` for `<id>.md` or `<id>-*.md`, the same match rule
  `findIssue` uses (exact id or id-prefixed slug, extension `.md`).
- `--spec`: `^spc-[0-9]+$` and the same probe over
  `.abcd/development/specs/{open,closed}/`.
- `--commit`: shape only, `^[0-9a-f]{7,64}$` — no git invocation; a sha's
  home may be the remote and shallow/rebased states must not refuse a
  legitimate resolution.

Any failure refuses the whole transition with a diagnostic naming the failing
flag; nothing is written and the issue stays in `open/`.

**Write.** Extend `transition`'s extras so `Resolve` passes the validated
object; `setMapField` renders

```yaml
resolved_by:
  intent: "itd-119"
  spec: "spc-24"
```

with only the supplied members, quoted like the ledger's other scalars, in the
one atomic checksum-guarded write `transition` already performs under
`withLedgerLock`. No flags → no `resolved_by` key (assert byte-identical
output to the pre-change writer in a regression test). The read side
(`validate.go`'s `resolved_by` object check and the `ResolvedBy` parse) is
already correct and unchanged — round-trip tests prove write→read fidelity.

**CLI.** `capture resolve <iss-N> "<note>" --impact <…> [--intent itd-N]
[--spec spc-N] [--commit sha] --json`. The human rendering names what was
stamped; JSON adds a `resolved_by` object mirroring the frontmatter. Fault
conventions mirror the existing resolve errors (non-zero exit, diagnostic,
nothing written).

## Acceptance-criteria mapping

- *Provenance written in the same atomic transition* — the `transition`
  extension; test: resolve with all three flags, assert frontmatter object,
  member order, and single-write atomicity (file appears in `resolved/` with
  the object; no intermediate state).
- *Existence for ids, shape for sha; refusal writes nothing* — the validation
  block; tests: unknown `itd-N`/`spc-N`, malformed sha, each asserting the
  issue remains in `open/` byte-identical.
- *Flagless resolve unchanged* — regression test asserting byte-identical
  output against the current writer.
- *`--json` contract* — surface test asserting the `resolved_by` members in
  the result.
- *`wontfix` untouched* — no code path change; existing tests stand guard.
- *Docs sweep* — file list in Scope; `docs-lint` / `record-lint` stay clean.

Every new behaviour lands with a test watched fail before the change.
