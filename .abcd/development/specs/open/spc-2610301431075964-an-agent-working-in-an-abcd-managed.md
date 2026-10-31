---
id: spc-2610301431075964
slug: an-agent-working-in-an-abcd-managed
intent: itd-2610301346085246
origin: researcher-authored
production_mode: hand-written
---
# An agent learns how to call abcd from the abcd it runs

## Summary

Delivers itd-2610301346085246. A new verb, `abcd explain <verb> [<sub-verb>…]`,
prints one command's facts in at most 40 lines. `abcd explain <path> --input
<flag>` prints the exact shape a filled-in input must take, at the schema
version this binary accepts, with one example that passes the command's own
decoder. Those facts are gathered once, from the program's own code. The same
facts are written to a plain JSON file per command and to a generated block in
each command page. A drift test fails the build when any copy is stale, and a
lint fails when a page's hand-written prose names a flag, command or schema
version that the facts do not hold. Every page's first instruction sends the
agent to `abcd explain` before it fills in an input.

**Build order across the three sibling specs.** All three touch
`internal/surface/cli/cli.go`, the owning core packages and the command pages,
and `abcd build` refuses two runs whose footprints overlap. So they are built
one after another, in this order:
1. this spec's step 1;
2. spc-2610301431131636 (the practice run);
3. this spec's steps 2 to 4;
4. spc-2610301431108175 (refusal codes and next steps), which needs `explain`.

Grounding, read on 30 October 2026:
- Most command facts already have one home in code, and the front door reads
  them from there:
  - the cobra tree, which `commandSurface` (`internal/surface/cli/surface.go`)
    walks into `[]surface.Command` for the committed snapshot
    `.abcd/development/release/surface.json` (schema 4: commands, flags,
    whether a flag is required, help placement, sentence, successor);
  - each command's one sentence (`internal/core/surface/sentences.go`), which
    `applySentences` sets as cobra's Short and writes into each page's
    `description:`;
  - one worked example per command with a required input
    (`internal/core/surface/examples.go`), set as cobra's Example;
  - closed sets of flag values, which `enumHelp` (`internal/surface/cli/cli.go`)
    renders into help from the same slice the reader checks against.
- What has no home in code is the shape of each filled-in input. The shape is
  a Go type in the owning core package, decoded through `jsonstrict.Decode`
  or a local strict decoder. Its version is a constant there, such as
  `ideate.SchemaVersion = 2`, `release.ChangelogSchemaVersion = 2`,
  `scribe.SchemaVersion = 1` or the lifeboat synthesis versions. Its example
  is written only in the page's prose, and nothing checks it.
- The tree registers these filled-in inputs (a flag ending `-json` that the
  command reads):
  - `ideate record --verdict-json`;
  - `intent prepass --findings-json`;
  - `intent consistency ingest --findings-json`;
  - the intent audit ingest `--verdict-json`;
  - `docs fidelity record --verdict-json`;
  - `launch ship --changelog-json`;
  - `reading ingest --reading-json`;
  - `scribe ingest --scribe-json`;
  - the four disembark synthesis inputs (`--lessons-json`, `--principles-json`,
    `--press-release-json`, `--review-json`);
  - `library ingest --pages-json` and `library ask --page-json`.
- Two generated page blocks already set the pattern this spec repeats:
  - the asking-rules block in `commands/intent.md`, between
    `<!-- generated: asking-rules -->` and `<!-- /generated -->`, written by
    `cmd/asking-sync` and gated by `TestIntentPageAskingBlockIsGenerated`;
  - the brief's surface appendix (`internal/core/surface/appendix.go`), which
    is composed from the same `[]surface.Command` as the snapshot.
- `docs/reference/cli/commands.md` is written by `cmd/abcd-gen-cli-ref` from
  `cli.GenerateReference` and gated by `reference_test.go`.
- `TestCommandPagesMatchTheMenu` (`internal/surface/cli/menu_test.go`)
  already holds pages to the menu.

### Approach

1. **The command facts.** A `surface.Facts` value per command. The facts
   come from the same walk of the cobra tree as the snapshot:
   `commandSurface` (`internal/surface/cli/surface.go`) is extended to return
   both values from one pass. The snapshot's `Command` and `Flag` keep their
   schema-4 shape, so the release guard's comparison with the last tag
   (`changelog.GuardSurface`) is untouched. The facts carry what the snapshot
   leaves out: defaults, allowed values, the usage line, the example and the
   inputs. Each value holds:
   - the path and the one sentence;
   - the usage line;
   - each flag's name, type (bool, string, int, list), default, whether it is
     required, and its allowed values when it takes a closed set;
   - the filled-in inputs the command reads;
   - the worked example.
   A closed set reaches the walk through a cobra flag annotation, set by one
   helper (`enumFlag`) that also writes the help text through `enumHelp`. So
   the help, the reader's check and the facts all read one slice. Every flag
   whose help today calls `enumHelp` moves to the helper. A test holds each
   annotation equal to the slice its reader checks against.
2. **The input shapes.** A registry in the front door
   (`internal/surface/cli/inputs.go`) names, for each filled-in input flag:
   - the command path and the flag;
   - the owning package's Go type and schema-version constant;
   - an example, held as a constant beside the type in the owning core package
     (for instance `ideate.Example`), never in the front door;
   - the owning package's parse-and-validate function, the one its writing
     command calls, run without writing.
   Six inputs carry no schema version today:
   - `intent audit ingest --verdict-json`;
   - `intent prepass --findings-json`;
   - `intent consistency ingest --findings-json`;
   - `docs fidelity record --verdict-json`;
   - `library ingest --pages-json`;
   - `library ask --page-json`.
   Each gets a version constant of 1 in its owning package. For an object
   input, the decoder reads an absent `schema_version` as 1, so every caller
   that works today keeps working, and it refuses any other version. One input
   is not an object: `library ingest --pages-json` is a bare array
   (`[]DistilledPage`). Its entry declares an array root, and its version is
   the registry's alone. `explain` prints "schema_version 1 (the input carries
   none)" for it.
   `surface` derives a field list from the Go type by reflection, starting from
   the declared root (an object's fields, or an array's element): each JSON
   name, its type, whether it is required (no `omitempty`), and a nested
   object's fields at most four levels deep. Field docs come from a
   `doc:"…"` struct tag where the owning package writes one. The registry is
   held whole by a test: every visible flag ending `-json` that the command
   reads is in it, and every entry names a real command and flag. So a new
   input cannot ship without its shape, and an output-only flag is declared as
   such once, in the same table.
3. **`abcd explain`.** A new top-level verb in the agents block.
   - `abcd explain <path…>` prints, in this order, at most 40 lines:
     - the sentence;
     - the usage line;
     - one line per flag (`--name <type> [one of a | b] (required)`);
     - one line per input (`--verdict-json: ideate verdict, schema_version 2;
       abcd explain ideate record --input verdict-json`);
     - the example;
     - one line naming the sub-commands, when there are any.
   - A command whose flags would run past 40 lines prints the first lines and
     then one line saying how many more there are and how to get them
     (`--json`). The test holds the bound for every visible command.
   - `--input <flag>` prints that input's schema version, its field list and
     its example.
   - `--json` prints the facts document. It is the same document as the
     command's committed JSON file (approach 4), cut to the one command.
   - A hidden command (the hook plane) is answered too, marked `(hidden)` on
     its first line, so a refusal on that plane can name `explain` as its next
     step. A path the tree does not have is refused (exit 2), with the
     nearest real path named, the way the staleness note reads paths today. A
     bare `abcd explain` prints its own help.
   - `explain` gets its sentence, its example, its help placement, a plugin
     page (`commands/explain.md`), a brief chapter
     (`04-surfaces/41-explain.md`) with its register row, and the regenerated
     snapshot and CLI reference, so it is wired on both surfaces.
4. **The derived copies.** One generator, `cmd/command-facts-sync`, run by
   `make command-facts-sync`, writes two things from the facts:
   - `docs/reference/cli/commands/<verb>.json`: one file per top-level verb,
     holding that verb's whole subtree. It sits beside the generated CLI
     reference, ships in every checkout and in the plugin, and is not read by
     any host as a command page. The file has a `format_version` (starting at
     1) and the schema version of each input. It carries no binary version, so
     it changes only when a fact changes.
   - A block in each `commands/<verb>.md`, between
     `<!-- generated: command-facts -->` and `<!-- /generated -->`, holding:
     - for each sub-command, its usage line;
     - its inputs, each with its schema version;
     - its example.
     The block's first line names its format version and says the running
     binary's `abcd explain` outranks it. The block goes at the end of the
     page, so the hand-written guidance reads first. A verb with no flags and
     no inputs gets a one-line block. A visible verb with no plugin page
     (`rules`, `spec` and `statusline` today) gets its JSON file but no page
     and no block. Which verbs have a page stays the menu test's rule
     (`TestCommandPagesMatchTheMenu`), so the generator never creates a page.
   Drift is gated the way the asking block is: one test per copy fails, naming
   the file, when the committed bytes differ from a fresh rendering. Both
   tests are in the `PRECHECK_AUDIT_TESTS` set, so `make precheck` and `make
   preflight` both run them.
5. **The prose lint.** A pure function in `internal/core/lint`
   (`CommandPageFacts`) reads each page outside its generated blocks, given
   the facts, and reports a finding with the page and line for each of:
   - an abcd invocation (a fenced line, or a backticked span, starting with
     the binary or `abcd`) whose command path is not a command the tree has,
     hidden commands included;
   - a `--flag` in such an invocation that the resolved command, or a
     persistent flag on its path, does not register;
   - a `"schema_version": N` in a JSON example that is next to an input's
     flag, where N is not that input's version.
   A line carrying `<!-- facts-lint: historical -->` is exempt. That is for
   prose that names an old version on purpose, such as `commands/ideate.md`
   saying that version 1 is refused. A test in `internal/surface/cli` runs the
   function over the committed pages and fails on any finding. The step that
   adds the lint also fixes, or marks, every finding it reports on the pages
   as they stand.
6. **The page's first instruction.** A second generated block,
   `<!-- generated: ask-the-binary -->`, stands straight after each page's H1
   heading. It holds one sentence: before filling in an input for this
   command, run `abcd explain <verb>` (spelled with the page's own binary
   rung); where that answer and this page disagree, follow the binary, since
   the page may be older. The same generator writes it, and a test holds it
   as the first content after the heading on every page in `commands/`. A
   page has no other first instruction.

### Decisions made in this spec

Neither decision was put to the product thinker, because each answers a
mechanism question that the intent's criteria leave open:

- **The per-command JSON lives in `docs/reference/cli/commands/`**, not
  beside the pages in `commands/`. Host loaders read that folder as command
  pages, so an extra file type there is a risk with no gain. The CLI reference
  is already the generated, user-facing copy of the tree.
- **The page blocks carry each input's schema version and a format version,
  not the binary's version.** A binary version changes at every release and
  in every dev build, and would make every page drift at each cut. The
  version an agent needs is the input's. This is the "version on the
  generated page blocks" that the intent's decision 6 claims over itd-111.

### How each criterion is met

- **`explain` in at most 40 lines, from the one source.** Approaches 1 and 3.
  Tests:
  - a tree-wide test runs `explain` on every visible command and sub-command,
    holding each to at most 40 lines;
  - each output's flags, inputs and example equal the command's facts;
  - a hidden path and a misspelt path are refused with exit 2.
- **An input's shape and an example that passes the decoder.** Approach 2.
  Tests:
  - for every registry entry, `explain … --input <flag> --json` names the
    owning package's schema-version constant;
  - its example passes the owning package's parse-and-validate function;
  - a registry test refuses a visible reading `-json` flag with no entry.
- **One build, three copies, preflight fails on drift.** Approaches 1, 3 and 4.
  Tests:
  - the JSON-file drift test and the page-block drift test;
  - a test that `explain --json` for a command is byte-equal to that
    command's slice of its committed JSON file.
- **The prose lint names page and line.** Approach 5. Tests:
  - fixtures with a flag the command lacks, a path the tree lacks, and a
    stale `"schema_version"`, each reported with its line;
  - the historical marker suppresses the finding;
  - the committed pages report none.
- **Each page's first instruction is `abcd explain`.** Approach 6. Test: every
  `commands/*.md` has the ask-the-binary block as its first content after the
  H1, and the block names that page's verb.

Out of scope, recorded elsewhere:
- refusal codes and next steps (itd-2610301428378830);
- the practice run (itd-2610301428390406);
- the root help's `--json` verb listing, which will read these facts
  (iss-2610301320457709); <!-- record-lint: forward-looking -->
- the help's screen layout, including the agents block's `(read
  commands/<verb>.md)` pointer (itd-2610301327282266). <!-- record-lint: forward-looking -->

The `--answers` files that interviews read are not `-json` flags. They are
answers to questions, and their shape is already printed by the interview
that asks them, so they stay outside the registry. An answers file is listed
as an input in `explain` by name only.

## Footprint

- packages: internal/core/surface/facts.go, internal/core/surface/inputs.go, internal/core/surface/sentences.go, internal/core/surface/examples.go, internal/surface/cli/explain.go, internal/surface/cli/inputs.go, internal/surface/cli/surface.go, internal/surface/cli/cli.go, internal/surface/cli/helpgroups.go, internal/core/lint/commandpagefacts.go, internal/core/ideate, internal/core/intent, internal/core/release, internal/core/reading, internal/core/scribe, internal/core/lifeboat, internal/core/docfidelity, internal/core/memory, cmd/command-facts-sync, Makefile, commands, docs/reference/cli, .abcd/development/release/surface.json, .abcd/development/brief/04-surfaces/41-explain.md, .abcd/development/brief/04-surfaces/README.md
- tests: internal/core/surface/facts_test.go, internal/core/surface/inputs_test.go, internal/surface/cli/explain_test.go, internal/surface/cli/inputs_test.go, internal/surface/cli/commandfacts_drift_test.go, internal/surface/cli/commandpage_lint_test.go, internal/core/lint/commandpagefacts_test.go, cmd/command-facts-sync/main_test.go

## Steps

1. The command facts and the input registry
   - needs: none
   - packages: internal/core/surface/facts.go, internal/core/surface/inputs.go, internal/surface/cli/inputs.go, internal/surface/cli/surface.go, internal/surface/cli/cli.go, internal/core/ideate, internal/core/intent, internal/core/release, internal/core/reading, internal/core/scribe, internal/core/lifeboat, internal/core/docfidelity, internal/core/memory
   - tests: internal/core/surface/facts_test.go, internal/core/surface/inputs_test.go, internal/surface/cli/inputs_test.go
2. The explain verb on both surfaces
   - needs: 1
   - packages: internal/surface/cli/explain.go, internal/core/surface/sentences.go, internal/core/surface/examples.go, internal/surface/cli/helpgroups.go, commands/explain.md, .abcd/development/brief/04-surfaces/41-explain.md, .abcd/development/brief/04-surfaces/README.md, .abcd/development/release/surface.json, docs/reference/cli/commands.md
   - tests: internal/surface/cli/explain_test.go
3. The per-command JSON, the page blocks and the first instruction
   - needs: 1
   - packages: cmd/command-facts-sync, Makefile, commands, docs/reference/cli/commands
   - tests: internal/surface/cli/commandfacts_drift_test.go, cmd/command-facts-sync/main_test.go
4. The prose lint, with the pages made to pass it
   - needs: 3
   - packages: internal/core/lint/commandpagefacts.go, commands
   - tests: internal/core/lint/commandpagefacts_test.go, internal/surface/cli/commandpage_lint_test.go
