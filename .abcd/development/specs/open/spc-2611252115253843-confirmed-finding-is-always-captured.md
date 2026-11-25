---
id: spc-2611252115253843
slug: confirmed-finding-is-always-captured
intent: itd-2611251831138387
origin: researcher-authored
production_mode: hand-written
---
# confirmed-finding-is-always-captured

## Summary

abcd keeps a list of findings under its own home and holds three things to it (itd-2611251831138387). A session's stop is held once for each state of the list. A lane does not land while the list holds a finding of that lane. A run that ends with one still owed names it, and so does the next session start in that checkout.

Three readings put a line on the list: the fixed block an agent ends its text with, the words of a closing text, and a check still failing at the head a lane offers. They are crude, and the hold and the landing read the list and not the readings, so a better reader can replace one and change neither.

It is built as three steps, each landing by itself:

1. **The rule, the list and its commands.** The ISSUES line, the store, `abcd capture owed`, the one command that files a fixed finding, the board's count. It needs nothing that is not on the main branch.
2. **Closing texts are read and the stop is held.** It needs the sub-agent start hook (itd-2611251725190796, branch `build-hook-1125`) on the main branch first.
3. **A landing is refused and a run's end says what is owed.** Then the close.

What a change itself breaks and mends on the way is building, and is never captured or listed (dcn-2611251839563777). Nothing in this spec reads a failing test as it happens, so there is nothing to filter out: a check is read only at the head a lane offers.

### What it uses of the branch `build-hook-1125`

Read at 1a85b62b8. Four seams, all in step 2:

- `subagentConventions` (`internal/surface/cli/hook_subagent_start.go`) says whom the start hook tells anything. Where it says nothing applies, this spec tells nothing and reads nothing: the four cold-reading agents, an unmanaged repository, rules switched off.
- `composeSubagentStart` and `subagentStartDoc` hold the start message to `rules.InjectionBudgetBytes`. The line and the block instruction are two more parts of that document, placed straight after the conventions pointer; a test holds the message with both whole under the bound for the longest conventions file that branch's tests use.
- `keepSubagentLine` and `subagentStopLine` (`internal/surface/cli/hook_subagent.go`) are where a sub-agent's stop is already read. The findings reading is one call beside them, with its own error.
- `history.NoteSubagentLine` (`internal/core/history/subagentreading.go`) takes a closed kind. "Findings not judged" is one more kind; its line is listed by `abcd history staged` and counted at session start with the others.

No hook script and no `hooks/hooks.json` entry changes, so the manifest's pins and the shim lists stay as that branch leaves them. Step 1 and the code of step 3 do not need the branch. Step 3 is ordered last because a landing that refuses over a list nothing yet fills has nothing to prove it.

### Approach

**The store.** `internal/core/findings`, a new package, holds one small JSON file for each finding under `~/.abcd.noindex/findings/<root-sha>/`, in `open/`, `private/` or `settled/`. The home is `abcdhome.MachineHome()`, so an agent of a drive's stage writes to the person's home and not the stage's. Directories are 0700, files 0600, opened without following a link, and judged by the owner and mode checks the away store makes (`fsutil.EnsureHomeScope` makes them; the away store's own file is not shared, since its cap and its lock are its own). A file is written once, whole, by rename. Settling writes the settled file and then removes the open one, so a crash between leaves both and the settled one wins. No lock is taken: two writers of one finding write the same name.

The name is a hash of the kind, the normalised text and the place (the run and lane, or the checkout), so a finding said again is the same file and the same state. It is shown as `f-` and eight characters; a shorter or ambiguous one is refused. An entry holds the session, the agent and its type, the checkout's root, the run and lane where the transcript names a lane's brief, the kind, the text and the time. Text is cut to 600 characters, cleaned by `termsafe` and `fsutil.RedactHome`. Nothing is removed by age, by a new session or by the worktree going.

The kinds are closed: `untested`, `handed-up`, `doubt` (owed a capture), `line` (a line to answer), `check` (still failing at the offered head), `known` (said to be known, and the record does not answer it), `private`.

**Which entries a session answers for.** Those it put, and those its sub-agents put, by the session id the payload carries. An entry of an earlier session in the same checkout is named at the next session's start and counted on the board, and refuses that checkout's landing; it does not hold another session's stop, since that session did not meet it.

