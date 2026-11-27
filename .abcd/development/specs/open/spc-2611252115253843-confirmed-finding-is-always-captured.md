---
id: spc-2611252115253843
slug: confirmed-finding-is-always-captured
intent: itd-2611251831138387
origin: researcher-authored
production_mode: hand-written
---
# confirmed-finding-is-always-captured

## Summary

abcd keeps a list of findings under its own home and holds three things to it (itd-2611251831138387). A session's stop is held once when the list gains a finding. A lane does not land while the list holds a finding of that lane. A run that ends with one still owed names it, and so does the next session start in that checkout.

Three readings put an entry on the list: the fixed block an agent ends its text with, the words of a closing text, and a check that fails at the head a lane offers. They are crude, and the hold and the landing read the list and not the readings, so a better reader can replace one and change neither.

It is built as four steps, each landing by itself:

1. **The rule, the list and its commands.** The ISSUES line, the store, `abcd capture owed`, the one command that files a fixed finding, the board's count.
2. **A session's own text is read and its stop is held.** The block, the words, the hold, the notices. It needs nothing of the sub-agent hook.
3. **Sub-agents are told of the block and read for it.** It needs itd-2611251725190796 (branch `build-hook-1125`) on the main branch.
4. **A landing is refused and a run's end says what is owed.** Then the close.

What a change itself breaks and mends on the way is building, and is never captured or listed (dcn-2611251839563777). Nothing here watches a test run in an agent's shell.

### What step 3 uses of the branch `build-hook-1125`

Read at 5740a75e1. Nothing is built on a copy of it; step 3 waits for it to be on the main branch, and steps 1, 2 and 4 do not.

- `subagentConventions` says whom the start hook tells anything. Where it says nothing applies (a cold-reading agent, an unmanaged repository, rules off), nothing is told and nothing is read.
- `subagentStartDoc` and `composeSubagentStart` hold the message to `rules.InjectionBudgetBytes`; the line and the block instruction are two more parts of it, right after the conventions pointer.
- The told fact (`rules.SubagentRecord`) gains one field saying the block was sent. The stop reads it.
- `history.NoteSubagentLine` takes a closed kind; "findings not judged" and "sent no block" are two more, listed by `abcd history staged` and counted at session start with the others.

No hook script and no `hooks/hooks.json` entry changes in any step.

### Approach

