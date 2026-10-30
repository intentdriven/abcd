---
id: spc-2610301431108175
slug: when-abcd-refuses-an-agent-s-command-the
intent: itd-2610301428378830
origin: researcher-authored
production_mode: hand-written
---
# abcd tells an agent what to do next, after a refusal and after a success

## Summary

Delivers itd-2610301428378830. Every refusal gets a registered code, a
one-line hint and the next command to try. The `--json` error envelope
carries all three, and `abcd explain <code>` prints the code's explanation.
Every successful `--json` result names what usually comes next. Both are read
from tables held in code, and tests hold them whole. Codes follow
[adr-2610301638558260](../../decisions/adrs/2610301638558260-a-published-refusal-code-is-never.md):
a code is a short name, and once a release has carried it, it is never
renamed or reused.

This spec builds on spc-2610301431075964, which adds `abcd explain` and the
command facts. `abcd build` reads `- needs:` only within one spec, so the order
is stated here: no step of this spec starts before that spec's steps 1 to 3
have landed. Steps 1 and 6 extend `explain` and the facts, and every step
touches `internal/surface/cli/cli.go`, which that spec and
spc-2610301431131636 also change. The order across the three is: that spec's
step 1, then spc-2610301431131636, then that spec's steps 2 to 4, then this
spec.

Grounding, read on 30 October 2026:
- Every refusal leaves through one place. `cli.run`
  (`internal/surface/cli/cli.go`) takes the error `root.Execute()` returns
  and reads its exit code from `exitError` (or anything with `ExitCode()`).
  It then scrubs paths, withholds operands, masks control characters and, under
  `--json`, writes `errorEnvelope{abcd: "error", error, exit_code}` to
  stdout. Otherwise it writes one `abcd: <msg>` line to stderr. Under
  `--json`, stdout is a stream of documents, and the refusal is always the
  last one.
- The front door builds `exitError` literals in 64 files. Counted by literal:
  432 at exit 2 (refused), 5 at exit 3 (held by someone else) and 60 at exit 1
  (a fault). Another 17 take their code from a variable, such as
  `exitError{Code: res.ExitCode}`, so no reading of the source can tell what
  they are. Most read a core error and wrap it in a per-verb helper, such as
  `captureRefusal`, `implementRefusal`, `landRefusal`, `memoryRefusal` or
  `checklistRefusal`.
- Usage errors (an unknown command or flag, the wrong number of arguments, a
  flag-group conflict, a missing required flag) are set to exit 2 tree-wide
  by `markUsageErrorsExitTwo`. On the hook plane, `failOpenNoArgs` and
  `failOpenFlagError` move them to exit 1.
- Successful `--json` output goes through `render`
  (`internal/surface/cli/cli.go`) at about 220 call sites, plus 8 direct
  encoders. Each verb's result is its own struct. Several already carry a
  top-level `next` of their own: `drain`, `build` and `implement` (the loop's
  next action), `intent drain`, `update news`, `lab`, `ahoy`'s setup report,
  `site` and the board. Some verbs stream several documents, such as the
  build loop's dispatch.
- Next moves exist today only for a record id (itd-121): `record.Describe`
  returns `next_moves`.

### Approach