**The block.** A fence whose info string is `abcd-findings`, so it is neither a `Findings` heading nor a `Verdict` heading and the loop's and the landing's readers of a return pass it by. One line for each finding:

```text
captured iss-<N>: <one line>
settled <record>: <one line>
refuted: <one line>; test: <what was run>
private
handed-up: <one line>
untested: <one line>
doubt: <one line>
none
```

`captured`, `settled` and `refuted` put nothing on the list when their record holds (below). `private` puts nothing: its text goes by command, and the line is the state alone. A fence that does not parse puts one `line` entry naming the first bad line. Only the last such fence in the text is read.

**The words.** Read line by line outside every fence, quote and code span, whole words, no case: pre-existing, unrelated, known, flaky, out of scope, skipped, worked around, left for later, deferred, follow-up, tracked by, accepted, carried to a later change, with their plain inflections ("skip", "skips", "defer", "works around"). A line that names an open issue, an open spec, or a draft or planned intent is passed, and so is one naming a listed handle or a settled record of the block's own kind. A line saying a test or a check "still fails" or "is still failing" is a `check` entry. Every other match is a `line` entry.

**The records (criterion 12).** One resolver, `recordid.NewResolver`, asked in the payload's checkout, the session's start copy and the repository's other worktrees, since a capture may be on a branch. `captured iss-N` holds when the issue is open anywhere, or resolved in this change and not already resolved on the default branch. `settled` holds for an open issue, a wontfix, an ADR or a decision. Anything else lists the finding as `known`, naming the state the record is in.

**The commands.** One new sub-verb, `abcd capture owed`. Bare, it lists and writes nothing. With one or more handles and one state flag it settles them: `--captured iss-N`, `--settled <record>`, `--refuted "<test>"`, `--own "<run on the base>"` (a `check` entry alone), `--not-a-finding "<reason>"` (a `line` entry alone), `--private`. `abcd capture owed --private -` with no handle adds one from standard input. A `doubt` is cleared by `--captured` or `--settled` alone. Every reason and run given is kept in the settled file.

The text of a private entry is printed only where `connectIsTerminal` holds for the input and the output, none of `driveSessionMarkers` is set, and the process is in no stage; anywhere else, and in `--json` always, the count alone. A private entry holds no stop and no landing, and only a person in such a terminal settles it further.

`abcd capture "<text>" --remedy … --fixed "<what fixed it>" --impact <impact> --grounds "<grounds>"` writes the record into `resolved/` in one command: capture's and resolve's own checks are all made first, then both writes happen under one ledger lock, so a refusal writes nothing. The commit it names is the change's own, so the stamp is left out as `capture resolve` leaves it out today, and RS001 accepts the trailer, as it does for a record captured and resolved in one change (`scripts/check-issue-resolution-cases.sh`).

**The sub-agent's start.** The ISSUES line and the block instruction. An agent whose shipped definition fixes its reply as one document is given the line and not the block, from a list in code: graveyard-interpreter, intent-auditor, lifeboat-reviewer, planning-interviewer, press-release-composer, principle-distiller, question-drafter, reflection-composer, release-changelog-composer, scribe. A test over `agents/*.md` holds that list to the definitions, so a new one fails until it is placed.

**The sub-agent's stop.** The closing text is the payload's `last_assistant_message`, or the last assistant text of the sub-agent's transcript where the field is absent. The text of an agent told of the block is read for the block and the words; any other's is not read at all. The stop is never held. Unreadable is one "findings not judged" line.

**The session's stop.** In `hook_stop.go`, straight after the session's copy is known and before the owed read, so an owed read that fails or is cut short changes nothing here:

- the session's own closing text is read and its entries put;
- `autonomy.JudgeFindings` folds the list into the decision as `JudgeChecklist` folds the checklist. A state not yet held for is a block naming each finding and its clearing act, twenty at most and then the count. The state held for is kept in `SessionState.FindingsHeld`;
- while away, the reason is appended to the away check's own, one answer, and counts toward the same cap;
- while present, this block is the only thing that can hold a stop, and its read is the disk alone;
- a state already held for allows the stop with one line for the person: the count and `abcd capture owed`;
- a list that cannot be read allows it and says so.