**The store.** `internal/core/findings`, a new package, holds one JSON file for each finding under `~/.abcd.noindex/findings/<root-sha>/`, in `open/`, `private/` or `settled/`. The home is `abcdhome.MachineHome()`, so a drive's stage writes to the person's home. Directories are made and opened with `fsutil.EnsureHomeDeclarationScope` and `OpenHomeDeclarationScope`, which judge each level's owner and mode (`EnsureHomeScope` judges neither), and each file is read with `ReadHomeDeclarationDenyingIn`, denying 0o077. Directories are 0700, files 0600. A file is written whole, by rename: two writers of one finding write the same name. A put and a settle judge their bounds and write under the root's one lock file, `.lock`; what is slow runs before it (the scan, and a settle's question to git, checked again under the lock), the wait is the caller's deadline where that is nearer than ten seconds, and a lock that cannot be taken, or an unsafe file at its path, is refused naming the store. A read takes no lock. A put sweeps the stale twins and the temporary files a cut-short write left. Settling writes the settled file and then removes the open one, so a crash between leaves both and the settled one wins. Nothing is removed by age, by a new session or by the worktree going.

| Bound | Value | Past it |
| --- | --- | --- |
| One file | 4 KiB | every field is cut first and the file is written without HTML escaping; a put leaves room for the largest settlement with its reason withheld, so every entry put can be settled; an entry still past that has its text (or a settlement its reason) withheld and is kept, and one past it even so is refused whole; one found is refused as unreadable, named, and a put repeating its name names it and goes on with the rest |
| Entry text, and a settlement's reason | 600 bytes | cut |
| Any other field | 128 bytes, a path 512 | cut |
| A closing text read | 256 KiB | its last part is read, and one entry says so |
| Entries from one closing text | 32 | one more entry holds the count of the rest |
| Open entries for a root | 1,000 | nothing more is put; the stop and the board say the list is full |
| Private entries for a root | 100 | nothing more is put there |
| Files read for one hook | 2,200 (every open and private entry the bounds allow, and as many again for twins and stray files, which count) | the list is unreadable to that hook; `capture owed` names a budget of its own, so it can still list the handles past it |

Each file carries `schema_version: 1`. A later version, and a kind this build does not know, is counted and shown by its handle as "a newer abcd's"; no command here settles it, it holds no stop, and it refuses a landing.

**What an entry holds, and its name.** The session, the agent and its type, the checkout's root, the run and lane, the kind, the text, the time. The name is a hash: for an entry a reading made, of the kind, the normalised text (redacted, before its cut), the place and the session; for one a gate made, of the kind, the place and the head. So a finding said again in one session is the same file, another session's saying of it is its own, and a gate's failure at a new head is a new one. One a person or an agent settled stays settled when it is put again. An entry whose text was withheld is never named by any function of its text: a gate's entry keeps its kind, place and head name; one a reading of a closing text made is named by where it came from (the session, the agent, the reading's position in the transcript, the kind), so a re-read under the same degraded scan is the same entry, and the step-2 reader passes that position; any other is named at random, and a repeat of it is its own entry. A withheld entry holds the stop like any other, and the stop's line says its text was withheld because the full scan could not run and names the repository's scanner configuration (`.abcd/config/pii.json`, and `.abcd/config/gitleaks.json` where it configures the augmenter) as what to fix. A scan counts as not run whole when the scanner cannot be made, runs degraded, degrades during the text (a configured augmenter's run failing), lacks the augmenter the repository configured, or leaves a span its own re-scan of the redacted text still finds; the entry keeps the scanner's sanitised cause. A caller's deadline that passed is its own reason, and its line does not send the person to the configuration. A withheld re-put keeps the cleared text the entry already held, a receipt's file included when it reopens the entry; a cleared reading marks, in `settled/`, the name a withheld reading at its position takes, so the entry withheld there is superseded wherever the cleared one stands and a withheld re-read finds the cleared one. A text whose redaction leaves a span its re-scan still finds is withheld alone, and the redactor stays for the rest of the call. Checkout and place are made valid UTF-8 and safe before they are cut, and an entry records when that changed its checkout; a settle withholds its reason for such an entry, for one whose stored checkout sits at its bound, and for one put from a checkout other than the one the settle is typed in, so a settle reads no other checkout's scanner configuration. It is shown as `f-` and eight characters; an ambiguous handle is refused.

The place is the checkout the payload's directory is in. It is a lane's when a run's state names that checkout as the lane's worktree. It is never read from what a transcript says. Any other entry is unplaced.

Text, and a settlement's reason, goes through capture's own redactor for the entry's checkout, exported from `internal/core/capture/redact.go` as one constructor built once for each checkout a call meets, then `termsafe` and `fsutil.RedactHome`. Where the scanner cannot be made, or runs degraded, the entry is put with its text withheld, and says so. A private entry's text is handed back by one accessor alone; every other read carries it blank.

**The kinds.** Closed, and each is cleared by the flags the table marks.

| Kind | Made by | `--captured` | `--settled` | `--refuted` | `--own` | `--not-a-finding` | `--private` |
| --- | --- | --- | --- | --- | --- | --- | --- |
| `untested` | the block, the command | yes | yes | yes | | | yes |
| `handed-up` | the block, the command | yes | yes | yes | | | yes |
| `doubt` | the block, the command | yes | yes | | | | |
| `known` | a record that does not answer | yes | yes | yes | | | yes |
| `line` | the words, a line of the block that does not parse | yes | yes | yes | | yes | yes |
| `check` | the words "still fails", a gate | yes | yes | | yes | | yes |
| `private` | the command | person | person | person | | | |

"Person" is a terminal as the private text's test below has it.

**The block.** A fence whose info string is `abcd-findings`, with nothing after it but blank lines: the fence ends the text. It is neither a `Findings` nor a `Verdict` heading, so the loop's and the landing's readers of a return pass it by. One line for each finding:

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

`captured`, `settled` and `refuted` put nothing when their record holds; `private` puts nothing, its text going by command. Each line that does not parse is one `line` entry. A fence that does not end the text is not a block, and its text is not read as words.

**The words.** Read line by line outside fences, quotes and code spans, whole words, no case. Two sets:

- alone: pre-existing, flaky, left for later, tracked by, carried to a later change;
- only within 40 characters of a fault word on the same line (failure, fault, bug, defect, error, finding, regression, flake, drift, and their plurals): known, skipped, follow-up, accepted, unrelated, deferred, out of scope, worked around.

"Still fails" and "still failing" of a test or a check make a `check`; any other match makes a `line`. A line is passed when it names an open issue, an open spec, or a draft or planned intent, or a handle of the list. The return of a review role (`ruthless-reviewer`, `security-reviewer`, `docs-currency-reviewer`), and any text with a `Verdict` heading, is read for the block alone.

The bound: at most 6% of sub-agent returns that are not reviews, and 2% of turn ends, gain an entry by words. Measured on 2026-11-25 over the host's stored texts for this repository with a script that applied these two sets: 5.1% of 529 such returns and 1.85% of 16,604 turn ends, none in the median session and three in the ninetieth. The loose reading the first draft had gave 45% of all returns and 10% of turn ends in the same script; a second reader's own count gave 39% and 8%. Step 2 measures its own code again. Over the bound, the second set is cut whole and the criterion says so.

**The records (criterion 12).** The default branch decides. Its listing is one `git ls-tree` of the family's directory at `gitutil.DefaultRefContext`, under the hook's deadline and only when the text names an id, read by `recordid.NewResolverFromPaths`. A record it lacks is looked for with `recordid.LookupOne` in the payload's checkout and the session's start copy, and nowhere else. `captured iss-N` holds for an issue the default branch has open, and for one it lacks that is in `open/` or `resolved/` there (filed in this change, fixed or not). `settled` holds for an open issue, a wontfix, an ADR or a decision. Anything else is a `known` entry naming the state found. With no default branch to read, the checkout answers alone.

**The commands.** One new sub-verb, `abcd capture owed`.

- Bare, it lists the open entries of the whole repository for this account, every checkout and session, this checkout's first, a hundred at most and then the count. It writes nothing. `--settled` lists the last fifty settled ones, each with its act and the reason or run given.
- `--untested -`, `--doubt -`, `--handed-up -` and `--private -` each put one entry from standard input, for this checkout and the session of the environment.
- Handles and one clearing flag settle them, as the table allows. `--own` takes the run on the base; `--not-a-finding` and `--refuted` take their reason or test.

It writes only under abcd's home, so it never calls `refuseSharedMainCheckout`; a test runs it in a main checkout two sessions started in. No new refusal code.

A private text is printed only where the input and the output are a terminal (`connectIsTerminal`), none of `driveSessionMarkers` is set and the process is in no stage. Everywhere else, and always in `--json`, the count alone. A private entry holds no stop and no landing.

`abcd capture "<text>" --remedy … --fixed "<what fixed it>" --impact <impact>` writes the record into `resolved/` in one command, capture's and resolve's checks all made first and both writes under one ledger lock, so a refusal writes nothing. RS001 already accepts a trailer for a record captured and resolved in one change.

**The session's stop.** In `hook_stop.go`, as soon as the session's copy is known and before the owed read, the closing text is read and its entries put. The hold is judged once, and every exit after it goes through it, so an owed read that fails or is cut short does not drop it.

- A stop is held when the list holds an entry of this session that it was not yet held for. A list that only shrinks holds nothing. Which entries it was held for is one file for each session, `held/<session-id>.json` in the findings store, so no field is added to `SessionState`, which an older build rewrites without it.
- The mark is written before the hold is emitted. One that cannot be read or written, and a list that cannot be read, allows the stop and says so.
- While the person is away, or in a drive's implementer stage, every open entry of the session counts. While the person is present, an entry made from words alone does not: it stays on the list and in the count. The ruling is that a confirmed finding is held in both cases (dcn-2611252038554826); a word match is not a confirmed finding, so holding a present person's session on one would go past it.
- At most four holds in one chain of held stops (`stop_hook_active`), below the host's eight, and then the stop is allowed with the line below. While away a hold joins the away check's own reason as one answer and counts toward the same cap, through one function beside `JudgeChecklist`; no other hold changes.
- In such a chain, a closing text that names a handle of its session is an answer: it is read for the block and not for words. The hold says to name the handle on each line that answers.
- The hold names each entry by handle, kind and 120 bytes of its redacted text, ten at most and then the count, within 3,000 bytes; beside an away reason that would pass 9,500 together it is the count and the command. Its clearing act names the hook's own executable by absolute path.
- A later stop with nothing new is allowed with one line for the person: the count and the command. It is the disk alone, and while present nothing but the list holds a stop.

In a drive's stage the away check returns at once, as today. An implementer's stage is read, put and held; any other role's is left alone. The stage's role reaches the hook as one more sealed variable, `ABCD_STAGE_ROLE`, set where the others are (`internal/core/runner/claude.go`).

**The session's start.** One notice: the count, the command, and each entry an earlier session left in this checkout by handle and kind, ten at most, within 600 bytes. The checklist's render already yields to what the hook printed before it.

**The sub-agent's start and stop (step 3).** The ISSUES line goes to every agent told anything, the block instruction to those whose definition does not fix its reply as one document. Those that do are a list in code (graveyard-interpreter, intent-auditor, lifeboat-reviewer, planning-interviewer, press-release-composer, principle-distiller, question-drafter, reflection-composer, release-changelog-composer, scribe), held to `agents/*.md` by a test. The stop reads the payload's `last_assistant_message`, else the transcript's last assistant text, and only of an agent whose told fact says the block was sent. Its entries carry the starting session's id. A told agent that ends with no block is one "sent no block" line, and an unreadable text one "findings not judged" line. The stop itself is never held, and the session reads its sub-agents' declared entries at its next stop.

**The landing (step 4).** A `check` entry refuses no landing. The gate answers for it: a landing that goes on holds a preflight receipt naming its head, and one whose preflight fails is refused as today.

- `internal/core/landing/findingshold.go`, called from `internal/surface/cli/land.go`, settles every `check` entry of the lane when a receipt names the offered head, before `landing.Land` and again after it, so both a skipped gate and a run one settle. A `check` settled so may be put again. A preflight that fails in `Options.Gate` puts one `check`, named by the head. In between, it refuses over the lane's other entries, and nothing is pushed. `internal/core/landing/land.go` is not edited.
- For a build or a drive, `findingsHeld` wraps the landing stage's body, as `takingReceipts` does, at the one call where that is done (`internal/core/implement/loop/loop.go`), and does nothing once the landing has pushed. It settles by the receipt first, refusing a receipt a drive would not take (`ReceiptOwn`), then refuses over the rest before the sync and the preflight are spent. The refusal is a contention, which `mayWait` lets a landing be, so the same call moves the next lane. In a drive it is also final, so the lane goes back as stuck with its worktree and branch kept. Whether it is a drive comes from `Options.ReceiptOwn`, not from the process. A drive's own failed preflight puts one `check` (`preflightcheckout.go`).

A lane's list is the entries placed in its checkout, those naming its run and lane, and the unplaced ones put by the session that runs the landing, `abcd land` or a host's `abcd implement step`. A host's implementer is a sub-agent whose payload names the session's own directory, so its entries are unplaced, and this is what holds them to a landing. A drive runs in no session and has none.

**What a refusal may carry.** Handles, kinds, the count and the clearing command; never an entry's text. A refusal's reason goes to the hand-back, and from there to a store designed to be pushed. The same holds for the drive's end, `implement leave`, the session start and the board. Entry text appears in `capture owed` and in a hold to the session that owns it, and nowhere else.

**The end.** `abcd implement leave` and the drive's end print each entry still owed by handle and kind, ten at most, after what they print today. A handover written by hand is not read.

### How each criterion is met

1. The domain's line (step 1) and the start's (step 3), one constant and a test that holds the two equal.
2. "Still fails" in a text (step 2), and a gate failing at the head (step 4), list it; a receipt naming the head settles it (step 4); `--own` and `--captured` clear it (step 1). abcd runs nothing on a base.
3. The words (step 2); the flags and the kept reason (step 1).
4. `doubt` (steps 1 and 2).
5. The start message and the two exclusions (step 3).
6. The sub-agent's stop and the live check (step 3).
7. The hold (step 2).
8. The landing holds and the two reports (step 4). `--captured` with a record saying which test is owed clears one that cannot be tested.
9. `abcd capture … --fixed` (step 1). The ISSUES domain gains a second line: a fix outside what the lane reserved is captured and left open (step 1).
10. The store and `capture owed` and the board (step 1), the session-start notice (step 2).
11. `private/` and its terminal test (step 1). Every render of steps 2 to 4 has a test that plants a private text and searches the output for it.
12. The resolver: at the settle side (step 1) and at the block (step 2).

### The live check

In step 3. What makes the host run the new hook is the project settings file of a scratch repository abcd manages, whose start, sub-agent start, sub-agent stop and stop hooks name the branch's built binary by absolute path. No install and no dogfood opt-in is touched. The sub-agent stop's entry there is a two-line shell that appends the payload's field names, never their content, to a log of its own and hands the payload on. One session starts one sub-agent told to end with one known block. The receipt is `.abcd/.work.local/logs/findings-live-<date>.md` in the main checkout: the host's version, the log's field names, the entry's handle. Where `last_assistant_message` is absent the receipt says so and the transcript carries the criterion. It needs a login and one small sub-agent.

### The lists that pin this change

- **A new sub-verb and flags**: `internal/core/surface/sentences.go`, `internal/core/surface/examples.go`, `.abcd/development/release/surface.json`, `docs/reference/cli/commands.md`, `docs/reference/cli/commands/capture.json`, the generated block of `commands/capture.md`. Tests: `TestEveryVisibleVerbCarriesItsSentenceEverywhere`, `TestEveryVisibleCommandCarriesAWorkedExample`, `TestEachWorkedExampleIsShapedLikeItsCommand`, `TestPluginSurfaceReachesEveryBinaryVerb`, `TestCommandFactsFilesMatchTheCommittedCopies`, `TestCommandPageBlocksMatchTheCommittedPages`.
- **Refusals**: `capture/invalid-input`, `capture/unknown-issue`, `usage/missing-argument` and `usage/flag-conflict` gain the verb in their `RaisedBy` lists and `.abcd/development/release/refusal-codes.json`. Tests: `TestRaisedByIsWhatTheVerbsRaise`, `TestExitErrorIsBuiltOnlyThroughItsConstructors`, `TestEveryNextCommandAndRaiserIsACommandTheTreeHas`. `refusal-uncoded.txt` only shrinks, so every new file of `internal/surface/cli` is coded from its first line.
- **The hooks**: `internal/surface/cli/testdata/hook-git-baseline.txt` gains no line (`TestHookGitCallsMatchBaseline`): each git call takes the hook's context. No budget is raised.
- **The sweep**: neither ceiling of `internal/core/guard/sweep_test.go` is raised (`TestFalsePositiveSweepCeiling`).
- **The rules**: the ISSUES domain's render and the injection budget in `internal/core/rules`.
- **The readers of a return**: the landing's and the loop's verdict and findings readers, each given a return that ends with the block, and reading it as without.

### Generated files

`go generate ./internal/surface/cli`, `make command-facts-sync` and the surface snapshot command, as `make precheck` names them, in step 1, and in step 4 for the pages it changes (`land.json`, `implement.json`, `drain.json`). Run twice, the second leaves the tree unchanged.

### Decisions made in this spec

Each is technical, the writing session's, and follows what has landed.

1. The words have the two sets above and a stated bound, because the loose reading listed most of what agents write and would be answered without being read.
2. A state of the list is new only when it gains an entry. A count going down is an answer.
3. An unplaced entry refuses any landing its own session runs: no lane would ever answer for it.
4. A session is held for what it and its sub-agents put, never for an earlier session's. A drive's implementer stage is a session, and is the one that can still test what it met.
5. The prompt carries no note. Criterion 6 says the next stop or prompt, and the stop is the one place already bounded.
6. A failing check never refuses a landing by itself: the preflight does, and a receipt is what settles it.
7. The sub-verb is under `capture`, so no top-level verb, plugin page or manifest row is added.
8. A list that cannot be read refuses a landing and holds no stop: the landing's other reads already refuse when they cannot tell (`stopHold`). A newer build's entry is read so too.
9. A private text goes by command and not in the block, because the host hands a sub-agent's reply to its parent.

### What it must not do

- Capture, list or count what a change made and removed before it lands, a failing test on the way among them (dcn-2611251839563777).
- Run a check on a base, or watch a command in an agent's shell.
- Change the capture verb's own gates or the automatic filers (iss-2610271115000748's build).
- Read or write the embargo store, or change what a capture of category `security` does.
- Hold a sub-agent's stop, or change any hold of the away mode.
- Read a handover, or any other file written by hand.
- Edit an agent's definition, a drive's wall, or a hook script. Add a dependency.