1. **The refusal registry.** A new core package, `internal/core/refusal`,
   holds `type Code string` and one table of entries. Each entry has:
   - the code (`<verb>/<reason>`, lower-case kebab-case, such as
     `capture/remedy-missing`, or `usage/…` for the framework's refusals);
   - a summary of one line;
   - an explanation of a few lines;
   - a next-command template (`abcd capture "<text>" --remedy "<fix>"`), or
     the explicit word `none` where no command can help (for example, "held
     by a peer: wait");
   - an optional `retired` release tag.
   A core error carries a code by wrapping: `refusal.Wrap(code, err)` returns
   an error whose `refusal.CodeOf(err)` finds the code through `errors.As`.
   So a core package names the code where the refusal is decided, and the
   front door adds nothing.
2. **The front door carries it out.** `exitError`'s fields become unexported,
   and it gains a refusal code. It is built only through two constructors:
   - `refused(exit, ref, msg)` for exit 2 or 3, where `ref` is a registered
     code;
   - `exitWith(exit, ref, msg)` for a code known only at run time (the 17
     variable-code sites). Its `ref` argument is required, and it may be
     `refusal.None` only when the exit turns out to be 1. An exit of 2 or 3
     with `refusal.None` is still returned, but carries the code
     `internal/uncoded`. The test-only check below fails on that code, and so
     does the registry test, because the code is never registered.
   A test parses the package and refuses any composite literal of
   `exitError` outside the two constructors. Before rendering, `run` takes the
   code from the error, or else from `refusal.CodeOf(err)`, or else from the
   usage classifier (approach 3). Then:
   - The `--json` envelope gains `code`, `hint` and `next`. `abcd`, `error`
     and `exit_code` stay exactly as they are, so a reader of today's
     envelope is not broken.
   - The text form adds one line after today's: `abcd: <hint> — try: <next>
     (abcd explain <code>)`.
   - A next command is filled from the refused call's own command path. It
     is never filled from the caller's arguments, which `run` withholds today
     and goes on withholding.
3. **Usage errors.** One classifier in `run` reads cobra's and pflag's error
   types (and, where they have none, the wording that `withholdOperands`
   already parses) and maps each to a `usage/…` code:
   - `usage/unknown-command`;
   - `usage/unknown-flag`;
   - `usage/missing-argument`;
   - `usage/extra-argument`;
   - `usage/flag-conflict`;
   - `usage/missing-required-flag`;
   - `usage/bad-flag-value`.
   The next command is `abcd explain <the path that resolved>`. On the hook
   plane, that path is a hidden command, which `explain` answers marked
   `(hidden)` (spc-2610301431075964, approach 3).
4. **What counts as a refusal.** It is an exit 2 or an exit 3: abcd judged
   the call and declined it. An exit 1 is a fault: abcd could not do what it
   meant to, such as a read that failed or a host that closed the pipe. A
   fault is not coded by this spec. There is one exception: the hook plane's
   usage refusals. They are usage refusals that the fail-open rule moves from
   exit 2 to exit 1, so they carry their `usage/…` code at exit 1. Where one
   of today's other exit-1 literals is really a refusal, it moves to exit 2
   with a code, and the step that moves it names it in its pull request.
5. **The gates.**
   - Every code that is raised is registered: the literal test from
     approach 2, plus a test over `internal/core` that every `refusal.Wrap`
     names a registered constant.
   - Every refusal carries a code: a test-only check in `run` fails the test
     binary when an exit 2 or 3 leaves without one. The tree-wide refusal
     tests that already run every verb's usage and refusal paths (for example
     `TestEveryParentRefusesAnUnknownSubverb` and the per-verb refusal tests)
     exercise it.
   - Every code resolves: each registered code is answered by `abcd explain
     <code>`, and each next template names a command path the tree has.
   - No two entries share a code: a uniqueness test over the registry, run in
     the merge queue like every test. This is the test the ADR names for two
     branches that pick the same name.
   - No code is renamed or reused: `cmd/refusal-codes-sync` writes
     `.abcd/development/release/refusal-codes.json` from the registry, and a
     drift test holds it. The release guard
     (`internal/core/changelog/guard.go`, beside `surfaceSnapshotAt`) reads
     the ledger at the last tag the way it reads the surface snapshot, and
     refuses a cut in which a code that ledger carries is missing (a retired
     code is still present). The code list never shrinks.
   - While the migration runs, `.abcd/development/release/refusal-uncoded.txt`
     lists the files whose literals are not yet coded. The literal test
     exempts only those files, and the list may only shrink. The last step
     deletes the file and its exemption. The intent cannot ship while the file
     exists, and the close's fidelity review is told to check for it.
6. **`abcd explain <code>`.** `explain` takes a code wherever it takes a
   command path: an argument containing a `/` is a code. It prints:
   - the summary;
   - the explanation;
   - the next command;
   - the command that raises the code;
   - for a retired code, the tag it was retired in.
   `--json` prints the entry. An unknown code is refused as
   `explain/unknown-code`, naming the nearest code.
7. **Next steps on success.** The command facts of spc-2610301431075964 gain
   a `next` list per visible command, held in one table
   (`internal/core/surface/next.go`). Each item is a command template and one
   line saying when it applies, such as `abcd capture resolve <iss-N>` with
   "once the fix lands". An entry may be the explicit empty list with a
   reason, such as `abcd version`: "nothing usually follows". That is how
   such a command names what comes next, and the reason is shown by `explain`.
   Under `--json`, a successful command's result gets a top-level
   `next_steps` key carrying the list. It is added inside `render`, where the
   document is written, so `run`, the verbs' structs and the byte order of
   every existing key stay exactly as they are:
   - A `PersistentPreRunE` on the root wraps the command's stdout in a writer
     that carries the command path. `render`, given that writer, looks up the
     path's `next` list in the facts.
   - `render` encodes the result to bytes as today. When the document is a
     JSON object, it splices `,"next_steps":[…]` in before the final `}`.
     There is no decode and re-encode, so no key moves and no number changes.
   - The key is `next_steps` everywhere, never `next`, because many verbs
     already use `next` for something else (the grounding lists them). So no
     existing key is replaced or shadowed. A test fails if a result already
     carries a top-level `next_steps`, which is checked on the encoded bytes
     rather than kept as a hand-written list.
   - A command whose facts mark it as streaming (the build loop's dispatch,
     `history drain`) gets the key only on the document it writes through
     `renderFinal`, its closing summary. Every other command renders one
     document, and that document gets the key.
   - The 8 direct encoders move onto `render`, or onto `renderFinal` for a
     streaming summary.
   - The hook plane (hidden commands) is left alone.
   - A command whose `--json` document is not an object (a bare array)
     cannot take a key without breaking its readers. Step 6 first lists every
     such command. If it finds any, the step stops and the product thinker is
     asked how such a command should name its next step. Until then, nothing
     is wrapped.
   The facts table is the one source, so `explain`, the per-command JSON and
   the page block show the same next steps that a result carries.

### How each criterion is met

- **Every refusal carries a code, a hint and the next command, and the code
  resolves.** Approaches 1 to 6. Tests:
  - the literal test, the constructor rule and the `refusal.Wrap` test, so
    every refusal raised in code names a registered code, including the
    variable-code sites;
  - the uniqueness test over the registry;
  - the test-only `run` check over the tree-wide refusal tests, so no exit 2
    or 3 leaves without a code;
  - an envelope test, where a refused `capture` with no `--remedy` has
    `code: capture/remedy-missing`, a hint, and `next` naming `abcd capture`;
  - an `explain` test over every registered code;
  - the next-template test;
  - the ledger drift test, and a release-guard test that removes a code from
    a fixture ledger and is refused.
- **Every successful result names what usually comes next, and each step is
  a real command.** Approach 7. Tests:
  - a tree-wide test that every visible command has a `next` entry;
  - every template names a command path the tree has;
  - a successful `--json` run of a fixture verb carries that verb's
    `next_steps`, with every other byte of the document unchanged;
  - a verb whose own result already has `next` keeps it as it was;
  - a streaming verb carries `next_steps` only on its closing summary.

Out of scope: exit-1 faults (approach 4); the person's what-next menu under
the board (itd-2610031215002409), which reads the same facts once both exist;
`next_moves` for a record id (itd-121), which stays as it is.

## Footprint

- packages: internal/core/refusal, internal/core/surface/next.go, internal/surface/cli/cli.go, internal/surface/cli/usage_errors.go, internal/surface/cli/explain.go, internal/surface/cli, internal/core, cmd/refusal-codes-sync, internal/core/changelog, .abcd/development/release/refusal-codes.json, .abcd/development/release/refusal-uncoded.txt, commands/explain.md, .abcd/development/brief/04-surfaces/41-explain.md
- tests: internal/core/refusal/refusal_test.go, internal/surface/cli/refusal_codes_test.go, internal/surface/cli/refusal_envelope_test.go, internal/surface/cli/next_steps_test.go, internal/core/changelog/refusal_ledger_test.go

## Steps

1. The registry, the envelope, `explain <code>`, the ledger and the shrinking exemption list
   - needs: none
   - packages: internal/core/refusal, internal/surface/cli/cli.go, internal/surface/cli/explain.go, cmd/refusal-codes-sync, internal/core/changelog, .abcd/development/release/refusal-codes.json, .abcd/development/release/refusal-uncoded.txt, commands/explain.md, .abcd/development/brief/04-surfaces/41-explain.md
   - tests: internal/core/refusal/refusal_test.go, internal/surface/cli/refusal_codes_test.go, internal/surface/cli/refusal_envelope_test.go, internal/core/changelog/refusal_ledger_test.go
2. Usage errors on every command, hook plane included
   - needs: 1
   - packages: internal/surface/cli/cli.go, internal/surface/cli/usage_errors.go
   - tests: internal/surface/cli/refusal_codes_test.go
3. The record verbs' refusals: capture, intent, spec, decide, ideate, reserve, announce, peers, status, embargo and banlist
   - needs: 1
   - packages: internal/surface/cli, internal/core/capture, internal/core/intent, internal/core/spec, internal/core/ideate, internal/core/claim
   - tests: internal/surface/cli/refusal_codes_test.go
4. The delivery verbs' refusals: build, implement, drain, land, launch, ship, checklist and docs
   - needs: 1
   - packages: internal/surface/cli, internal/core/implement, internal/core/landing, internal/core/launch, internal/core/release
   - tests: internal/surface/cli/refusal_codes_test.go
5. The remaining verbs' refusals, and the exemption list deleted
   - needs: 2, 3, 4
   - packages: internal/surface/cli, internal/core, .abcd/development/release/refusal-uncoded.txt
   - tests: internal/surface/cli/refusal_codes_test.go
6. Next steps on success
   - needs: 1
   - packages: internal/core/surface/next.go, internal/surface/cli/cli.go, internal/surface/cli/explain.go
   - tests: internal/surface/cli/next_steps_test.go
