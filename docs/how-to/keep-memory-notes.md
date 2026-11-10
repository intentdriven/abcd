# Keep memory notes

A memory note is a short lesson an agent keeps so that a later session does
not have to learn it again. Agents write notes on their own, through abcd's
agents' command for memory. This guide shows how to write one by hand, review
the notes kept for you, and see what a prompt brings back. For what notes are
and how they differ from the library, see
[Library and memory](../explanation/memory.md).

## Write a note

Pass the title, one or more recall words, and who wrote it, with the body on
standard input:

```bash
abcd memory add --title "The tests need the race flag" \
  --recall test --recall race \
  --by "<Vendor>:<model-version>" <<'NOTE'
Run the tests with the race flag before saying they pass; the plain run
misses the data race in the hook tests.
NOTE
```

- Choose recall words a later prompt would contain. The note comes back when a
  prompt matches one of them, by the same rule the project's rules are
  matched by, so `test` also matches "tests" and "testing".
- `--by` takes the form of an `Assisted-by:` trailer, `<Vendor>:<model-version>`.
- `--session <id>` records the host's id for the session. Without it the note
  records its session as unknown, and the command says so.
- The note is tied to this repository. Add `--everywhere` for a lesson that
  holds in every project; such a note has been checked against this
  checkout's private names only, and the command says so.

The command prints the new note's `mem-` id. It refuses a title over 120
characters, a body over 2 KiB, fewer than one or more than twelve recall
words, and any note holding a secret, a private name or a network address. A
refusal names the field and the kind of thing found, never the text itself,
exits 2 and writes nothing.

## Review your notes

Count the notes kept for this repository, for every repository, and in the
project's file:

```bash
abcd memory
```

It also names any memory file it could not read, and why. Read the notes
themselves, each with its body, recall words, who wrote it, when, in which
session, and whether it is tied to this repository or recalled everywhere:

```bash
abcd memory list
```

## Remove a note

Remove any personal note by the id `abcd memory list` shows, whoever wrote it:

```bash
abcd memory remove <mem-id>
```

A note in the project's `.abcd/memory.json` is refused here. Remove it by
editing that file in a reviewed change, the same way a note gets there.

## See what a prompt brings back

```bash
abcd rules --prompt "why do the tests fail under load?"
```

It shows, for that one prompt in a fresh session, the rule domains it matches
with the words that hit, then each note and page it would recall with its
rank, its score and the terms that scored it, and the form each would take:
in full or as one line. Without `--fill` it does not know how full a session
is, so it shows the form used when nothing tells: rules in full, notes and
pages as one line each. To see a session that is 65% full:

```bash
abcd rules --prompt "why do the tests fail under load?" --fill 65
```

The command reads only and writes nothing. Add `--json` for the same facts as
one JSON document.