Within one chain of held stops (`stop_hook_active`) the list holds at most `DefaultBlockCap` times, whatever it gains on the way, and then allows with the line above. A stop in an unmanaged repository, or where the rules are off, is not touched. In a drive's stage the away check still returns at once, as today, and the list alone is read, put and held.

**The prompt.** A finding a sub-agent left is read by the session at its next prompt too, through the note path the checklist uses (`checklistPromptNote`), once for each state.

**The landing.** `internal/core/landing/findingshold.go` holds the lane's list and is called from `internal/surface/cli/land.go` before `landing.Land`, so `internal/core/landing/land.go` is not edited. It refuses naming each finding and its clearing act, with nothing pushed. It also wraps `Options.Gate`: a preflight that fails at the offered head puts one `check` entry, and one that passes there settles it, so a check that failed on the way and passes at the head leaves nothing.

A lane's list is the entries whose checkout is the branch's checkout, those naming its run and lane, and those the landing session met in no linked worktree (the main checkout), since no other lane will answer for them.

For a build or a drive, `findingsHeld` is called from `performMove` in `internal/core/implement/loop/schedule.go`, in the switch that already widens a lane's hold before a step, on a lane at `StageLand` whose landing has pushed nothing. So the lane is refused before its sync and its preflight are spent. It is a contention refusal, which `mayWait` already lets a landing be, so the same call goes on to the next lane; under `claim.DriveMarked()` it is also final, so a drive hands the lane back as stuck at its next move, with its worktree and its branch kept, and takes up the next issue.

**The end.** `abcd implement leave` and the drive's end print each finding still owed, after what they print today. The next session start in that checkout names each, twenty at most and then the count. The board carries one line with the count and the command, and no line at none. A handover written by hand is not read.

### How each criterion is met

1. The domain's line: `internal/core/rules/defaults/rules.json` (step 1). The start's line: `composeSubagentStart` (step 2). One constant in `internal/core/findings` is the text of the second, and a test holds the domain's line equal to it.
2. The words "still fails" (step 2) and the gate wrapper (step 3) list it; a gate that passes at the head settles it (step 3); `--own` takes it off naming the run, `--captured` clears it as a finding in what has landed (step 1). abcd runs nothing on a base. A test fails, then passes, then lands, and the store is empty.
3. The word reading (step 2); cleared by a state flag or `--not-a-finding`, whose reason stays in the settled file (step 1).
4. `doubt` in the block (step 2); the settle side refuses every flag but `--captured` and `--settled` for it (step 1).
5. The block instruction in the start message, and the two exclusions (step 2).
6. The sub-agent stop (step 2). The live check is below.
7. `JudgeFindings`, the state field and the line for the person (step 2).
8. The landing hold, the loop's, and the reports at the end (step 3). `--captured` with a record saying which test is owed clears one that cannot be tested; the list does not judge the record's words.
9. `abcd capture … --fixed` (step 1). A fix outside the lane's hold is refused by the hold as today, and the ISSUES domain gains a second line: capture it and leave it open (step 1).
10. The store's own rules and `capture owed` (step 1), the board (step 1), the session-start notice (step 2).
11. `private/`, the terminal test and the count everywhere else (step 1); the renders to a session (step 2) and the landing's and the end's (step 3) each carry the count alone. Each has a test that plants a private text and searches every output for it. The embargo store is neither read nor written.
12. The resolver, at the settle side (step 1) and at the block (step 2).

### The live check

Criterion 6 asks for one live check on the installed host. In step 2, one session starts one sub-agent that ends with a known sentence in its block. The entry then stands on the list, and the hook's own log of its payload names the field the text came in. The dated receipt goes in the main checkout's `.abcd/.work.local/logs/findings-live-<date>.md`, naming the host's version, the payload's field names (not their content) and the entry's handle. Where the field is absent the receipt says so and the transcript path is what carries the criterion, which the writer says in the pull request; the intent's scope condition is then settled at the audit as it stands. The run needs a login and spends one small sub-agent.

### The lists that pin this change

