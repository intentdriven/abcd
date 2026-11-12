---
name: landings
description: "List how each change landed on the default branch and what admitted it: Writes only its forge cache in the local tier; refuses no default branch or no range."
argument-hint: "[--since <tag|YYYY-MM-DD>] [--unattended | --reverted] | <commit>"
block: people
---

# `/abcd:landings`

Say how each change landed on the default branch: who or what let it in, and
under which rule. The verb reads history as this checkout last fetched it. It
fetches nothing and pushes nothing; its one network act is a read through the
forge client (`gh api graphql`) of each pull request it has not read before,
and it keeps each answer it read in `.abcd/.work.local/landings/`, the only
thing it writes.

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" landings --json                       # since the last release tag
"${CLAUDE_PLUGIN_ROOT}/abcd" landings --since <tag|YYYY-MM-DD> --json
"${CLAUDE_PLUGIN_ROOT}/abcd" landings --unattended --json          # the rows no person is seen on
"${CLAUDE_PLUGIN_ROOT}/abcd" landings --reverted --json            # the reverted rows, by rule
"${CLAUDE_PLUGIN_ROOT}/abcd" landings <commit> --json              # trace one commit
```

## The range

The listing runs from `--since` to the default branch's tip: after a tag, or
from the first first-parent commit at or after a date `YYYY-MM-DD`. With no
`--since` it starts after the last release tag (`v*`) the tip reaches. With no
such tag it refuses (exit 2) and names the flag.
A `--since` that names neither a tag nor a date is refused the same way.

## What a row says

One row per commit on the default branch's first-parent line, newest first.
The text prints each as one line: the commit, the pull request, the class, the
rule (or, for a person's push, its committer), the lane kind, the approval and
where it was read, and the issues and intents it resolves and delivers, with
the commit that reverts it where one does. After the rows come the notes and
the tip it read.

The classes:

- `person-approved` — a merge a person approved: a recorded approval of its
  merged head in the local tier, a forge review approving that head by someone
  other than the pull request's author, or the human facilitator mode its
  first parent committed.
- `drain` — a merge under the auto mode with no approval seen whose branch
  carries the drain's `Admitted-by:` trailer; its rule is the drain rule record.
- `automatic` — every other merge under the auto mode with no approval seen; its
  rule is `facilitator.mode: auto` as the first parent committed it.
- `person-push` — a commit whose subject names no pull request, or one the
  forge says did not merge as that commit.

The approval's source is `forge`, `local tier`, `none seen` or `forge unread`.
A row whose pull request the forge did not answer for is never placed with a
person, and the `forge` note says why and how many.

`--json` prints the listing: `tip`, `from`, `to`, `since`, `rows`, and, when
they say something, `since_note`, `forge_unread`, `cache_unwritten` and
`mode_unread`. Each row carries `commit`, `when`, `pr`, `merged_head`,
`committer`, `class`, `rule` (`kind`, `id`, `path`, `at`), `lane`,
`approval` (`seen`, `by`, `at`, `source`), `mode`, `resolves`, `delivers`,
`assisted_by`, `forge_says`, `forge_unreadable`, `unattended`, `reverted`
(`by`, `by_row`, `kind`, `withdrawn_by`) and `reverts`.

`--unattended` keeps the rows with `unattended` set: the drain's and the
automatic merges, a person's push no recorded approval names, and every row
the evidence cannot place with a person. `--reverted` adds `groups`, each a
`rule` with its `count` and `rows`, and the untraceable reverts in a group of
their own; the text prints each group's rule and count, then its rows.

## Tracing one commit

Name a commit and the verb traces it: the row that brought it onto the default
branch, however far back, with its class and rule, and for a reverting commit
the rows it reverts. `--json` prints `commit`, `row`, `reverts`, `tip`, `to`
and the notes. A commit the tip does not reach prints `not on the default
branch` and exits 0. A name that names no one commit is refused (exit 2), and a
trace takes none of `--since`, `--unattended` and `--reverted`.

Tell the user the rows that matter to their question, by class and rule, and
read every note out: a `forge` note means some rows could not be placed. Every
value is text someone typed and is masked for the terminal. Exit 0 with the
listing or the trace; exit 1 when history cannot be read; exit 2
outside a git checkout, with no default branch, with no range, or with a
commit named that names none.
