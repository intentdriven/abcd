---
name: explain
description: "Print how to call one command, or the shape of one input it reads: Writes nothing; refuses a command path the tree does not have."
argument-hint: "<verb> [<sub-verb>...] [--input <flag>]"
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
`flag`, `kind`, `name`, `schema_version` and `version_in`), `example` and
`subcommands`. A sub-command
of a hidden parent, such as one on the hook plane, carries no `hidden` key of
its own, though its text answer says `(hidden)`. With `--input` it prints the
input's shape: `path`, `flag`, `name`, `schema_version`, `version_in`, `root`
(`object` or `array`), `fields` and `example`.

## Refusals

Each refusal exits 2 and names what was wrong on stderr, or, with `--json`, as the
error object every verb prints on stdout:

- a command path the tree does not have, naming the nearest real path when one
  is a letter or two away, and otherwise the sub-commands where the path went
  wrong; a word after a command that takes no sub-command is refused the same
  way;
- `--input` naming a flag the command reads no input through, naming the inputs
  it does read;
- `--input` with no command named.

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
**Command facts, format version 2.** Where this block and `"${CLAUDE_PLUGIN_ROOT}/abcd" explain explain` from the running binary disagree, follow the binary.
<!-- Written by `make command-facts-sync` from the command tree; change the command in internal/surface/cli or internal/core/surface, never this block. -->

- `abcd explain [<verb> [<sub-verb>...]] [flags]`
  - An example call is `abcd explain ideate record --input verdict-json`.
<!-- /generated -->

**User input:** $ARGUMENTS
