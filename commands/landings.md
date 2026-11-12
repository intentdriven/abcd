---
name: landings
description: "List how each change landed on the default branch and what admitted it: Writes only its forge cache in the local tier; refuses no default branch or no range."
argument-hint: "[--since <tag|YYYY-MM-DD>] [--unattended | --reverted] | <commit>"
block: people
---

# `/abcd:landings`

Say how each change landed on the default branch: who or what let it in, and
under which rule. The verb reads history as this checkout last fetched it. It
fetches nothing and pushes nothing; its one network act is the forge client
(`gh api graphql`), and it writes only `.abcd/.work.local/landings/`.

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
such tag it refuses (exit 2) and names the flag, and so it does when the tag
git describes names no tag the repository holds (git describes an annotated
tag by the name stored inside it, which a renamed ref no longer carries); a
failed read of the tags, or of the ref or commit of the tag named, is not taken
for no tag, and exits 1, but for the two answers git gives without failing that
the exit codes below name. A tag is looked up as that exact ref, its name as git
printed it, never read as the commit a name ending in `-g` and hex names. A
`--since` that names neither a tag nor a date, a tag that points at no commit,
or a date that is no day, is refused at exit 2.

## What it asks the forge, and what it keeps

It asks the forge about each pull request the range names whose answer its
cache does not keep, up to twenty to a document. It keeps, one file per pull
request, only the answers that cannot change: one it accepts for its row, one
saying the pull request merged as another commit, and the forge's word that a
number names no pull request. An answer it could not read, one whose reviews
run past the page it reads, one read without the commits its row asked, and one
saying the pull request has not merged or naming no merge commit are not kept,
so every listing asks for them again. A document whose answer is too large, or
carries an error naming no one pull request, is asked again in halves, within a
bound on the calls one listing spends so. A squashed revert's pull request body
is asked apart, on every listing that reads it, and never kept.

## What a row says

One row per commit on the default branch's first-parent line, newest first.
The text prints each row as its first line: the commit, the pull request, the
class, the rule (or, for a person's push, the committer name git holds), the
lane kind, the approval, where it was read and `seen` where one was, up to
eight of the issues and intents it resolves and delivers and how many more, its
mode, `unattended` where it is, and the commit that reverts it where one does.
The next line says when it landed, the head it merged, where its rule was read,
and up to eight of its `Assisted-by:` values and how many more. Then, each on
its own line, never dropped and each cut at 600 bytes: what the forge said
against the pull request its subject names, and why the forge's answer for it
was not read. Then its revert lines, each with the row it marks or why it marks
none: first those that mark no row, the untraceable ones before the rest, up to
sixteen; then up to eight that mark a row; then how many more, how many of
those mark a row, how many do not, and how many of the ones that do not are
untraceable. A row can carry many lines that mark no row (one for each of its
commits that reads as git's revert and names nothing it resolves, up to 64 for
each commit's message or pull request body, and one for each bound passed),
which is why they have a cap of their own; every printed line that marks none
comes before any that marks a row, and the lines past either cap are counted
in the remainder. `--json` keeps them all. After the rows
come the notes and the tip read.

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

`--unattended` keeps the rows with `unattended` set: the drain's and the
automatic merges, a person's push no recorded approval names, and every row
the evidence cannot place with a person; with none, the text says no row in the
range is unattended. `--reverted` adds `groups`, each a rule with its count and
rows, and the rows carrying an untraceable revert in a group of their own; the
text prints each group's rule and count, then its rows. The two filters are not
taken together (exit 2).

## Tracing one commit

Name a commit and the verb traces it: the row that brought it onto the default
branch, however far back, with its class, rule and revert lines, and for a
reverting commit the rows it reverts; the notes say what the range could not
place. A commit the tip does not reach prints `not on the default branch` and
exits 0. A name that names no one commit (none, two or more, or an object that
is no commit) is refused (exit 2), and `--json` prints the trace with its
`says` first. A trace takes none of `--since`, `--unattended` and `--reverted`.

## What --json prints

- The listing: `tip`, `from`, `to`, `since`, `rows`, and, when they say
  something, `since_note`, `forge_unread`, `cache_unwritten` and
  `mode_unread`; under `--reverted`, `groups`, an empty list when no row is
  reverted.
- A row: `commit`, `when`, `pr`, `merged_head`, `committer`, `class`, `rule`,
  `lane`, `approval`, `mode`, `resolves`, `delivers`, `assisted_by`,
  `forge_says`, `forge_unreadable`, `unattended`, `reverted` and `reverts`.
- A row's rule: `kind`, `id`, `path`, `at`.
- A row's approval: `seen`, `by`, `at`, `source`.
- A row's reverted mark: `by`, `by_row`, `kind`, `withdrawn_by`.
- Each of a row's reverts: `commit`, `kind`, `names`, `target`, `row`,
  `unmarked`.
- A group: `rule`, `count`, `rows`.
- A trace: `commit`, `says`, `row`, `reverts`, `tip`, `to`, `from`,
  `range_note`, `forge_unread`, `cache_unwritten`, `mode_unread`.

## Telling the user

Tell the user the rows that matter to their question, by class and rule, and
read every note out: a `forge` note means some rows could not be placed. The
render masks every value it prints for the terminal; the committer name, the
forge's logins and failure text, the lane, the ids, a revert line's names and
the notes carry text someone typed. One row's text is bounded; the number of
rows is bounded only by the history the reader takes, whose log it refuses
whole past 64 MiB.

Exit 0 with the listing or the trace, and with a listing whose forge answers
were partly or wholly unread: those rows are unattended and the `forge` note
says how many, so a script reads `forge_unread`, not the exit alone. Exit 1
when history, the tags or the repository cannot be read, but for the three
reads below. Exit 2 outside a git checkout, with no default branch, with no
range, with both filters, or with a commit named that names none.

A lookup of the default branch that failed is reported as no default branch,
at exit 2, because the shared lookup gives one answer for both. Answers git
gives without failing are taken as git's answer: a tag ref git's listing skips
reads as no tag, a tag given as the range's start whose commit object is absent
reads as pointing at no commit, and a default branch whose tip's commit object
is absent reads as no default branch.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout.