### The checks, and how each is run

- Each test a step names: by name, once red on an assertion and once green, under `scripts/abcd.sh implement load --wait --narrow --timeout 2m -- go test -trimpath -count=1 -timeout 7m -run '<names>'`, up to three packages at once. The whole of `internal/surface/cli` is run by the preflight alone.
- `make record-lint` and `scripts/abcd.sh intent ready itd-2611251831138387` after an edit of a record.
- At the end of each step: `make precheck`, the sweep ceiling and the git baseline by name, then `make preflight` on the committed head and a plain push.
- The independent fit-challenge the intent's SOTA section owes, before step 1 lands; the review pair on each step's diff.

### What waits if time runs short

Each step is one branch and one pull request. After step 1 the list can be kept by hand; after step 2 a session is held on its own text; steps 3 and 4 do not need each other. The spec stays open, with no `Delivers:` trailer, until all four are in.

### Limits the code sets

- A finding an agent never mentions, and one it words outside the two sets, is not read. The words also find ordinary prose, within the bound.
- A drive's wall carries no sub-agent start hook, so a stage's own sub-agents are told of no block and read for none.
- A preflight a host runs in its own shell is not seen; the lane's text is what lists its failure.
- The terminal test keeps a private text out of what abcd renders to a session. It does not hold against an agent of the same account, which can read the file or open a terminal of its own. The private list does not replace the advisory rule: a vulnerability still goes to a private advisory on the forge.
- The host's transcript holds what an agent typed, a private text on a command's input included, and abcd's staged copy is redacted only as any transcript is.
- An agent that settles a finding with a false test or reason is recorded and not caught; a reader of a sample is the check.
- A stop whose hook did not run puts nothing. The list is an account's, on one computer.

