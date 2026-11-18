# `/abcd:explain` — Ask the Running Binary How to Call a Command

`/abcd:explain` answers, from the binary an agent runs, how to call one of that
binary's commands (itd-2610301346085246, spc-2610301431075964). An agent used
to learn a verb from its plugin page, which ships with the plugin version the
session loaded rather than with the binary; on 2026-10-30 an agent composed an
ideate verdict from a page at schema version 1 and the binary, which reads
version 2 only, refused it. This verb is the answer that cannot be older than
the binary. It is listed in the agents' block of the help, its page names it as
the page to read, and it runs in the process, reading only, touching no network
and writing nothing.

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

## What it answers

Given the words of a command path, it prints that command's facts in at most
40 lines: the path and the command's one sentence, the usage line, each flag
with its type, its closed set of values (on a list flag, with what in each
value the set applies to: each member of a comma-list, or one segment of each
repeated value), its default and whether it is required, the persistent flags
the command inherits from the commands above it, each filled-in input with the schema version the binary reads, the worked
example and the visible sub-commands. A flag list that would run past the bound
is cut, with one line saying how many flags are left and that the
machine-readable form lists them all. A hidden flag is left out of the text.

Asked for one input, it prints that input's schema version and where the
document carries it, the field list derived from the owning package's Go type,
and the example the owning package holds beside the type. A test runs every
registered input's example, as the verb prints it, through the check the
command runs before it writes. An interview's answers file is named among a
command's inputs but has no shape here, because the interview that asks the
questions prints it.

The machine-readable form for a command is a document holding the facts'
format version and a list of commands with the one command's facts in it, the
shape each per-command JSON file under `docs/reference/cli/commands/` carries,
cut to one command. For an input it is the input's shape as the registry
builds it.

## Where the facts come from

Everything the verb prints comes from one walk of a fresh command tree, never
executed: the same walk that writes the compatibility snapshot
(`internal/surface/cli/surface.go`), with each input's shape from the input
registry (`internal/surface/cli/inputs.go`). The sentence and the example are
the manifests' in `internal/core/surface/`. There is no second copy for the
answer to drift from. The tests in `internal/surface/cli/explain_test.go` run
the verb on every command in the tree, hidden ones included, hold each answer
to the bound, compare its flags, inputs, example and sub-commands with the
command's facts, and compare the machine-readable answer with the facts
themselves.

## Paths and refusals

A hidden command is answered, its first line marked hidden, so a refusal on the
hook plane can name this verb as its next step. The root answers by its own
name, and a leading root name before a verb is read as the root. A path the
tree does not have is refused at exit 2, naming the nearest real path where one
is within two edits of the word, and otherwise the visible sub-commands where
the path went wrong; a word after a command that takes no sub-command is
refused the same way. An input the command does not read is refused at exit 2,
naming the inputs it does read, and so is an input asked for with no command
named. A refusal names only paths and flags the tree has, never a word that
matched none. Bare, the verb prints its own help, which is its entry in the
bare-invocation exceptions ([`README.md`](README.md#bare-invocation)).

`make command-facts-sync` (`cmd/command-facts-sync`) writes the same facts
into derived copies: one JSON file for the root and one for each visible
top-level verb of the command tree as built, under
`docs/reference/cli/commands/` (the `help` and `completion` verbs, which cobra
adds only when the binary runs, have none), and two generated blocks on every
page under `commands/`. The first stands straight after the page's heading and
tells an agent to run `abcd explain <verb>` before it fills in an input, and to
follow the binary where the page disagrees with it. The second ends the page,
or stands just before its closing user-input line. Its first line names the
facts' format version. Where the page's commands declare a visible flag or read
an input, the block then lists each visible command's usage line, the inputs it
reads (a filled-in JSON input with its schema version) and its example;
otherwise the first line is all it holds. The drift tests in
`internal/surface/cli/commandfacts_drift_test.go` fail when a committed copy
differs from a fresh rendering of the tree. The command-page facts lint
(`internal/core/lint/commandpagefacts.go`) reads each page's own prose outside
those blocks and reports, by page and line, an invocation naming a command the
tree lacks, a flag that neither the command nor a persistent flag on its path
registers, or a `schema_version` the input does not have. It reports nothing on
a line marked `<!-- facts-lint: historical -->`, though it still reads that
line to tell which input a nearby `schema_version` belongs to.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd explain`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--input` | string |

<!-- surface-appendix:end -->
