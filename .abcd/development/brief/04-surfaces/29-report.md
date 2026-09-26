# `/abcd:report` — Report Back to abcd

`/abcd:report` is how a repository abcd manages tells abcd itself about a defect
or proposes an enhancement. It files a written account against an abcd-issued
template into an inbox in the user account's machine store, where abcd finds it
by itself; nothing is written into the reporting repository or into abcd's
(itd-2609221656361680, spc-2609221657168936). The reading and filing half is
[`/abcd:inbox`](30-inbox.md).

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|

The table is empty: the verb registers no sub-command. Printing the skeleton is
a flag and the filled report is a positional; neither is a sub-verb.

## Behaviour

- Asked for the template, `abcd report` prints the skeleton and writes nothing.
  It is a machine-readable block between `---` lines beside prose the reporter
  writes.
- `abcd report <file>` validates a filled report and files it; `abcd report -`
  reads it from stdin. The verb prints the report's id (`rpt-` and the sixteen
  digits of the shared record-id mint) and where it landed, with the home
  directory written as `~`.
- Bare `abcd report` opens the skeleton in `$VISUAL` or `$EDITOR` when both ends
  of the session are a terminal, from a private temporary file outside both
  repositories, and files what is saved. Without a terminal or an editor it
  refuses and names the two other ways in. An edit that is refused, or fails
  to file, keeps the file and names it, so what the reporter wrote is never
  lost.

## The template

| Field | Holds |
|---|---|
| `schema_version` | the template version; this abcd writes and reads `1` |
| `kind` | `defect` or `enhancement` |
| `severity` | capture's severity vocabulary |
| `category` | capture's category vocabulary |
| `title` | one line saying what was found |
| `abcd_version` | the abcd in play, filled in by the template |
| `surface` | the verb, hook or page in play |
| `remedy` | optional: what the reporter would change |
| `evidence` | optional pointers, one `  - pointer` per line: record ids, commit SHAs, URLs |

The prose below the block is required; the template's placeholder is an HTML
comment and does not count. Every HTML comment is removed from the prose when
the report is read, so the placeholder a reporter writes below never reaches the
inbox or a capture, and a comment opened and never closed is refused. The issued
skeleton, unchanged, is refused.

## What a report may not carry

A report arrives from another repository, so it is untrusted input, and the
validator refuses it, naming the field, when:

- it is over 32 KiB, or not UTF-8, or carries a control byte (C0, DEL or C1)
  other than a tab or a line break;
- a field or the prose carries a bidirectional control or a zero-width
  character (a byte-order mark opening the file excepted), judged by the
  terminal sanitiser's own predicate: such text displays differently from its
  bytes;
- a field is missing, empty where it is required, outside its closed
  vocabulary, over its length bound, duplicated, or not a field of the
  template;
- it carries `received_at`, `sender_key` or `sender_name`, which are abcd's to
  write;
- a field names a filesystem location: an absolute or home-relative path,
  `$HOME` or `%USERPROFILE%`, a Windows drive or a UNC path in either slash, a
  path after a colon, a `file:` URL, or a `..` segment. A report points at
  records, commits and URLs, never at a location on a machine. This check covers
  the block's fields only and is best effort, not a boundary: the prose is not
  matched, and a location spelt some other way passes. Nothing rests on it being
  complete, because nothing in a report is ever opened, fetched or executed.

A report written to a template version this abcd does not know is refused at
filing with the version named, and listed as unreadable in the inbox.

## Where it lands

```text
~/.abcd/inbox/<received-stamp>-<sender-key>.md
```

The inbox is a machine-scoped store beside the history, transcript, worktree,
sources, labs and run stores. The sender key is the reporting repository's full
root-commit SHA, the key those stores use; the stamp is the report id's digits.
abcd derives the file name and writes the file by an exclusive create at mode
0600, in a directory created one real level at a time at 0700; a symlink or a
file at any level is refused, naming that level
([`30-inbox.md`](30-inbox.md)). The stored file
is what the validator accepted, written back by abcd with the envelope it
stamps: `received_at`, `sender_key`, and `sender_name`, the name of the
repository's main checkout directory, so a worktree reports under its
repository's name.

## The greeting

At the next session start, one line on the hook's stdout says how many reports
wait and from how many repositories, and nothing else:

```text
abcd: 3 report(s) from 2 managed repositories wait in the inbox; `abcd inbox` lists them.
```

It carries counts only — no sender name and no word a report wrote — because
the session-start stdout is injected into the session's context. The bare
[`/abcd`](08-abcd.md) board carries the same count as its `inbox:` row. Both are
silent when nothing waits. Neither is silent when the inbox cannot be counted:
the hook names the refusal in one line among its notices on stderr, and the
board prints the same line on stderr in place of the row —
`abcd: the inbox is not counted — ~/.abcd/inbox is not a real directory …`,
naming the level refused, home-redacted — so a refused inbox never reads as an
empty one.

## Exit codes

`0` filed; `1` filing failed (the inbox cannot be created, every id drawn this
second is taken, the write fails), with nothing filed; `2` refused, with nothing
filed. After the editor ran, a failure names the kept file as a refusal does. The JSON output holds on every path:
a refusal is the `{"abcd":"error",…}` envelope on stdout.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd report`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--template` | bool |

<!-- surface-appendix:end -->
