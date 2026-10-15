# `/abcd:embargo` — Private Security Advisories, Kept Private Until Published

`/abcd:embargo` keeps a privately reported security advisory private until its
fix ships, then closes or publishes it with the person
(itd-2610112239022766, spc-2610120557432880). Its trust rule is
[adr-2610120558328615](../../decisions/adrs/2610120558328615-a-private-advisory-s-record-never-enters.md)
and brief invariant 22
([`02-constraints/03-invariants.md`](../02-constraints/03-invariants.md)): a
private advisory's record never enters the public ledger, its store is never
committed, nothing public carries an advisory id or a hint of one, and closing
or publishing an advisory takes one approval per act, bound to exactly what was
shown.

This chapter describes the first rung: intake, triage, closing, the outbound
check and publication. Running fix lanes, landing the fixes with redaction,
requesting a CVE, restoring redacted records and the broken embargo are later
rungs, each its own record.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only._

| Verb | Bucket | Status |
|---|---|---|
| `intake` | — | shipped |
| `triage` | — | shipped |
| `close` | — | shipped |
| `check` | gate | shipped |
| `publish` | — | shipped |

## The embargo store

The private records live in a worktree of the repository in the machine's
worktree store, `~/.abcd.noindex/worktrees/<root-sha>/embargo/`
(adr-2610031751065746), created detached at the default branch's tip so
nothing in it has a ref to push. Records sit in the store's own ledger, where
the capture and resolve code reads them, under a neutral slug; the link from an
advisory to its record lives in the store's state file in its local tier,
never in the record. Every sub-verb verifies the store first and refuses,
writing nothing, when its HEAD is on a branch, its index tracks a record the
state names, it resolves inside the checkout, or its state file is not this
account's alone. `abcd peers` skips the store, counting it without its path.

## What each sub-verb does

- **intake** lists the advisories GitHub holds in triage and draft through the
  person's own `gh` and mints one private record per advisory not seen before;
  severity maps critical to critical, high to major, medium and low to minor,
  and an advisory with no severity is listed for the person, not recorded.
- **triage** records one of four verdicts (vulnerability, not abcd's, not a
  vulnerability, public track) with who decided it and when; the two closing
  verdicts draft the closing explanation, and "not abcd's" names both halves of
  [abcd-fixes-what-it-adds](../../principles/abcd-fixes-what-it-adds.md).
- **close** shows the closing text and its digest and sends only a read; with
  the approval of that digest it closes the advisory with the text at the head
  of its description (GitHub offers no comment on an advisory), the reporter's
  text kept below a separator, and reads it back.
- **check** is the outbound check: over a commit range (messages and added
  lines, a merge's against its first parent) and texts, it refuses an
  unpublished advisory's id, its reserved CVE id, its title (12 characters or
  more) and the store's location on this machine, naming where and never
  repeating the match. A paraphrase or a count is beyond it and stays the
  author's to avoid.
- **publish** shows the plan with each patched version taken from the first
  release tag containing the fix, leaving out, with the reason, an advisory not
  resolved or released, one with no package, and one whose text would disclose
  a secret, a private name or another advisory under embargo; with the approval
  of that plan's digest it publishes one advisory as a canary, reads it back,
  publishes the rest, and confirms none is left in draft.

## What stays the person's

Every verdict, every approval, fixing and landing the fixes, requesting a CVE,
and restoring redacted records after publication. The command page,
[`commands/embargo.md`](../../../../commands/embargo.md), says so to the host
and to the person.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd embargo`

Sub-verbs: `abcd embargo check`, `abcd embargo close`, `abcd embargo intake`, `abcd embargo publish`, `abcd embargo triage`.

Flags: none.

### `abcd embargo check`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--allow` | stringArray |
| `--file` | stringArray |
| `--range` | string |

### `abcd embargo close`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--approve` | string |

### `abcd embargo intake`

Sub-verbs: none.

Flags: none.

### `abcd embargo publish`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--approve` | string |
| `--release` | string |

### `abcd embargo triage`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--by` | string |
| `--cannot-help` | string |
| `--exists-without` | string |
| `--reason` | string |

<!-- surface-appendix:end -->