## Footprint

- packages: internal/core/findings (new), internal/core/capture/capture.go, internal/core/capture/workflow.go,
  internal/core/capture/redact.go, internal/core/rules/defaults/rules.json, internal/core/rules/subagent.go,
  internal/core/autonomy/findings_hold.go (new), internal/core/history/subagentreading.go,
  internal/core/runner/claude.go, internal/core/landing/findingshold.go (new),
  internal/core/implement/loop/findingshold.go (new), internal/core/implement/loop/loop.go,
  internal/core/implement/loop/preflightcheckout.go, internal/core/surface/sentences.go,
  internal/core/surface/examples.go, internal/core/refusal, internal/surface/cli/capture_owed.go (new),
  internal/surface/cli/hook_findings.go (new), internal/surface/cli/cli.go, internal/surface/cli/board.go,
  internal/surface/cli/hook_stop.go, internal/surface/cli/hook_subagent.go,
  internal/surface/cli/hook_subagent_start.go, internal/surface/cli/land.go, internal/surface/cli/implement.go,
  internal/surface/cli/drain_drive.go, internal/surface/cli/refusal_raisers_test.go,
  .abcd/development/release/refusal-codes.json, .abcd/development/release/surface.json,
  docs/reference/cli/commands.md, docs/reference/cli/commands/capture.json, docs/reference/cli/commands/land.json,
  docs/reference/cli/commands/implement.json, docs/reference/cli/commands/drain.json,
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
  .abcd/work/issues/open/iss-2611251828531764-confirmed-finding-is-always-captured.md,
  .abcd/work/issues/resolved/iss-2611251828531764-confirmed-finding-is-always-captured.md