- **A new sub-verb**: `internal/core/surface/sentences.go`, `internal/core/surface/examples.go` (one worked example), `.abcd/development/release/surface.json`, `docs/reference/cli/commands.md`, `docs/reference/cli/commands/capture.json`, the generated block of `commands/capture.md`. Tests: `TestEveryVisibleVerbCarriesItsSentenceEverywhere`, `TestEveryVisibleCommandCarriesAWorkedExample`, `TestEachWorkedExampleIsShapedLikeItsCommand`, `TestPluginSurfaceReachesEveryBinaryVerb`, `TestCommandFactsFilesMatchTheCommittedCopies`, `TestCommandPageBlocksMatchTheCommittedPages`.
- **Refusals**: no new code. The verb raises `capture/invalid-input`, `capture/unknown-issue`, `usage/missing-argument` and `usage/flag-conflict`, so each one's `RaisedBy` list and `.abcd/development/release/refusal-codes.json` gain the verb. Tests: `TestRaisedByIsWhatTheVerbsRaise`, `TestExitErrorIsBuiltOnlyThroughItsConstructors`, `TestEveryNextCommandAndRaiserIsACommandTheTreeHas`. `refusal-uncoded.txt` only shrinks, so every new file in `internal/surface/cli` carries codes from its first line; `land.go`, `implement.go` and `drain_drive.go` are on it and may keep their form.
- **The hooks**: `internal/surface/cli/testdata/hook-git-baseline.txt` gains no line; every git call made by this change takes the hook's context. `TestHookGitCallsMatchBaseline`. The stop's budgets are not raised: the list is read in the first budget and the resolver's git calls share it.
- **The sweep**: `maxSweepWarns` and its sibling ceiling in `internal/core/guard/sweep_test.go` are not raised. No hook script changes; the commands the pages gain are the capture verb's. A rise is a finding about a page's wording and is fixed there.
- **The rules**: the ISSUES domain's render in the standing tests of `internal/core/rules`, and the injection budget: the domain with its new line still fits whole beside the domains it is commonly matched with.
- **The readers of a return**: the landing's verdict reader and the loop's findings and verdict readers are each given a return that ends with an `abcd-findings` fence, and read it as they read it without. These tests come first in step 2.

### Generated files

`go generate ./internal/surface/cli`, `make command-facts-sync`, and the surface snapshot command, as `make precheck` names them, in steps 1 and 3 each. Run twice, the second leaves the tree unchanged. None is edited by hand.

### Decisions made in this spec

Each is technical, follows what has landed, and is the writing session's.

1. A finding met in the main checkout, and placed in no lane, refuses any landing the same session runs. It is the stricter reading, and no lane would ever answer for it otherwise.
2. A drive's stage is held at its stop on the list alone, once for each state. Its agent is the one that can still test what it met; the away check stays skipped there.
3. An entry of an earlier session does not hold a later session's stop. Criterion 7 says "a session with a finding on its list".
4. The loop's refusal is before the landing's first act, in `schedule.go`, and not at the push fence: a landing begun holds its siblings' back, and a preflight would be spent first. Contention is the word the scheduler reads to go on.
5. The sub-verb is under `capture`, where the act that clears a finding already lives, so no top-level verb, no new plugin page and no manifest row is added.
6. One file for each finding and no lock, where the away store takes one: a stop that waited on a lock would spend the host's timeout.
7. A private text goes to the list by command and not in the block, because the host hands a sub-agent's reply to its parent, which is a render to a session.
8. The words reading is line by line. A record named on the next line does not answer, so a false listing costs one answer and a missed one is not silent.
9. A list that cannot be read refuses a landing, where it holds no stop: the landing's other reads already refuse when they cannot tell (`stopHold`), and the refusal names the file.
10. The held stop has a bound within one chain, so that no wording can catch a session in a loop (dcn-2611252038554826).

### What it must not do

- Capture, list or count what a change made and removed before it lands, a failing test on the way among them (dcn-2611251839563777). No hook, gate or reading in this spec watches a test run in an agent's shell.
- Run a check on a base.
- Change the capture verb's own gates or the automatic filers (iss-2610271115000748's build), or require a record of a person's own typing.
- Read or write the embargo store, or change what a capture of category `security` does.
- Hold a sub-agent's stop, or change any hold of the away mode: work owed while present stays allowed and flagged.
- Read a handover or any other file written by hand.
- Edit an agent's definition, a drive's stage wall, or a hook script.
- Add a dependency.

### The checks, and how each is run

