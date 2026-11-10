# Library and memory

abcd keeps two kinds of remembered material, and they do different jobs. The
**library** is evidence you consult. **Memory** is know-how that finds you.

| | Library (`abcd library`) | Memory (`abcd memory`, and memory pages) |
|---|---|---|
| What it holds | Outside material: papers, links, and the cited pages distilled from them | Short lessons learned while working, such as "this test is flaky under load" or "the person wants every question to carry an example" |
| Who adds to it | The person, deliberately, saying whether each source is confidential or public | Mostly agents, in the middle of a session, without asking |
| How it comes back | You ask ("what do my sources say about pacing?") and get answers that cite the page they came from | Automatically, when a prompt matches the words a note or page was saved with |
| Where it can go | Into a decision as a cited source; a confidential source is never named in anything public | Into the project's own memory file, through a person's reviewed change, so that agents working in the project receive it too |

A library answer can become a memory note ("source X says Y", with its
citation), but the two are not one store with two names. The library is for
what a decision may need to cite. Memory is for what an agent would otherwise
have to learn again.

## Notes and pages

Memory comes in two sizes.

A **memory note** is a lesson of a few sentences: a title, a short body, and
the recall words that should bring it back. An agent writes one with
`abcd memory add` whenever it learns something worth keeping. Each note is
stamped with who wrote it, when, and in which session, and the command refuses
a note that would hold a secret, a private name or a network address.

A **memory page** is longer: a cited page in the project's `.abcd/memory/`,
written by `abcd library ingest` from a source. A page comes back on its own
only when it carries a `recall:` list of words; a page without one is reached
by asking the library.

## Where memory lives

Personal notes live on your machine, in `~/.abcd.noindex/memory.json`. A note
is tied to the repository it was written in and comes back only there, unless
it was written for every project. Notes the project shares live in its
`.abcd/memory.json`, which is committed like any other file. abcd's memory
commands do not write that file: a note gets there when a person adds it in a
change the project reviews.

Pages live in the project's `.abcd/memory/`, and personal pages in
`~/.abcd.noindex/memory/`. When both hold a page of the same name, the
project's page is the page of that name: the personal one is not recalled,
even when the project's page carries no recall words or could not be read.

When abcd recalls memory it reads its own notes and pages, not your agent
host's own memory files. To move a lesson from them into a note, point the
agent at it; the agent reads it and files it with `abcd memory add`, which
takes the text on standard input rather than a file path, and the original
stays where it was.

## How memory comes back

The hook that brings the project's rules into a session also looks for notes
and pages whose recall words appear in your prompt. What matches comes back in
one block, headed `## MEMORY`, after the rules. Each item says where it came
from, so a note from the project's file never reads as one of your own. Notes
and pages are ranked together, and a rare word that matches counts for more
than a common one, so the most telling item comes first.

An item that has come back is not sent again unchanged within the session.
It can come back again after the session is compacted, in a new session, or
on the loader's refresh backstop, a prompt that sends the matched rules and
memory again: every fifteenth prompt by default, or the interval
`rules.force_refresh_every_n` sets in `.abcd/config.json`. An item whose words
have changed since it was sent counts as new.

## Memory shrinks as the session fills

A session's context window is finite, so recall gives way as it fills. Every
turn has a size limit, which the rules and memory share. While most of the
window is free, matched notes and pages come back in full, best-ranked first,
as far as the turn has room, and the next few as one line each. As it fills,
the best-ranked come back in full within a smaller bound and the rest as one
line each as far as the turn has room, and the rules shrink to each domain's
first rule and a pointer to the rest. Near the end of the window the
best-ranked come back as one line each, within a small bound, and at least
the top one does. A match a turn
leaves out is not counted as sent, so a later prompt that matches it can
bring it back.

abcd learns how full the session is from its status line, where you accepted
it at install, or else from the end of the session's transcript. When neither
tells, the rules come back in full where the turn has room, and notes and
pages as near the end of the window: the best-ranked as one line each, at
least the top one.

To see everything one prompt matches, why each item ranks where it does, and
the form each would take, run `abcd rules --prompt "<the prompt>"`. It
changes nothing.

For the steps, see [Keep memory notes](../how-to/keep-memory-notes.md).
