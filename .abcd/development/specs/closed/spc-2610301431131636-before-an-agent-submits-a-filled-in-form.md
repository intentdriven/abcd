---
id: spc-2610301431131636
slug: before-an-agent-submits-a-filled-in-form
intent: itd-2610301428390406
origin: researcher-authored
production_mode: hand-written
---
# An agent can practise a filled-in form before it counts

## Summary

Delivers itd-2610301428390406. Every command that reads a filled-in input
and writes from it takes `--dry-run`. A practice run reads the input, judges
it through the same validator the real write uses, and reports one of two
things:
- what it would write: each path, whether it would be created, replaced or
  moved, its size, and the record a single-record write would produce;
- the refusal the real write would give, with the same message and exit 2.

It writes nothing anywhere: not the working tree, the local tier, abcd's home,
git's refs or index, or the remote. A tree-wide test proves both halves.

Grounding, read on 30 October 2026:
- A filled-in input is a flag ending `-json` that the command reads, the same
  definition criterion 2 of itd-2610301346085246 uses. The tree registers
  these on writing commands:
  - `ideate record --verdict-json`;
  - `intent prepass --findings-json`;
  - `intent consistency ingest --findings-json`;
  - the intent audit's `ingest --verdict-json`;
  - `docs fidelity record --verdict-json`;
  - `launch ship --changelog-json`;
  - `reading ingest --reading-json`;
  - `scribe ingest --scribe-json`;
  - the disembark synthesis verbs' `--lessons-json`, `--principles-json`,
    `--press-release-json` and `--review-json`;
  - `library ingest --pages-json` and `library ask --page-json`.
- `--dry-run` is already the tree's name for "show what a run would do and do
  nothing", with that meaning on:
  - `ahoy worktree prune`;
  - `drain`;
  - `reading assemble`;
  - `scribe assemble`;
  - `launch`, where it is a flag of `launch` itself, not of `launch ship`;
  - `ahoy`, where it prints the detection envelope.
  So the practice run takes that name rather than a new one.
- Each writing command already decodes and validates before it writes, and
  refuses atomically when the input is wrong. For example, `ideate.Record`
  parses and checks the payload, then resolves every citation, and only then
  writes; a refused verdict leaves nothing behind. What is missing is a way to
  stop after the checks and report.

### Approach

1. **Split check from write, in core.** Each owning core function gets a
   `Plan` stage that does everything up to the first write: decode, validate,
   resolve every reference the real write resolves, run the redaction it
   applies, and compose the bytes it would write. Plan returns two lists:
   - planned writes: path, action (create, replace or move), size and content;
   - planned effects that are not files: a tag, a push, a forge release, or
     a write to the remote, each with its target (`would tag v0.16.0`,
     `would push refs/tags/v0.16.0 to origin`).
   The real call becomes
   `Plan`, then `Apply(plan)`. So the practice run and the real write share
   every check by construction, not by copying. Checks that today run between
   writes move into Plan. Where a check cannot run before an earlier write (a
   later step reads what an earlier one wrote), the plan works on an
   in-memory overlay of its own planned writes, never on disk.
2. **The flag.** Each command from the grounding list gains `--dry-run`:
   "check the input and report what would be written; write nothing". With
   it, the front door calls Plan and renders the result:
   - Text: a "practice run" heading, one line per planned write
     (`would create .abcd/…/x.md (2.1 KB)`), and for a single-record write the
     record's frontmatter and first lines.
   - `--json`: the verb's usual result fields as the real run would fill
     them, plus `"dry_run": true`, `would_write: [{path, action, bytes}]` and
     `would_do: [{effect, target}]`.
   A refused input gives exactly the real run's refusal: same message, same
   exit code, and the same refusal code once itd-2610301428378830 lands. The
   page block of itd-2610301346085246 lists the flag like any other, with
   nothing extra.