- Each test named in a step is run by name, once red on an assertion and once green, under `scripts/abcd.sh implement load --wait --narrow --timeout 2m -- go test -trimpath -count=1 -timeout 7m -run '<names>'` on up to three packages at once. The whole of `internal/surface/cli` is run once, by the preflight.
- `make record-lint` and `scripts/abcd.sh intent ready itd-2611251831138387` after any edit of a record.
- `make precheck` at the end of each step: the gates, the build, the vet, the record and page and surface audits, the refusal registry and the generated files.
- The sweep ceilings and the git baseline by name at the end of each step: `-run TestFalsePositiveSweepCeiling` in `internal/core/guard`, `-run 'TestHookGitCallsMatchBaseline|TestHooks'` in `internal/surface/cli`.
- `make preflight` on the committed head of each step, then a plain push.
- The independent fit-challenge the intent's SOTA section owes is run before step 1 is built, and the review pair on each step's diff as for any lane.

### One branch, and what waits if time runs short

One branch for each step. Step 1 is whole and useful alone: an agent can be asked to keep the list by hand. If time runs short after step 2, the stop is held and a landing is not refused; criteria 2 (its gate half) and 8 wait. The spec stays open, no `Delivers:` trailer is made and the intent stays planned until step 3 is in.

### Limits the code sets

