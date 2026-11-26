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
example, what usually comes next once the command succeeds, and the visible
sub-commands. A flag list that would run past the bound
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
facts' format version. Where the page's commands declare a visible flag, read
an input or name what follows them, the block then lists each visible
command's usage line, the inputs it reads (a filled-in JSON input with its
schema version), its example and what usually follows its success; otherwise
the first line is all it holds. The drift tests in
`internal/surface/cli/commandfacts_drift_test.go` fail when a committed copy
differs from a fresh rendering of the tree. The command-page facts lint
(`internal/core/lint/commandpagefacts.go`) reads each page's own prose outside
those blocks and reports, by page and line, an invocation naming a command the
tree lacks, a flag that neither the command nor a persistent flag on its path
registers, or a `schema_version` the input does not have. It reports nothing on
a line marked `<!-- facts-lint: historical -->`, though it still reads that
line to tell which input a nearby `schema_version` belongs to.

## Next steps on success

Every visible command names what usually comes next once it succeeds
(itd-2610301428378830, spc-2610301431108175 step 6). One table held in code
(`internal/core/surface/next.go`) gives each visible command its steps, each a
command template and one line saying when it applies, or the explicit empty
list with the reason nothing usually follows. The facts carry the table's
entry as `next`, `no_next` and `streaming`, so the text answer, the
machine-readable form, the per-command JSON files and the command-facts block
of every page show the same steps; the facts' format version moved to 3 with
them.

A successful machine-readable result carries the same steps as a top-level
`next_steps` key. The root's pre-run hook wraps a visible command's stdout,
when the call asks for machine-readable output alone, in a writer that
carries the command's path and its entry, and the one helper every result is
written through (`render`, `internal/surface/cli/next_steps.go`) encodes the
result as it always has and splices the key in before the closing brace:
nothing is decoded and encoded again, so no key a result already has moves.
The key is never `next`, which several results already use for their own next
action. A command whose run can write a document before its last one is
marked streaming, and only its closing summary, written through
`renderFinal`, carries the key; a document a refusal follows, or that stands
in for the error object, is written through `renderRefused` and carries none.
A hidden command keeps its stdout as it is. A document that is not an object
takes no key: four commands print a JSON array, which a key cannot join
without breaking its readers, so their results are left as they are while
their facts still carry the steps; the exemption list in
`internal/surface/cli/next_steps_test.go` names them. How such a command
should name its next step in its result is an open product question. The test
binary checks every machine-readable document the package's tests render, and
fails on a result that already carries a top-level `next_steps`, or on an
array from a command the exemption list does not name; the tests in the same
file hold every visible command to an entry and every template to a command
path the tree has, with flags that command takes.

## Refusal codes

A refusal (exit 2, or 3 when someone else holds what the call needs) carries a
short code, a one-line hint and the next command to try
(itd-2610301428378830, spc-2610301431108175), and `abcd explain <code>`
prints what the code means. A word holding a slash is read as a code rather
than a command path. The answer gives the code's summary, a few lines of
explanation, the command to try next or `none` where no command gets past the
refusal, the commands that raise the code (every command, for a `usage/`
code) and, for a retired code, the release that retired it; the
machine-readable form is the code's registry entry. An unknown code is
refused at exit 2 as `explain/unknown-code`, naming the nearest registered
code and never repeating the word.

The codes live in one table held in code (`internal/core/refusal`). A code is
a short name, the verb, a slash and the reason in lower-case kebab-case, never
a number ([adr-2610301638558260](../../decisions/adrs/2610301638558260-a-published-refusal-code-is-never.md)):
`usage/` names the command framework's own refusals (an unknown command or
flag, too few or too many positional arguments, two flags that cannot go
together, a required flag left out, a flag value that is missing or does not
parse), which every command raises, and the hook plane raises at exit 1. A
core package names a code where the refusal is decided, by wrapping its error
(`refusal.Wrap`), and the front door finds it however the error was wrapped;
the front door builds a refusal only through two constructors, one naming a
registered code and one for a code known only at run time, which marks a
refusal left without one as `internal/uncoded`, a code that is never
registered.

The error boundary (`run`, `internal/surface/cli/cli.go`) reads the code from
the error, or else from the usage classifier, which reads the parser's error
types and, where it has none, the command framework's wording. In the
machine-readable form the error object keeps `abcd`, `error` and `exit_code` exactly as they were
and adds `code`, `hint` and `next`; the text form adds one line after the
refusal's own: `abcd: <hint> — try: <next> (abcd explain <code>)`. A next
command holding a word in angle brackets (`<fix>`) is a template whose
bracketed words the caller fills in, and the text form offers it with `fill
in and run:` instead of `try:`; any other runs as it stands. The next
command is filled from the refused command's own path, never from the
caller's arguments, which the boundary withholds. A refusal not yet coded
carries none of the three.

Tests hold the table whole: no two entries share a code, every code follows
the grammar, every code resolves through `explain`, every next command and
every raising command names a command the tree has, every command codes an
unknown flag and every parent the framework refuses codes an unknown
sub-verb, and the test binary fails on a refusal that leaves with
`internal/uncoded` or with a code the table does not hold.
`make refusal-codes-sync` (`cmd/refusal-codes-sync`) writes the ledger of
codes, `.abcd/development/release/refusal-codes.json`, from the table, and a
drift test holds it. The release cut reads the ledger at the last tag and
refuses, as `refusal-codes`, a cut in which a code that ledger lists is
missing from HEAD's, or a code it lists as retired is live again
([`04-launch.md`](04-launch.md)); a tag that carries no ledger passes.

The migration of every refusal to a code runs over several steps. While it
runs, `.abcd/development/release/refusal-uncoded.txt` names the files whose
refusals carry no code yet, and the test that refuses a refusal built outside
the two constructors exempts only those. The list only shrinks, under a
ceiling the test holds, and the intent cannot ship while it exists.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd explain`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--input` | string |

<!-- surface-appendix:end -->