- tests: internal/core/findings, internal/core/capture/workflow_test.go, internal/core/capture/redact_test.go,
  internal/core/rules/rules_test.go, internal/core/rules/subagent_test.go,
  internal/core/autonomy/findings_hold_test.go (new), internal/core/history/subagentreading_test.go,
  internal/core/landing/findingshold_test.go (new), internal/core/landing/review_test.go,
  internal/core/implement/loop/findingshold_test.go (new), internal/core/implement/loop/findings_test.go,
  internal/surface/cli/capture_owed_test.go (new), internal/surface/cli/hook_findings_test.go (new),
  internal/surface/cli/hook_stop_test.go, internal/surface/cli/hook_subagent_start_test.go,
  internal/surface/cli/hook_subagent_stop_test.go, internal/surface/cli/board_view_test.go,
  internal/surface/cli/land_hold_surface_test.go, internal/surface/cli/implement_surface_test.go,
  internal/surface/cli/drain_drive_test.go, and the standing tests named under "The lists that pin this change"

## Steps

1. The rule, the list and its commands
   - needs: none
   - packages: internal/core/findings/list.go (new), internal/core/findings/list_test.go (new), internal/core/findings/records.go (new), internal/core/findings/records_test.go (new), internal/core/capture/capture.go, internal/core/capture/workflow.go, internal/core/capture/redact.go, internal/core/rules/defaults/rules.json, internal/surface/cli/capture_owed.go (new), internal/surface/cli/cli.go, internal/surface/cli/board.go, internal/core/surface/sentences.go, internal/core/surface/examples.go, internal/core/refusal, internal/surface/cli/refusal_raisers_test.go, .abcd/development/release/refusal-codes.json, .abcd/development/release/surface.json, docs/reference/cli/commands.md, docs/reference/cli/commands/capture.json, commands/capture.md, .abcd/development/brief/04-surfaces/06-capture.md, .abcd/development/brief/04-surfaces/08-abcd.md, .abcd/development/specs/open/spc-2611252115253843-confirmed-finding-is-always-captured.md
   - tests: internal/core/findings/list_test.go, internal/core/findings/records_test.go, internal/core/capture/workflow_test.go, internal/core/capture/redact_test.go, internal/core/rules/rules_test.go, internal/surface/cli/capture_owed_test.go, internal/surface/cli/board_view_test.go, internal/surface/cli/refusal_raisers_test.go

   Meets the domain half of 1, the settle side of 2, 3, 4 and 12, 9, 11 (the store and the command), and 10 but its notice.

   Tests first. The store: said twice in one session is one file, in two sessions two; a settled one stays settled, a receipt-settled `check` does not; a crash inside a settle; the worktree gone; a year; a link, another owner, a loose directory and a loose file refused and named; each bound of the table; a later schema and an unknown kind; two writers at once; a planted secret and a home path redacted, and a scanner that cannot be made withholding the text. The records: open, resolved and wontfix on the default branch; open and resolved in the checkout alone; open in a third worktree not counted; no default branch. The command: each cell of the kind table; each adding flag; both lists; `--json`; a main checkout two sessions share. Private: planted text absent with a marker, a pipe and a stage, present in a terminal. The fixed filing: one record, nothing on each refusal, the trailer accepted by `scripts/check-issue-resolution.sh`. The two ISSUES lines; the board's line, and none at none.

   The core half (the findings package, the capture package and the rules line) is one commit and the command half another, since the files the command half needs are held longest. Stays for the next: no text is read.

   Done when: every test above was watched fail on an assertion and passes; `make precheck` is clean; the generating commands leave the tree unchanged when run again; a bare `abcd capture owed` in a fresh checkout makes nothing under abcd's home beyond what every command lays.

