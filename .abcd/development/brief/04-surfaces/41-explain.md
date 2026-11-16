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
with its type, its closed set of values, its default and whether it is
required, the persistent flags the command inherits from the commands above
it, each filled-in input with the schema version the binary reads, the worked
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

The page blocks generated from the same facts, and the first instruction on
every command page that sends an agent here, are the spec's later steps.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd explain`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--input` | string |

<!-- surface-appendix:end -->
