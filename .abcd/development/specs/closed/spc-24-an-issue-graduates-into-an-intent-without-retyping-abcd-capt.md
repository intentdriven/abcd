---
id: spc-24
slug: an-issue-graduates-into-an-intent-without-retyping-abcd-capt
intent: itd-119
---
# an-issue-graduates-into-an-intent-without-retyping-abcd-capt

## Summary

Delivers `abcd capture promote <iss-N> [--intent <itd-N>] --json`: the native
verb for step 2 of the record walk. Default mode mints an intent draft from the
issue and stamps the issue's `promoted_to` in the same invocation; `--intent`
is the stamp-only mode that links an existing draft instead of minting. Closes
the `promoted_to` half of iss-245.

## Scope

- `internal/core/capture/promote.go` (new) — `Promote(req PromoteRequest)
  (PromoteResult, error)`, transport-agnostic, plus its typed request/result.
- `internal/core/intent/create.go` — factor the mint path so promote can reuse
  it with an explicit slug and seed body (one-canonical-primitive; see
  Approach).
- `internal/surface/cli/cli.go` — `promote` sub-command inside
  `newCaptureCommand` (cli.go:1841).
- Docs sweep: `commands/capture.md` (replace the "skill-orchestrated, not a
  binary sub-verb" promote paragraph and extend the `argument-hint`),
  `.abcd/development/brief/02-constraints/04-naming.md` (drop the
  design-target marker on `capture promote`),
  `.abcd/development/brief/04-surfaces/` capture entry (promote is shipped),
  `CHANGELOG.md` (additive entry).

Out of scope: any change to `resolve`/`resolved_by` (sibling intent), any
sub-verb-table machinery, any change to the planning interview.

## Approach

**Core (`capture.Promote`).** `PromoteRequest{RepoRoot, IssuesRoot, ID,
LinkIntent string}`. Flow:

1. `findIssue` (alloc.go:348) locates the issue in **any** status folder —
   promotion is orthogonal to fix-status; the file never moves.
2. Read via `readWithChecksum`; if the parsed issue already carries
   `promoted_to`, refuse with an error naming the existing `itd-N` (the CLI
   surfaces it as a normal verb error, mirroring `resolve` conflicts).
3. **Mint mode** (`LinkIntent == ""`): call the factored intent-create core
   with slug = the issue's slug, title = the issue's first body line, impact
   unset (a promoted draft is "not judged yet"), and a seed body that carries
   the standard placeholder Press Release section plus a Why This Matters of
   the form "Graduated from `iss-N`: <issue first line>. Read that issue
   record for the source observation." — a by-id pointer, **never** a copy of
   the issue body (SSOT). The minted draft's frontmatter gains
   `promoted_from: iss-N` (string, optional field; the intent reader's
   `Intent` struct gains `PromotedFrom` parsed leniently — absent on every
   existing record).
4. **Stamp** (both modes): under `withLedgerLock` (alloc.go:86), re-find and
   checksum-re-read the issue, `setScalarField(content, "promoted_to", itdID)`
   (serialize.go:114), and write back **in place** to the same path
   (atomic-rename write like `commitTransition`, minus the move). In link mode
   the `itd-N` is first verified to exist in the intent store (any bucket);
   unknown → structural fault, nothing written.
5. **Ordering + residue contract**: mint first, stamp second. A failure after
   the mint returns an error carrying the orphan draft's path and the remedy
   string naming `capture promote <iss-N> --intent <itd-N>`. No cross-store
   lock is attempted; the ledger lock alone guards the stamp, exactly as
   `transition` does.

`PromoteResult{IssueID, IssueStatus, IssuePath, IntentID, IntentPath, Linked
bool}` — repo-relative paths, JSON-rendered by the CLI with the shared
marshaller.

**Intent-create factoring.** Extract the body of `CreateFromText`
(create.go:41) into an unexported `createDraft(repoRoot string, opts
draftOptions) (Intent, string, error)` where `draftOptions{Slug, Title,
SeedBody, Impact, PromotedFrom string}`; `CreateFromText` becomes a thin
wrapper deriving slug/title from text (`deriveIntentSlug`, `titleLine`) with
an empty `PromotedFrom`. Both paths share `nextIntentID` and
`withIntentMintLock`, so promote can never mint outside the id lock.
`capture` imports `intent` (cycle-free; intent imports only changelog,
recordid, spec, frontmatter, fsutil).

**CLI.** `capture promote <iss-N>` with `--intent` and the standard `--json`;
human rendering states what was minted or linked and both paths. Faults follow
the existing capture verb conventions (invalid/unknown id → non-zero exit with
diagnostic, nothing written).

## Acceptance-criteria mapping

- *Mint + stamp in one invocation* — steps 3–4; test: promote an open issue,
  assert draft exists under `drafts/` with reused slug, placeholder press
  release, by-id pointer body, `promoted_from`; issue frontmatter gains
  `promoted_to` with the minted id.
- *Any status, folder kept* — step 1; tests over `resolved/` and `wontfix/`
  issues assert stamp-in-place, no move.
- *Already-promoted refuses* — step 2; test asserts the error names the
  existing itd-N and no second draft appears.
- *Malformed/unknown id structural fault* — `findIssue` + link-mode existence
  check; tests assert non-zero exit, ledger and intent store byte-identical.
- *Mint-first residue + `--intent` repair* — step 5; test simulates stamp
  failure (unwritable ledger) and asserts the error carries the draft path and
  remedy; a follow-up `--intent` run completes the link. Test that link mode
  never mints.
- *Two-sided edge* — `promoted_from` in the draft; parse-back test via the
  intent reader.
- *`--json` contract* — surface test asserting the result fields.
- *Docs sweep* — covered by the file list in Scope; `docs-lint` and
  `record-lint` stay clean.

Every new behaviour lands with a test watched fail before the change.