2. A session's own text is read and its stop is held
   - needs: 1
   - packages: internal/core/findings/read.go (new), internal/core/findings/read_test.go (new), internal/core/autonomy/findings_hold.go (new), internal/core/runner/claude.go, internal/surface/cli/hook_findings.go (new), internal/surface/cli/hook_stop.go, internal/surface/cli/cli.go, .abcd/development/brief/02-constraints/03-invariants.md, .abcd/development/brief/04-surfaces/01-ahoy.md, .abcd/development/brief/05-internals/03-configuration.md, .abcd/development/specs/open/spc-2611252115253843-confirmed-finding-is-always-captured.md
   - tests: internal/core/findings/read_test.go, internal/core/autonomy/findings_hold_test.go, internal/surface/cli/hook_findings_test.go, internal/surface/cli/hook_stop_test.go, internal/core/landing/review_test.go, internal/core/implement/loop/findings_test.go

   Meets the words half of 2, 3, 4, 7, the notice of 10, the renders of 11 and 12 at the block.

   Tests first. The readers of a return, unchanged by a trailing block. The block: each state, every bad line, a fence that does not end the text. The words: each phrase of each set, with and without a fault word, in a fence, a quote and a code span, beside an open issue, spec, draft and planned intent, beside a resolved record; a review return; a `Verdict` heading. The stop: held once when the list gains; the same list and a shorter one allowed with the line; an entry from words alone held while away and in an implementer's stage and not while present; the fifth in a chain allowed; an answer making no entry by words; an unreadable list and an unreadable or unwritable mark allowed and said; a mark an older build's session state does not drop; an owed read that failed; rules off; unmanaged; a validator's stage; one answer with the away reason, within the bytes, whose existing tests pass unedited; no forge call while present. The start's notice.

   The invariants and configuration chapters name the one thing that holds a stop while the person is present. The home's tree in `01-ahoy.md` gains its row here. Stays for the next: no sub-agent is told or read, and no landing reads the list.

   Done when: every test above was watched fail on an assertion and passes; the standing stop tests pass unedited; the word reading, run over the host's stored texts for this repository, is within the bound and the counts (no text) are in the pull request; the git baseline and the sweep ceiling are unchanged; `make precheck` is clean.

