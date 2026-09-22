---
id: spc-2609211859391533
slug: three-intent-kinds
intent: itd-34
origin: researcher-authored
production_mode: hand-written
---
# three-intent-kinds

## Summary

The design record for itd-34's remainder, from the product thinker's
interview of 2026-09-21 (decisions 1 to 4): the bundle command and the
`reclassify` verb. The kinds, the shelves and the two-way supersession link
already exist and are read, not built.

## Scope

1. **The bundle command**: `abcd intent plan itd-A itd-B [itd-C…]` takes
   `--bundle <name>` (asked for by the plugin page, refused absent on the
   CLI), refuses a member that names another in `blocked_by`, naming the edge, mints ONE
   spec whose frontmatter lists every intent (`intents: [itd-A, itd-B]`
   beside the existing `intent:` key naming the first), stamps
   `kind: bundle-member` and `bundle: <name>` on each, links each `spec_id`,
   stamps scope conditions on each, and moves all together under the intent
   store's mint lock; any refusal leaves nothing moved (criterion 1).
2. **The close-hook**: `spec close` on a spec naming several intents ships
   every member whose `bundle:` matches, with the same impact rule per
   member (criterion 2).
3. **`abcd intent reclassify <itd-N> --kind <standalone|bundle-member
   [--bundle <name>]|superseded --by <id> --reason "…">`**: one write under
   the lock that sets `kind`, moves the shelf where the kind implies one
   (superseded → `superseded/`), writes `superseded_by` on the record and
   appends to `supersedes` on the successor (an intent or an ADR), appends a
   `reclassification_history` entry (from, to, date, reason), and prints
   the paths moved; `--kind discipline` on a shipped intent is refused with
   the remedy "file a discipline that supersedes it" (criterion 3).
4. **The survivor rule**: superseding a bundle member leaves the other as
   `bundle-member` and appends a line to its `reclassification_history`
   stating the bundle now has one member (criterion 4).
5. **The lint**: `record_schema` gains two checks: a planned or shipped
   intent's `kind` matches its shelf (`discipline` on `disciplines/`, the
   other two on `planned/` or `shipped/`), and every `bundle-member`'s
   `bundle:` names a bundle at least one other record names or a history
   line saying it is one (criterion 5).

## Out of scope

- Reclassifying a shipped intent to `discipline` (decision 4).
- Dissolving a bundle (decision 3).
- Any change to the disciplines' template or the supersession note's prose.

## Approach

The bundle command extends `intent.Plan` to take a list, minting once and
stamping per member inside the existing `withIntentMintLock`; the close-hook
extends the spec store's `Reconcile` to iterate the spec's `intents` list.
`reclassify` is a new `internal/core/intent` entry point over the same lock
and the record store's frontmatter writer, reusing the supersession-note
shape the superseded records already carry. The lint rows go beside the
one-way-supersession check that already exists. CLI and `commands/intent.md`
gain the two forms; the brief's intent chapter says what they do.

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 bundle plan, named, refused across phases | scope 1 |
| 2 members ship together | scope 2 |
| 3 reclassify in one write, shipped-to-discipline refused | scope 3 |
| 4 survivor stays and says so | scope 4 |
| 5 lint on kind and bundle | scope 5 |
