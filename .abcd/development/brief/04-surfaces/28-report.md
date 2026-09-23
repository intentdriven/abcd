# `/abcd:report` — Report Back to abcd

`/abcd:report` is how a repository abcd manages tells abcd itself about a defect
or proposes an enhancement. It files a written account against an abcd-issued
template into an inbox in the user account's machine store, where abcd finds it
by itself; nothing is written into the reporting repository or into abcd's
(itd-2609221656361680, spc-2609221657168936). The reading and filing half is
[`/abcd:inbox`](29-inbox.md).

## Behaviour

- `abcd report --template` prints the skeleton and writes nothing. It is a
  machine-readable block between `---` lines beside prose the reporter writes.
- `abcd report <file>` validates a filled report and files it; `abcd report -`
  reads it from stdin. The verb prints the report's id (`rpt-` and the sixteen
  digits of the shared record-id mint) and where it landed, with the home
  directory written as `~`.
- Bare `abcd report` opens the skeleton in `$VISUAL` or `$EDITOR` when both ends
  of the session are a terminal, from a private temporary file outside both
  repositories, and files what is saved. Without a terminal or an editor it
  refuses and names the two other ways in. A refused edit keeps the file and
  names it, so what the reporter wrote is never lost.

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
comment and does not count. The issued skeleton, unchanged, is refused.

## What a report may not carry

A report arrives from another repository, so it is untrusted input, and the
validator refuses it, naming the field, when:

- it is over 32 KiB, or not UTF-8, or carries a control byte other than a tab
  or a line break;
- a field is missing, empty where it is required, outside its closed
  vocabulary, over its length bound, duplicated, or not a field of the
  template;
- it carries `received_at`, `sender_key` or `sender_name`, which are abcd's to
  write;
- a field names a filesystem location: an absolute or home-relative path, a
  Windows drive or UNC path, a `file:` URL, or a `..` segment. A report points
  at records, commits and URLs, never at a location on a machine, and nothing in
  it is ever opened.

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
0600, in a directory created one real level at a time at 0700. The stored file
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
silent when nothing waits.

## Exit codes

`0` filed; `2` refused, with nothing filed. `--json` holds on every path: a
refusal is the `{"abcd":"error",…}` envelope on stdout.