3. Sub-agents are told of the block and read for it
   - needs: 2
   - packages: internal/core/rules/subagent.go, internal/core/history/subagentreading.go, internal/surface/cli/hook_findings.go, internal/surface/cli/hook_subagent.go, internal/surface/cli/hook_subagent_start.go, commands/history.md, .abcd/development/brief/04-surfaces/11-history.md, .abcd/development/brief/05-internals/03-configuration.md, .abcd/development/specs/open/spc-2611252115253843-confirmed-finding-is-always-captured.md
   - tests: internal/core/rules/subagent_test.go, internal/core/history/subagentreading_test.go, internal/surface/cli/hook_findings_test.go, internal/surface/cli/hook_subagent_start_test.go, internal/surface/cli/hook_subagent_stop_test.go

   It waits for itd-2611251725190796 on the main branch. Meets the start half of 1, 5 and 6.

   Tests first. The message: the line, the block, the bound with the longest conventions file that branch's tests use, the ten agents without the block, the four told nothing, the definitions' test, the fact saying what was sent. The stop: the entries on the starting session's list and held at its next stop; an agent told by the guard and not the start hook unread; no block counted; unreadable text a line; a blank answer on every input, so never a hold; the place from the payload's directory, with a transcript naming another lane.

   Stays for the next: no landing reads the list. Done when: every test above was watched fail on an assertion and passes; the live check's receipt is in the local tier; `make precheck` is clean.

