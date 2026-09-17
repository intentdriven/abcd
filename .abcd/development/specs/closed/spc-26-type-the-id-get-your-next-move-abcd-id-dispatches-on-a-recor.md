---
id: spc-26
slug: type-the-id-get-your-next-move-abcd-id-dispatches-on-a-recor
intent: itd-121
---
# type-the-id-get-your-next-move-abcd-id-dispatches-on-a-recor

## Summary

Delivers `abcd <id>`: a regex-gated positional on the namespace root that
locates a record by id — `iss-N`, `itd-N`, `spc-N`, `adr-N` — and reports what
it is, its links, and the concrete next move for its lifecycle state. Strictly
read-only. Makes the twelve-step walk observable without knowing the verb map.

## Scope

- `internal/core/record/` (new small package) — `Describe(repoRoot, id string)
  (Description, error)`: family detection, store lookup, and the next-move
  mapping. Transport-agnostic; imports `capture`, `intent`, `spec` read paths
  plus a thin adr reader (frontmatter id/status/title over
  `.abcd/development/decisions/adrs/`). Nothing imports it back (leaf package,
  no cycles).
- `internal/surface/cli/cli.go` — the root command (`NewRootCommand`,
  cli.go:54) currently declares `Args: cobra.NoArgs`; it changes to accept
  exactly one positional **iff** it matches `^(iss|itd|spc|adr)-[0-9]+$`
  (`cobra.ArbitraryArgs` + an explicit gate in `RunE`): a matching positional
  dispatches to `record.Describe`; anything else reproduces the current
  unknown-command error path byte-for-byte (assert in a golden test). `--json`
  on the root renders the `Description` with the shared marshaller.
- Docs sweep: the root surface page (`commands/abcd.md`) gains the positional
  ("bare answers *what can I do*; `abcd <id>` answers *what is this, and what
  is my next move*"), `.abcd/development/brief/04-surfaces/` root entry
  updated, `CHANGELOG.md` (additive entry).

Out of scope: dispatch for further families (plans, principles); any write or
transition; any change to the bare board.

## Approach

**Lookup per family** (each tolerant of any status folder):

- `iss-N` — `capture` read path (`findIssue` semantics + full parse).
- `itd-N` — the intent store readers across
  `drafts/planned/shipped/disciplines/superseded`; when planned, also run the
  read-only `intent ready` checks to pick between "write the spec body" and
  "implement".
- `spc-N` — the spec store (`open/closed`), plus its linked intent's ready
  state for the open-spec next move.
- `adr-N` — filename/front-matter probe over `decisions/adrs/` (files are
  `NNNN-slug.md` with `id: adr-N` frontmatter; probe by the filename's numeric
  ordinal, padding-agnostic, confirm via the frontmatter id).

**Description shape.** `Description{ID, Family, Title, Status, Path string,
Links map[string]string, NextMoves []string}` — links carry `spec_id` /
`intent` / `promoted_to` / `resolved_by.*` / `superseded_by` as present.

**The next-move mapping** is one Go table keyed on `(family, state)`:

| family, state | next move |
|---|---|
| itd, drafts | planning interview, then `abcd intent plan <id>` |
| itd, planned, spec body `_Draft:` | write the spec body (path shown) |
| itd, planned, ready | implement; when done `abcd spec close <spc>` |
| itd, shipped | none — audit state shown |
| itd, superseded | read the superseding record |
| iss, open, unpromoted | `abcd capture promote <id>` / `resolve` / `wontfix` |
| iss, open, promoted | see the intent it graduated into |
| iss, resolved / wontfix | none — trail shown (`resolved_by` when present) |
| spc, open (intent ready) | implement against the body, then `abcd spec close <id>` |
| spc, open (intent not ready) | the intent's failing checks, deferred there |
| spc, closed | none — done, linked intent shown |
| adr, any | none — decisions are read |

**Anti-drift test** (walked into the ACs): a test iterates every next-move
string that names a verb and asserts the verb path resolves to a registered
command in the cobra tree built by `NewRootCommand` — the rename intents then
break this test instead of shipping stale advice.

**Faults.** A shape-matching id found in no store → non-zero exit with a
diagnostic naming the id and the stores searched, nothing rendered as
success. Unknown positional shapes never reach the dispatcher.

## Acceptance-criteria mapping

- *Dispatch + read-only render* — `record.Describe` + root gate; per-family
  tests over fixture repos assert content and zero writes (store dirs
  byte-identical after the run).
- *Per-family next moves* — the mapping table; one test per row.
- *adr read-only* — probe + render test.
- *Unknown id fault* — test asserts exit code and diagnostic.
- *Non-matching positional unchanged* — golden test comparing the error
  output of `abcd nonsense` before/after the change.
- *`--json`* — surface test of the `Description` fields.
- *Tested mapping* — the anti-drift test above.
- *Docs sweep* — Scope's file list; `docs-lint` / `record-lint` clean.

Every new behaviour lands with a test watched fail before the change.