3. **Commands that reach out.** A practice run reads the network only where
   the real write's checks read it, and never writes to it. `launch ship
   --dry-run` judges the changelog payload against the cut it would make, and
   lists the tag, push and release under `would_do`. It pushes nothing, tags
   nothing and creates no release. It reads the baseline already in the
   checkout. `--fetch-baseline` with `--dry-run` is refused (exit 2), because a
   fetch writes `.git/FETCH_HEAD` and pack files; where no local baseline
   exists, the practice run says the real run would need `--fetch-baseline`.
   The existing rule that `--fetch-baseline` needs a staged payload directory
   is unchanged. `library
   ask --file-back --dry-run` reports the page it would file back and files
   nothing; `library ingest --dry-run` reads its source as the real run does
   and stores nothing.
4. **The tree-wide test.** One test enumerates the live command tree. For
   every visible command with a flag ending `-json` that it reads (the same
   set the input registry of itd-2610301346085246 holds, once that lands),
   it:
   - fails if the command has no `--dry-run`;
   - with a valid fixture input in a temporary repository and a temporary
     home, first runs one command that writes nothing (`abcd version`). Every
     invocation narrows abcd's home and lays its ignore file on its first run,
     so the hashes start from that settled home. The test then hashes every
     file under both, plus the git directory (refs, index, objects, config,
     hooks), runs the practice run, and asserts that every hash is unchanged
     and that the output lists at least one planned write or effect;
   - repeats with an invalid fixture, and asserts the practice run's exit
     code and message equal the real run's, and that nothing changed;
   - runs the real write on the valid fixture, and asserts the paths it
     wrote equal the practice run's `would_write`.
   The fixtures are per command, under `internal/surface/cli/testdata/dryrun/`.
   They do not depend on the examples itd-2610301346085246 adds.

**Build order.** This spec shares the owning core packages,
`internal/surface/cli/cli.go`, the snapshot, the CLI reference and the command
pages with its two siblings, and `abcd build` refuses runs whose footprints
overlap. It is built second: after step 1 of spc-2610301431075964 (the input
registry, whose six new version constants change the same decoders), and
before that spec's steps 2 to 4 and spc-2610301431108175.

### Decisions made in this spec

Neither decision was put to the product thinker, because each answers a
mechanism question that the intent's criterion leaves open:

- **A "filled-in input" is a `-json` flag**, the wording the sibling intent's
  criterion 2 uses. An interview's `--answers` file is answered through the
  interview that asks it, and stays outside this spec.
- **The practice run is `--dry-run`**, the name six commands already give the
  same meaning.

### How each criterion is met

- **Same validator, a report of what would be written or why not, and
  nothing written.** Approaches 1 to 4. Tests:
  - the tree-wide test's hash check (the working tree and every abcd store
    byte-identical);
  - its refusal-parity check (same validator, same refusal);
  - its write-parity check (the planned writes are the real writes);
  - per-command unit tests on each owning package's Plan, including
    `ideate` with an unresolvable citation, refused in Plan before any write.

Out of scope: a practice run for commands that take no filled-in input
(several already have `--dry-run`); interview `--answers` files.

## Footprint

- packages: internal/core/ideate, internal/core/intent, internal/core/docfidelity, internal/core/release, internal/core/reading, internal/core/scribe, internal/core/lifeboat, internal/core/memory, internal/surface/cli/ideate.go, internal/surface/cli/intent_prepass.go, internal/surface/cli/intent_consistency.go, internal/surface/cli/docfidelity.go, internal/surface/cli/ship.go, internal/surface/cli/reading.go, internal/surface/cli/scribe.go, internal/surface/cli/cli.go, commands/ideate.md, commands/intent.md, commands/docs.md, commands/launch.md, commands/reading.md, commands/scribe.md, commands/disembark.md, commands/library.md, .abcd/development/release/surface.json, docs/reference/cli/commands.md
- tests: internal/surface/cli/dryrun_test.go, internal/surface/cli/testdata/dryrun, internal/core/ideate/plan_test.go, internal/core/intent/plan_test.go, internal/core/release/plan_test.go

## Steps

1. The plan split and `--dry-run` for the records' inputs: ideate, intent prepass, intent consistency, intent audit and docs fidelity
   - needs: none
   - packages: internal/core/ideate, internal/core/intent, internal/core/docfidelity, internal/surface/cli/ideate.go, internal/surface/cli/intent_prepass.go, internal/surface/cli/intent_consistency.go, internal/surface/cli/docfidelity.go, internal/surface/cli/cli.go, commands/ideate.md, commands/intent.md, commands/docs.md
   - tests: internal/surface/cli/dryrun_test.go, internal/core/ideate/plan_test.go, internal/core/intent/plan_test.go
   - landed: c85f673ed4f6
2. The release, reading, scribe, disembark and memory inputs
   - needs: 1
   - packages: internal/core/release, internal/core/reading, internal/core/scribe, internal/core/lifeboat, internal/core/memory, internal/surface/cli/ship.go, internal/surface/cli/reading.go, internal/surface/cli/scribe.go, internal/surface/cli/cli.go, commands/launch.md, commands/reading.md, commands/scribe.md, commands/disembark.md, commands/library.md
   - tests: internal/surface/cli/dryrun_test.go, internal/core/release/plan_test.go
   - landed: 66d21b2d5e4a