4. A landing is refused, a run's end says what is owed, and the close
   - needs: 2, 3
   - packages: internal/core/landing/findingshold.go (new), internal/core/implement/loop/findingshold.go (new), internal/core/implement/loop/loop.go, internal/core/implement/loop/preflightcheckout.go, internal/surface/cli/land.go, internal/surface/cli/implement.go, internal/surface/cli/drain_drive.go, commands/land.md, commands/implement.md, commands/drain.md, docs/reference/cli/commands.md, docs/reference/cli/commands/land.json, docs/reference/cli/commands/implement.json, docs/reference/cli/commands/drain.json, .abcd/development/brief/04-surfaces/27-implement.md, .abcd/development/brief/04-surfaces/35-drain.md, .abcd/development/brief/04-surfaces/38-land.md, .abcd/development/specs/open/spc-2611252115253843-confirmed-finding-is-always-captured.md, .abcd/development/specs/closed/spc-2611252115253843-confirmed-finding-is-always-captured.md, .abcd/development/intents/planned/itd-2611251831138387-confirmed-finding-is-always-captured.md, .abcd/development/intents/shipped/itd-2611251831138387-confirmed-finding-is-always-captured.md, .abcd/work/issues/open/iss-2611251828531764-confirmed-finding-is-always-captured.md, .abcd/work/issues/resolved/iss-2611251828531764-confirmed-finding-is-always-captured.md
   - tests: internal/core/landing/findingshold_test.go, internal/core/implement/loop/findingshold_test.go, internal/surface/cli/land_hold_surface_test.go, internal/surface/cli/implement_surface_test.go, internal/surface/cli/drain_drive_test.go

   Its code needs step 2 alone; it needs 3 for the close. Meets the gate half of 2, 8, and 11 at the landing and the end.

   Tests first. The landing: refused with nothing pushed; another lane's entry, a private one and a `check` not counted; an unplaced entry of this session counted; an unreadable list and a newer build's entry refused. The check: a mid-build "still fails" in a text followed by a green head leaves nothing owed and lands, with the gate run and with the gate skipped over a receipt, in `abcd land` and in the loop; a gate that fails puts one entry, and a second head another; a receipt of a drive's that the drive would not take settles nothing. The loop: the sync and preflight not run; the next lane moved in the same call; a drive hands the lane back with its worktree and branch; a host run goes on refusing until cleared, an unplaced entry of its own session included and another session's not; a landing that has pushed goes on. A planted entry text absent from every refusal, hand-back, run record and end. The end with entries and with none.

   The close, in the same change: a docs review for the head is recorded, the issue and the spec are closed, and the commit carries `Delivers: itd-2611251831138387` and `Resolves: iss-2611251828531764`. The impact is the one the record carries, `breaking`, the person's to change before it.

   Done when: every test above was watched fail on an assertion and passes; `make precheck` and `make preflight` are clean; a landing in a scratch repository is refused on one entry, cleared and made, and its output is in the pull request; the spec is in `closed/` and the intent in `shipped/`.