- A finding an agent never mentions is not read (the intent's third scope condition).
- The words reading finds ordinary prose too, "the known cost" among it. The cost is one held stop and one line of reason.
- The host's own transcript holds what an agent typed, a private text passed to a command included, and abcd's staged copy of it is redacted only as any transcript is. The list keeps the text out of what abcd itself renders and commits.
- An agent that settles a finding with a false test or reason is recorded as having done so and is not caught; a person or a reviewer reading a sample is the check, as the Mechanism says.
- A stop whose hook did not run, or was killed by the host's timeout, puts nothing; the landing then has only what earlier stops put.
- The list is an account's, so a finding met on another computer or by another account is not on it.

## Footprint

- packages: internal/core/findings (new), internal/core/capture/workflow.go, internal/core/rules/defaults/rules.json,
  internal/core/autonomy/findings_hold.go (new), internal/core/autonomy/session_state.go,
  internal/core/history/subagentreading.go, internal/core/landing/findingshold.go (new),
  internal/core/implement/loop/findingshold.go (new), internal/core/implement/loop/schedule.go,
  internal/core/surface/sentences.go, internal/core/surface/examples.go, internal/core/refusal,
  internal/surface/cli/capture_owed.go (new), internal/surface/cli/hook_findings.go (new),
  internal/surface/cli/cli.go, internal/surface/cli/board.go, internal/surface/cli/hook_stop.go,
  internal/surface/cli/hook_subagent.go, internal/surface/cli/hook_subagent_start.go,
  internal/surface/cli/land.go, internal/surface/cli/implement.go, internal/surface/cli/drain_drive.go,
  internal/surface/cli/refusal_raisers_test.go, .abcd/development/release/refusal-codes.json,
  .abcd/development/release/surface.json, docs/reference/cli/commands.md, docs/reference/cli/commands/capture.json,
  commands/capture.md, commands/history.md, commands/land.md, commands/implement.md, commands/drain.md,
  .abcd/development/brief/02-constraints/03-invariants.md, .abcd/development/brief/04-surfaces/01-ahoy.md,
  .abcd/development/brief/04-surfaces/06-capture.md, .abcd/development/brief/04-surfaces/08-abcd.md,
  .abcd/development/brief/04-surfaces/11-history.md, .abcd/development/brief/04-surfaces/27-implement.md,
  .abcd/development/brief/04-surfaces/35-drain.md, .abcd/development/brief/04-surfaces/38-land.md,
  .abcd/development/brief/05-internals/03-configuration.md,
  .abcd/development/specs/open/spc-2611252115253843-confirmed-finding-is-always-captured.md,
  .abcd/development/specs/closed/spc-2611252115253843-confirmed-finding-is-always-captured.md,
  .abcd/development/intents/planned/itd-2611251831138387-confirmed-finding-is-always-captured.md,
  .abcd/development/intents/shipped/itd-2611251831138387-confirmed-finding-is-always-captured.md,
  .abcd/work/issues
- tests: internal/core/findings, internal/core/capture/workflow_test.go, internal/core/rules/rules_test.go,
  internal/core/autonomy/findings_hold_test.go (new), internal/core/history/subagentreading_test.go,
  internal/core/landing/findingshold_test.go (new), internal/core/implement/loop/findingshold_test.go (new),
  internal/surface/cli/capture_owed_test.go (new), internal/surface/cli/hook_findings_test.go (new),
  internal/surface/cli/hook_stop_test.go, internal/surface/cli/hook_subagent_start_test.go,
  internal/surface/cli/board_view_test.go, internal/surface/cli/land_hold_surface_test.go,
  internal/surface/cli/implement_surface_test.go, internal/surface/cli/drain_drive_test.go,
  and the standing tests named under "The lists that pin this change"

## Steps

1. The rule, the list and its commands
   - needs: none
   - packages: internal/core/findings/list.go (new), internal/core/findings/list_test.go (new), internal/core/capture/workflow.go, internal/core/rules/defaults/rules.json, internal/surface/cli/capture_owed.go (new), internal/surface/cli/cli.go, internal/surface/cli/board.go, internal/core/surface/sentences.go, internal/core/surface/examples.go, internal/core/refusal, internal/surface/cli/refusal_raisers_test.go, .abcd/development/release/refusal-codes.json, .abcd/development/release/surface.json, docs/reference/cli/commands.md, docs/reference/cli/commands/capture.json, commands/capture.md, .abcd/development/brief/04-surfaces/06-capture.md, .abcd/development/brief/04-surfaces/08-abcd.md, .abcd/development/specs/open/spc-2611252115253843-confirmed-finding-is-always-captured.md
   - tests: internal/core/findings/list_test.go, internal/core/capture/workflow_test.go, internal/core/rules/rules_test.go, internal/surface/cli/capture_owed_test.go, internal/surface/cli/board_view_test.go, internal/surface/cli/refusal_raisers_test.go

   Meets criterion 1's domain half, the settle side of 2, 3, 4 and 12, criteria 9 and 11 (the store and the list command), and 10 but its session-start notice.

   Tests first, in this order. The store: the same finding put twice is one file; a settled one put again stays settled; a crash between the two writes of a settle reads as settled; an entry stands after its worktree is gone and after a year; a link, a foreign owner or a loose mode is refused and named; two writers at once leave one whole file. The resolver: open, resolved on the default branch, resolved in this change, wontfix, ADR, decision, absent. The commands: each state flag on each kind it may and may not clear; two state flags; an unknown and an ambiguous handle; `--json`. Private: planted text is absent from every output with a marker set, with a pipe for output and in a stage, and present in a terminal. The fixed filing: one record in `resolved/`, nothing written on each refusal, and `scripts/check-issue-resolution.sh` accepting the trailer. The domain's line and the board's line, absent at none.

   The order of work keeps the files another account holds to the end: the core package, the capture workflow and the rules line first (none is held), then the verb with its lists in one commit. Stays for the next: nothing yet reads a closing text, so the list is filled by hand and by the tests.

   Done when: every test above was watched fail on an assertion and passes; `make precheck` is clean; the generating commands leave the tree unchanged when run again; `abcd capture owed` in a fresh checkout prints that nothing is owed and makes nothing under abcd's home beyond what every command lays; the step is one pull request with no `Delivers:` trailer.

2. Closing texts are read and the stop is held
   - needs: 1
   - packages: internal/core/findings/read.go (new), internal/core/findings/read_test.go (new), internal/core/autonomy/findings_hold.go (new), internal/core/autonomy/session_state.go, internal/core/history/subagentreading.go, internal/surface/cli/hook_findings.go (new), internal/surface/cli/hook_stop.go, internal/surface/cli/hook_subagent.go, internal/surface/cli/hook_subagent_start.go, internal/surface/cli/cli.go, commands/history.md, .abcd/development/brief/02-constraints/03-invariants.md, .abcd/development/brief/04-surfaces/01-ahoy.md, .abcd/development/brief/04-surfaces/11-history.md, .abcd/development/brief/05-internals/03-configuration.md, .abcd/development/specs/open/spc-2611252115253843-confirmed-finding-is-always-captured.md
   - tests: internal/core/findings/read_test.go, internal/core/autonomy/findings_hold_test.go (new), internal/core/history/subagentreading_test.go, internal/surface/cli/hook_findings_test.go (new), internal/surface/cli/hook_stop_test.go, internal/surface/cli/hook_subagent_start_test.go, internal/core/landing/review_test.go, internal/core/implement/loop/findings_test.go

   It waits for itd-2611251725190796 on the main branch: the branch's start verb, its stop reading and its line kinds are edited here, and nothing is built on a copy of them. Meets criterion 1's start half, 2's words half, 3 to 7, 10's notice, 11's renders to a session and 12 at the block.

   Tests first, in this order. The readers of a return, unchanged by a trailing `abcd-findings` fence. The block: each state, the last fence read, a bad line, none. The words: each phrase, each passed by for a fence, a quote, an open issue, an open spec, a draft, a planned intent; a resolved record not passing it. The start message: the two sentences, the bound, the ten agents without the block, the four told nothing, and the definitions' test. The sub-agent's stop: the entries on the starting session's list, never a block on any input, unreadable kept as a line. The session's stop, each present and away: held once, the same state again allowed with the line, a new finding held again, the chain's bound, an unreadable list allowed with its note, an owed read that failed, rules off, unmanaged, a stage; one answer with the away reason, whose existing tests pass unchanged; no forge call while present. The prompt's note. The start's notice.

   The chapters say the stop rule as it now is: `03-invariants.md` and the configuration chapter's stop section each name the one thing that holds a stop while the person is present. The home's directory tree in `01-ahoy.md` gains its row here rather than in step 1, since the hook branch holds that file until it lands. Stays for the next: no landing reads the list.

   Done when: every test above was watched fail on an assertion and passes; the standing stop tests pass unedited; the live check's receipt is in the local tier; the git baseline and the sweep ceilings are unchanged; `make precheck` is clean; the step is one pull request with no `Delivers:` trailer.

3. A landing is refused, a run's end says what is owed, and the close
   - needs: 1, 2
   - packages: internal/core/landing/findingshold.go (new), internal/core/implement/loop/findingshold.go (new), internal/core/implement/loop/schedule.go, internal/surface/cli/land.go, internal/surface/cli/implement.go, internal/surface/cli/drain_drive.go, commands/land.md, commands/implement.md, commands/drain.md, .abcd/development/brief/04-surfaces/27-implement.md, .abcd/development/brief/04-surfaces/35-drain.md, .abcd/development/brief/04-surfaces/38-land.md, .abcd/development/specs/open/spc-2611252115253843-confirmed-finding-is-always-captured.md, .abcd/development/specs/closed/spc-2611252115253843-confirmed-finding-is-always-captured.md, .abcd/development/intents/planned/itd-2611251831138387-confirmed-finding-is-always-captured.md, .abcd/development/intents/shipped/itd-2611251831138387-confirmed-finding-is-always-captured.md, .abcd/work/issues
   - tests: internal/core/landing/findingshold_test.go, internal/core/implement/loop/findingshold_test.go, internal/surface/cli/land_hold_surface_test.go, internal/surface/cli/implement_surface_test.go, internal/surface/cli/drain_drive_test.go

   Meets criterion 2's gate half, criterion 8, and 11 at the landing and the end.

   Tests first, in this order. The landing: refused with nothing pushed, naming each finding and its act; another lane's entry not counted; an entry met in the main checkout counted; a private one not; an unreadable list refused, naming the file that could not be read. The gate: fails then passes leaves an empty store; fails at the head leaves one check. The loop: the refused lane's sync and preflight do not run, the same call moves the next lane, a drive hands it back with its worktree and branch, a host run goes on refusing until it is cleared, a landing that has pushed is not stopped. The end: `implement leave` and the drive's end with entries and with none, private as a count. Each watched fail first.

   The close, in the same change: a docs review for the head is recorded, the issue and the spec are closed, and the commit carries `Delivers: itd-2611251831138387` and `Resolves: iss-2611251828531764`. The close takes the impact the record carries, `breaking`, which is the person's to change before it. Stays after it: nothing of this intent.

   Done when: every test above was watched fail on an assertion and passes; `make precheck` and `make preflight` are clean; a landing run by hand in a scratch repository is refused on one entry, cleared and then made, and the run's output is in the pull request; the spec is in `closed/` and the intent in `shipped/`.
