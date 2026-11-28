---
name: explain
description: "Print how to call one command, an input's shape, or what a refusal code means: Writes nothing; refuses a command path or a code the binary does not have."
argument-hint: "<verb> [<sub-verb>...] [--input <flag>] | <code>"
block: agents
user-invocable: false
---

# `/abcd:explain` — ask the binary how to call a command

<!-- generated: ask-the-binary -->
Before you fill in an input for this command, run `"${CLAUDE_PLUGIN_ROOT}/abcd" explain explain`, and where its answer and this page disagree, follow the binary, since this page may be older than it.
<!-- /generated -->

The running binary answers for itself. A command page ships with the plugin
version this session loaded, and the binary it runs may be newer or older, so
when a page and `explain` disagree about a flag, a value or an input's schema
version, follow `explain`. Ask it before you fill in an input for any command.

## One command

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" explain ideate record
```

It prints, in at most 40 lines and in this order:

- the command's path and its one sentence (what it does, what it writes and
  when it refuses);
- the usage line;
- each flag with its type, the values it accepts when it takes a closed set
  (`one of` for a single value; on a list flag, `each one of` for every member,
  or `each value's tier one of` where the set applies to one segment of each
  value, so a route value is `<agent>=<tier>`, never a tier alone), its default
  and whether it is required, then the flags it inherits from the commands
  above it, such as `--json`;
- each filled-in input it reads, with the schema version this binary reads and
  the call that prints the input's shape;
- the worked example, when it has one;
- what usually comes next once the command succeeds: each step's command,
  its words in angle brackets values to fill in, and when the step applies,
  or `Next: nothing usually follows:` and the reason;
- the sub-commands, when it has any.

A boolean flag takes no value and reads `(bool)`. A hidden flag is left out of
the text and kept in `--json`, marked hidden. A command whose flags would
run past 40 lines shows the first of them and one line saying how many more
there are; `--json` lists them all. Name the root as `abcd` (`explain abcd`
answers for the board), and a leading `abcd` before a verb is read as the root.

A hidden command, such as one on the hook plane, is answered too, and its first
line says `(hidden)`.

## One input

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" explain ideate record --input verdict-json
```

The flag is named with or without its dashes. It prints the input's schema
version and where the document carries it (a `schema_version` member, the
version in its `_type` tag, or none for an input that carries no version), each
field with its JSON type and whether the type declares it required, a nested
object's fields indented beneath it, and an example. The example is the one the
owning package holds beside the type, and a test runs it through the same check
the command runs before it writes, so a document built from it passes that
check. Build the input from this answer, not from a page.

An interview's `--answers` file has no shape here: the interview that asks the
questions prints it.

## One refusal code

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" explain capture/missing-remedy
```

When abcd refuses a call, the refusal carries a short code, such as
`capture/missing-remedy` or `usage/unknown-flag`: the verb, a slash and the
reason. A word holding a slash is read as a code, and `explain` prints:

- the code and its one-line summary, the hint the refusal showed;
- a few lines saying what the refusal means and how to get past it;
- the command to try next, or `none` where no command gets past it (a
  refusal that says to wait, say); a next command that names the refused
  command shows it as `<the refused command>`, and a template, whose words in
  angle brackets such as `<fix>` are values to fill in, says so;
- the commands that raise the code, or, for a `usage/` code, that every
  command raises it;
- for a retired code, the release that retired it.

A refusal under `--json` carries the same three facts beside its message:
the error object every verb prints on stdout keeps `abcd`, `error` and
`exit_code`, and adds `code`, `hint` (the summary) and `next` (the command to
try, filled in with the refused command, or `none`). A `next` holding a word
in angle brackets, such as `abcd capture "<text>" --remedy "<fix>"`, is a
template: fill in each bracketed word before running it. Any other `next`
runs as it stands. Without `--json`, the refusal's line on stderr is followed
by one more: `abcd: <hint> — try: <next> (abcd explain <code>)`, or
`— fill in and run: <next>` for a template. The next command is built from
the refused command alone, never from what was typed. A refusal not yet
given a code carries none of the three, and no second line.

Once a release has carried a code, the code keeps its meaning: it is never
renamed or given to another refusal, and a code whose refusal goes away is
kept, marked retired, so `explain` still answers it. Match on the code, never
on the message, which can be reworded in any release. A code from a newer
release is unknown to an older binary.

## Machine-readable

```bash
"${CLAUDE_PLUGIN_ROOT}/abcd" explain ideate record --json
"${CLAUDE_PLUGIN_ROOT}/abcd" explain ideate record --input verdict-json --json
```

For a command, `--json` prints `format_version` and `commands`, a list holding
the command's facts: `path`, `hidden` when the command itself is declared
hidden, `sentence`, `usage`, `flags` (each with `name`, `shorthand`, `type`,
`default`, `required`, `allowed`, `each`, `persistent` and `hidden`, a field
left out when it is empty or false; `each` is present only on a list flag and
says what in each value `allowed` applies to: `member` when every member of a
comma-list or repeated value is one of the set, or the name of the segment
that is, such as `tier` in a route's `<agent>=<tier>`), `inputs` (each with
`flag`, `kind`, `name`, `schema_version` and `version_in`), `example`,
`subcommands`, `next` (each step with `command` and `when`), `no_next` (the
reason nothing usually follows, in place of `next`) and `streaming` (present
on a command whose run can write a document before its last one). A sub-command
of a hidden parent, such as one on the hook plane, carries no `hidden` key of
its own, though its text answer says `(hidden)`. With `--input` it prints the
input's shape: `path`, `flag`, `name`, `schema_version`, `version_in`, `root`
(`object` or `array`), `fields` and `example`.

## What comes next after a success

Every visible command's successful result under `--json` is a JSON object
that ends with a `next_steps` key: the same steps `explain` prints for the
command, each with `command` and `when`, or an empty list where nothing
usually follows. The key is added after every key the result already has, and
no other byte of the result changes, so a reader of the result as it was is
not broken. It is `next_steps`, never `next`: several results carry a `next`
of their own, such as the loop's next action, and keep it as it was. A
command whose run can write more than one document (a loop's refusal
document ahead of the error object, say) carries `next_steps` on its closing
summary alone, and one that goes on running after its document, such as
`abcd session serve`, writes that document at once with no key. Only a command
that succeeds adds the key: a document a non-zero
exit or an error object follows carries none. A hidden command adds nothing. Four commands print a JSON array rather than an object, and an
array takes no key without breaking its readers: `abcd decide` bare,
`abcd history list`, `abcd history staged` and `abcd library ledger`. Their
results are left as they were, and `explain` still names their next steps.

For a code, `--json` prints the code's entry: `code`, `summary`,
`explanation`, `next` (holding `{command}` where the refused command is filled
in), `raised_by` (absent for a `usage/` code) and `retired` (present only for
a retired code).

## Refusals

Each refusal exits 2 and names what was wrong on stderr, or, with `--json`, as the
error object every verb prints on stdout, with its code:

- a command path the tree does not have, naming the nearest real path when one
  is a letter or two away, and otherwise the sub-commands where the path went
  wrong; a word after a command that takes no sub-command is refused the same
  way (`explain/unknown-path`);
- `--input` naming a flag the command reads no input through, naming the inputs
  it does read (`explain/unrecognized-input`);
- `--input` with no command named (`usage/missing-argument`), or with a code
  (`usage/flag-conflict`);
- a code this binary does not register, naming the nearest registered code
  (`explain/unknown-code`).

Bare, `explain` prints its own help. It reads only and writes nothing.

**Binary resolution.** Run `"${CLAUDE_PLUGIN_ROOT}/abcd"` — a plugin install
provisions the binary into the plugin root, so this is the rung that fires for a
plugin user. If that path does not exist, try `abcd` on `PATH`; if that fails
too, you are in a source checkout of this repo, where — and only there —
`go run ./cmd/abcd` works, the published payload carrying no `cmd/`. To put a
binary on `PATH`, run `ahoy install` through whichever rung just resolved:
`"${CLAUDE_PLUGIN_ROOT}/abcd" ahoy install`, `abcd ahoy install`, or
`go run ./cmd/abcd ahoy install` in a source checkout. In a checkout the
person has listed in `~/.abcd.noindex/dogfood-roots`, or a worktree that
checkout records, `"${CLAUDE_PLUGIN_ROOT}/abcd" hook dogfood-binary` prints
the path of a build of the reviewed branch's tip, as the remote that line
declares last reported it, once it is ready: run that path in place of the
plugin-root binary, and the plugin-root binary while it prints nothing.

<!-- generated: command-facts -->
**Command facts, format version 3.** Where this block and `"${CLAUDE_PLUGIN_ROOT}/abcd" explain explain` from the running binary disagree, follow the binary.
<!-- Written by `make command-facts-sync` from the command tree; change the command in internal/surface/cli or internal/core/surface, never this block. -->

- `abcd explain [<verb> [<sub-verb>...] | <code>] [flags]`
  - An example call is `abcd explain ideate record --input verdict-json`.
  - Nothing usually follows: the answer is what the next call is built from; no abcd command usually follows it.
<!-- /generated -->

**User input:** $ARGUMENTS
