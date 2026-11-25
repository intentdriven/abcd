---
id: spc-2610161949559050
slug: a-product-thinker-who-wants-to-know
intent: itd-2610161733316518
origin: researcher-authored
production_mode: dictated-and-formatted
---
# a-product-thinker-who-wants-to-know

## Summary

Delivers itd-2610161733316518: one task (a code feature, a written doc, or a check such
as doc fidelity) done many times across a declared matrix of models, from one frozen
commit, judged the way the task allows, merged nowhere, and reported with a record from
which the run can be started again and its drift shown. The design record is the
intent's Decisions 1–33; this spec says how they are built.

**Shape.** A new core package, `internal/core/matrix`, owns the run: the matrix
declaration, the plan and consent, the cells, the trial records, the judging ingest and
the report. It drives existing pieces rather than copying them:

- **Run as a lab procedure** (decision 4): `internal/core/lab` pins the frozen commit and
  the binary digest; the matrix adds a model pin per cell. Each copy is an isolated clone
  of the frozen commit in the machine-scoped store, keyed intent/cell/repeat, bundled and
  deleted once scored.
- **Trial mode of the build loop** (decision 4): `internal/core/implement/loop` gains a
  mode that takes no reservation, lands nothing, and switches off host fallback inside a
  cell; a runner failure is the cell's failure.
- **Routing per copy** (decision 14): the existing per-invocation route override, extended
  to name a runner and its model (iss-2610161801358637), on the person's own layer only.
- **Copy production per task kind** (decision 29): code through the runners; a
  self-contained writer through the provider adapter once on the self-contained list;
  a file-reading check through the runners with a pinned reviewer prompt. Fallback off on
  both paths.
- **Recording** (itd-166, its blocker): every call's asked and served sides, unpinnable
  fields marked; the run record names the frozen commit, binary digest and version,
  prompt and tool-definition hashes, the render template, machine, concurrency and load
  cap. The trial record maps onto a Harbor trial result and an Inspect sample (the seam
  for path 2 of sota-per-intent).
- **Judging** (decisions 21–24, 26–27, 32; adr-2610161733400551):
  - code: hidden tests outside the clone, reviewer models as self-contained blinded
    requests assembled through the reading surface's include table and hashed manifest,
    scored on known-good and seeded-defect copies;
  - writing: an abcdesign.app judging session (itd-2610091016262335, session kinds
    iss-2610161928115039); calibration, ranked subsets of at most eight, good-enough
    marks, no author guess; verdicts read back as an untrusted file the run validates;
  - checks: findings kept per copy (never `docfidelity.Record`, never the ledger), matched
    by exact key then a validated matcher, judged in two phases in the same session kind.
- **Statistics** (decisions 8, 26): pilot then power-sized repeats; pass@1 with interval
  and pass^k; regularised Bradley–Terry with bootstrapped intervals; α with interval or
  "not estimable"; floors below three judges.
- **Safety and load** (decisions 12, 17; technical criteria): consent screen with cost
  against a named cap for metered cells; secret scan of each provider-bound clone; one
  local inference server at a time as an owned process group under a consented core cap;
  every network call inside the consented run; copies rendered inert.
- **Report** (decisions 16, 28): one row set per task kind, the limitations and suggested
  improvements of the run, and a re-run that reports drift field by field.

**How each acceptance criterion is met.** Person-visible: (1) plan and consent — step 2;
(2) code report — steps 4 and 8; (3) writing judged — step 6; (4) check compared — step 7;
(5) power floor — step 8; (6) re-run with drift — step 8; (7) nothing merged or filed —
steps 3, 5, 7; (8) limitations — step 8. Technical: no host fallback, same-pin retry,
isolated clones, secret scan, compliance check — steps 3–4; blinded reviewers and the
author-guess for models — step 5; matching findings — step 7; recording and run record —
step 1 with itd-166; one local server — step 3; inert rendering and validated verdicts —
step 6; gate receipts untouched — step 7; Harbor/Inspect mapping — step 1.

**What steps 9 to 14 add.** Steps 1 to 8 built every piece above, but six criteria are
not reachable from the command line (iss-2611211344505614, dcn-2611211517021229): the
check judging and its matcher (ac-4, ac-15), a runner cell's launch and its same-pin
retry (ac-9, ac-10), the one local inference server (ac-18) and the judging session
people join by code (ac-3). A code copy's hidden-test score is also still the host's
report, not abcd's own run (ac-2). Steps 9 to 14 wire them, in that order of readiness:
step 9 the check judging, step 10 the hidden tests, step 11 the launcher, step 12 the
local server, step 13 the parts dcn-2611211433133728 left for a later change, step 14
the judging session. The intent ships whole: no step before 14 closes this spec, and
step 14 waits on the session kind itd-2610091016262335 builds (decisions 19, 23, 25).
Three points of the shape above are narrowed by these steps and stated where they
apply: a runner cell's copy is made by the cell's runner dispatched in the copy's
clone, not through the build loop's trial mode (step 11, dcn-2611251746554510); the matcher and the outside
judge are cells the person declares (step 9); and a code task's hidden tests and test
command are named by the declaration (step 10).

## Footprint

- packages: internal/core/matrix/doc.go, internal/core/matrix/declaration.go, internal/core/matrix/run.go, internal/core/matrix/trial.go, internal/core/matrix/export.go, internal/core/matrix/declaration_test.go, internal/core/matrix/run_test.go, internal/core/matrix/trial_test.go, internal/core/matrix/export_test.go, internal/core/matrix/testdata/inspect-eval-sample.json, internal/core/matrix/testdata/harbor-trial-result.json, internal/core/lab/procedure.go, internal/core/lab/procedure_test.go, internal/core/lab/preflight.go, internal/README.md, ACKNOWLEDGEMENTS.md, internal/core/oracle/store.go, internal/core/matrix/plan.go, internal/core/matrix/plan_test.go, internal/core/matrix/cap.go, internal/core/matrix/cap_test.go, internal/core/matrix/consent.go, internal/core/matrix/consent_test.go, internal/core/matrix/testdata/declaration-plan.json, internal/surface/cli/lab_matrix.go, internal/surface/cli/lab_matrix_test.go, internal/surface/cli/lab.go, internal/surface/cli/lab_surface_test.go, internal/core/surface/examples.go, commands/lab.md, .abcd/development/brief/04-surfaces/31-lab.md, internal/core/implement/loop/trial.go, internal/core/implement/loop/trial_test.go, internal/core/implement/loop/terminal_limit_release_test.go, internal/core/runner/dispatch.go, internal/core/runner/runner.go, internal/core/runner/cell_test.go, internal/core/launch/treescan.go, internal/core/launch/treescan_test.go, internal/core/matrix/clone.go, internal/core/matrix/clone_test.go, internal/core/matrix/cell.go, internal/core/matrix/cell_test.go, internal/core/matrix/localserver.go, internal/core/matrix/localserver_test.go, internal/core/runner/opencode.go, internal/core/runner/main_test.go
- tests: the matrix plan refuses to start without consent and a named cap for metered cells; a failing runner cell is scored failed with no host fallback; trial mode takes no reservation and lands nothing; a provider-bound clone with a secret-shaped file is refused; reviewer requests carry no authorship field; a malformed verdict file is refused with nothing recorded; a check run leaves the doc-fidelity receipts untouched; Bradley–Terry and α floors; the trial record round-trips through a Harbor and an Inspect fixture
- The packages line names step 1's files. Each later step's files join it
  when that step begins, and `abcd build` then reserves the widened
  footprint (dcn-2610230129331232). Generated output is reserved by no one.
- Four files of `internal/core/implement/loop` that step 3 changed (`loop.go`,
  `land.go`, `reservation.go`, `state.go`) left the packages line when steps 9 to 14
  were added: step 3 has landed, no step from 9 on changes that package, and
  itd-2611201450052326 builds there, so a reservation of this spec no longer
  collides with it.

## Steps

1. The run record and the trial seam
   - packages: internal/core/matrix, internal/core/lab
   - tests: the matrix declaration parses and refuses unknown keys; the run record names commit, binary digest, hashes, template, machine, concurrency and cap; the trial record maps onto a Harbor trial result and an Inspect sample fixture
   - waits on: itd-166 (the asked and served sides)
   - landed: 68079f06d8e3
2. Plan, cost and consent
   - packages: internal/core/matrix, internal/surface/cli, commands
   - tests: the plan shows copies, cost by source against the cap, load cap, time and judges; nothing starts without consent; a metered run without a named cap is refused; a local- or host-only run needs no cap; a matrix refusal printed by the front door carries a control or direction character escaped, never raw; a scanner kind naming a repository's own rule id is printed sanitised; the role echoed in a pin refusal is quoted
   - The front door passes every matrix error through the terminal sanitiser
     before printing it, because the core returns raw text. The run record's
     refusal prints the scanner's finding kind sanitised, since an opted-in
     augmenter's kind carries the repository's own rule id.
   - When a trial writer is added, it calls the run record's screen on every
     trial before writing it, so a trial's reason and score detail are
     screened for credentials as the run record is.
   - The plan's cost and time follow decisions 34-36: no copy starts past the
     cap's remainder, the time waits for the pilot, and the expected cost
     comes only from prices the person declares.
   - landed: 845220bb3b14
3. Trial mode, isolated clones and routing
   - packages: internal/core/implement/loop, internal/core/runner, internal/core/launch, internal/core/matrix
   - tests: trial mode takes no reservation and lands nothing; a runner failure in a cell is the cell's failure with no host fallback; an OpenRouter outage is retried on the same pin and not scored; each copy is a clone outside every working tree with no local tier; a provider-bound clone passes the secret scan or is refused; one local server at a time as an owned process group
   - needs: 1, 2
   - waits on: iss-2610161801358637, iss-2610161734413178, iss-2610161734428291, iss-2610161709506207
   - landed: 291c03b5ff22
4. Code tasks: hidden tests and compliance
   - packages: internal/core/matrix
   - tests: hidden tests sit outside the clone; the compliance check excludes a copy that edited tests or read another copy, and marks an unexaminable copy unexamined; host-harness isolation is reported as asserted
   - needs: 1, 3
   - landed: 9bcb22fba47d
5. Blinded reviewer models
   - packages: internal/core/matrix, internal/core/reading, internal/core/oracle
   - tests: a reviewer request is assembled through the include table with no trailer, author, branch or run metadata; recall and true-negative rate on known-good and seeded-defect copies; the author-guess score; same-family verdicts reported apart
   - needs: 2, 3
   - landed: 110d8f69cdcf
6. Writing tasks judged in an abcdesign.app session
   - packages: internal/core/matrix, internal/core/oracle
   - tests: a judge's subset holds at most eight copies and every pair of models is compared; copies render as inert text; a verdict file is validated and refused whole when malformed; one ranking per judge identity; guest verdicts and the host's self-blinded verdicts reported apart
   - needs: 2, 3
   - waits on: itd-2610091016262335 and iss-2610161928115039 (the session and its kinds)
   - landed: 2719e3765dcd
7. Checks compared in two phases
   - packages: internal/core/matrix, internal/core/docfidelity, internal/core/record/findingkey
   - tests: a check copy's findings are read before its clone is deleted; the doc-fidelity receipts and the issue ledger are unchanged by a run; exact-key matching first, the matcher's agreement on a labelled sample reported; the overlap table appears only after the person's verdicts are locked; recall on seeded falsehoods
   - needs: 2, 3
   - waits on: itd-2610091016262335 and iss-2610161928115039
   - landed: 15e3f9c91e2c
8. Statistics, report and re-run
   - packages: internal/core/matrix, internal/surface/cli
   - tests: pass@1 interval and pass^k; regularised Bradley–Terry with bootstrapped intervals and "not separable"; α with interval or "not estimable"; floors below three judges; a comparison under its power estimate states the smallest detectable difference; the report lists limitations and suggested improvements; a re-run reports drift field by field
   - needs: 1, 4, 5, 6, 7
   - the step that first runs the compliance check on a real trial puts the whole check, every git call and the transcript scan together, under one overall deadline, past which the copy is marked unexamined; step 4 bounds each call alone
   - landed: 3a24b096b6d0
9. Check judging from the command line, with a declared matcher
   - packages: internal/core/matrix/declaration.go, internal/core/matrix/declaration_test.go, internal/core/matrix/plan.go, internal/core/matrix/plan_test.go, internal/core/matrix/run.go, internal/core/matrix/run_test.go, internal/core/matrix/checkflow.go, internal/core/matrix/checkflow_test.go, internal/core/matrix/checkjudge.go, internal/core/matrix/checkmatch.go, internal/core/matrix/consent.go, internal/core/surface/sentences.go, internal/surface/cli/lab_matrix.go, internal/surface/cli/lab_matrix_copy.go, internal/surface/cli/lab_matrix_check_test.go, commands/lab.md, .abcd/development/brief/04-surfaces/31-lab.md
   - tests: a declaration names an optional `matcher` and an optional `proposer` cell, and either named as a builder of the same run is refused, on a builder's route or on its model named through another provider or in another case, and a runner judge naming the model a host builder's copy was served is refused before the check is drawn; a judge the run record does not pin is refused; unknown keys are still refused; a declaration written before this step still parses and a run recorded before it still reports; the plan digest changes when either cell is added; a check task with no matcher plans with its leftovers stated as unmatched; a code task's `hidden_tests` and `test_command` parse and a plan carrying them shows the consent line saying abcd will run each copy's code with that command once it runs hidden tests itself, and that until then a code copy's result is the host's report; the printed plan shows each code task's hidden-tests folder and test command before its digest, the folder quoted and every word holding anything but printable ASCII quoted; `lab matrix judge` on a check task draws the shuffled finding packet with no model named and no proposed verdict; a verdict file locks once and a second is refused; the overlap table is refused until the verdicts lock; exact-key matching runs first and the matcher's request holds only the leftovers; a matcher answer naming an unknown finding id is refused whole with nothing recorded; the sample of the matcher's pairs is shown in a seeded order, so a pair's place says nothing of whether the matcher joined it; the matcher's agreement on a labelled sample is reported, and "not estimable" under the floor; a matcher's answer recorded with no packet beside it, as an interrupted draw leaves it, is named and the verb alone draws the packet from it; a table asked for before the verdicts lock is refused as not locked; a proposal's reason passes the secret screen before the table is written; a matcher or proposer on a runner route names its wait at the stage the judging stops at; with no matcher declared the leftovers stay rows of their own; the doc-fidelity receipts and the issue ledger are unchanged; every refusal reaches the terminal through the sanitiser
   - needs: 7, 8
   - acceptance: ac-4 and ac-15 from the command line, with ac-20's receipts untouched. The table's proposed verdict comes from the declared `proposer` cell and is absent when none is declared. Until step 13 lands, ac-4 and ac-15 hold from the command line for a run whose matcher and proposer are host cells, which is a run with no host builder (dcn-2611251953231215).
   - The matcher and the outside judge are cells the person declares, each on
     the person's own route, and both optional. Nothing is chosen for the
     person: a host or runner stand-in would be a model they never declared
     (dcn-2611211517021229). Without a matcher the leftovers are unmatched rows
     and the agreement is "not estimable".
   - Check judging extends `lab matrix judge` by the task's kind, with files in
     and out as the writing path has, and adds no verb. Step 14's session page
     produces the same verdict file, so this path stays when the session exists.
   - The declaration's `hidden_tests` and `test_command` fields land here as
     schema only, so the plan digest moves once; step 10 is their first reader.
   - A matcher or proposer on a runner route answers in cell mode, which step 13
     builds; until it does, the judging stops at that cell's stage
     (`awaiting_matcher` or `awaiting_proposer`) and says so, in the plan's
     judges note and at the stage, and nothing stands in for the cell
     (dcn-2611251953231215).
   - The step starts no process and writes only the run's state in the lab. The
     matcher's answer and the verdict file are untrusted input, validated whole.
10. Hidden tests placed and run by abcd
   - packages: internal/core/matrix/hiddenrun.go, internal/core/matrix/hiddenrun_test.go, internal/core/matrix/hiddenrun_unix_test.go, internal/core/matrix/hidden.go, internal/core/matrix/hidden_test.go, internal/core/matrix/handoff.go, internal/core/matrix/launch.go, internal/core/matrix/launch_test.go, internal/core/matrix/trial.go, internal/core/matrix/trial_test.go, internal/core/matrix/report.go, internal/core/matrix/report_test.go, commands/lab.md, .abcd/development/brief/04-surfaces/31-lab.md
   - tests: the hidden tests are placed at the run's start outside every clone, under a name made from their digest; the test run happens in an export of the copy with the tests added, never in the copy's clone; the run is one owned process group, ended whole at its deadline through the handle the run holds; a run past its deadline is "unexamined", never "failed"; the run's environment carries no credential; the exit status maps to pass or fail and the output is screened for credentials before it is kept; a consented run places the hidden tests and an unconsented one places nothing; a host copy handed back and a runner copy launched are both scored by abcd's run; the trial says abcd placed and ran the tests; the report drops the host-reported qualifier for those copies only; a hidden-tests folder inside the checkout, any worktree of it or the clone store, one reached through a symlink, a symlinked last component and one differing from the checkout only by case are each refused before anything is placed, and the folder is read through a guarded root so it cannot be swapped between the check and the copy; the consent line stops saying abcd does not run the tests yet
   - needs: 4, 9
   - acceptance: ac-2's pass rate is abcd's own run of the hidden tests, and ac-11's hidden tests sit outside the clone from the run's start.
   - The step has two halves and lands once. The core half (placing and running,
     the `hidden*` files) needs steps 4 and 9 only. The wiring half (`handoff.go`,
     `launch.go`, `trial.go`, `report.go` and their tests) also needs step 11,
     whose `launch.go` it edits; a `needs:` line names earlier steps only, so
     that order is stated here. The printed plan's hidden-tests lines landed
     with step 9. A lane that opens this
     step before step 11 has landed builds the core half and leaves the step
     unlanded.
   - The hidden tests are named by the declaration: a folder outside the
     checkout and a test command, kept by content in the run's record and shown
     on the consent screen (decision 9). Running them executes code a model
     wrote on the person's machine, so the change takes a security review
     before it lands.
   - Step 9 holds the folder only to an absolute, clean path, since nothing
     reads it there. Where the folder lies, what a symlink or a case fold
     makes of it, and whether it can change between the check and the copy are
     this step's, checked when it places the tests and pinned by the export it
     runs them in.
11. The runner-cell launcher
   - packages: internal/core/matrix/launch.go, internal/core/matrix/launch_test.go, internal/core/matrix/launch_unix_test.go, internal/core/matrix/handoff.go, internal/core/matrix/cell.go, internal/core/matrix/compliance.go, internal/core/matrix/compliance_test.go, internal/core/matrix/trial.go, internal/core/matrix/trial_test.go, internal/core/matrix/report.go, internal/core/matrix/report_test.go, internal/core/matrix/doc.go, internal/surface/cli/lab_matrix_launch.go, internal/surface/cli/lab_matrix_launch_test.go, internal/surface/cli/lab_matrix.go, internal/core/surface/examples.go, commands/lab.md, .abcd/development/brief/04-surfaces/31-lab.md
   - tests: a runner cell's copy is dispatched in its clone in cell mode and its trial written; a runner that fails, refuses or answers unparsably is the cell's failure, with no fallback receipt and no host dispatch; a copy carrying a fallback receipt is excluded and counted as a failure; a provider down before the copy starts is retried on the same pin, recorded as not started and never scored; a host cell is refused by the launcher; a cell the secret scan refused is never launched; a launch whose copy the cap's remainder does not cover is refused; the brief and the receipt sit outside the clone, the clone store and the lab state; a transcript naming a sibling copy's brief folder breaks own_copy_only; a launched copy's served side comes from the call receipts and is not marked self-reported; the report stops counting a launched copy as awaiting its runner; a stop ends the launch as the run's error, never as the copy's score; the copies in flight never exceed the declared concurrency
   - needs: 3, 4, 8
   - acceptance: ac-9 and ac-10, with ac-12's secret scan and ac-13's compliance check holding for a launched copy as they do for a host copy.
   - The launcher dispatches the cell's runner directly in the copy's clone
     (`runner.Dispatcher` in cell mode, one implementer on the blind brief), as
     a host cell's copy is made from the same brief, so within a task the cells
     differ only by model and harness. It does not go through the build loop's
     trial mode, which cuts its lane into a worktree outside the clone that the
     compliance check, the review and the check collection do not read
     (dcn-2611211517021229). This narrows decision 4's wording; trial mode
     stays as step 3 built it, with no caller, and this step changes nothing in
     `internal/core/implement/loop` (dcn-2611251746554510).
   - The brief and the receipt sit in a third store,
     `~/.abcd.noindex/matrix-handoff/<root-sha>/<run>/<alias>/`, and
     own_copy_only treats any path in that store outside the copy's own alias
     folder as a read outside its copy. This settles the point
     dcn-2611211433133728 left open (dcn-2611251746554510).
   - A new sub-verb, `lab matrix launch <lab-id> <run-id> [<key>]`, launches the
     runner cells' copies in the foreground, up to the declared concurrency.
     Consent keeps handing off only: a launch can run for hours, longer than a
     host lets one command run.
   - Tested against a fake runner with no network, since the runners' live proof
     is still owed (iss-2609301519558538); the report's limitations say so until
     a real run exists. The step starts a process and writes outside the tree on
     a blinding boundary, so the change takes a security review before it lands.
12. One local inference server for local cells
   - packages: internal/core/matrix/launch.go, internal/core/matrix/launch_test.go, internal/core/matrix/localserver.go, internal/core/matrix/localserver_test.go, internal/core/runner/config.go, internal/core/runner/config_test.go, .abcd/development/brief/05-internals/03-configuration.md, .abcd/development/brief/04-surfaces/31-lab.md
   - tests: one server serves at a time across the run's local cells; it starts as one owned process group and stops as one when its cells end, through the handle the run holds; a second cell's server waits for the first's to stop; no server starts without the consented load cap; the start command is read from the person's machine layer, and one named by the declaration or the repository is refused
   - needs: 11
   - acceptance: ac-18.
   - The start command is a command abcd executes, so only the person's machine
     layer may name it, in the runner namespace that layer already owns. The
     step starts a long-lived child process, so the change takes a security
     review before it lands.
13. Sized repeats, seeded-defect copies, runner reviewers and clone bundling
   - packages: internal/core/matrix/launch.go, internal/core/matrix/launch_test.go, internal/core/matrix/handoff.go, internal/core/matrix/handback.go, internal/core/matrix/handback_test.go, internal/core/matrix/review.go, internal/core/matrix/review_test.go, internal/core/matrix/checkflow.go, internal/core/matrix/checkflow_test.go, internal/core/matrix/stats.go, internal/core/matrix/stats_test.go, internal/core/matrix/clone.go, internal/core/matrix/clone_test.go, internal/surface/cli/lab_matrix_copy.go, internal/surface/cli/lab_matrix_run_test.go, commands/lab.md, .abcd/development/brief/04-surfaces/31-lab.md
   - tests: the repeats past the pilot are sized from the pilot by the power estimate, and capped by `repeats.max` and by the spending cap's remainder; a seeded-defect copy keeps its source copy's key, and reviewer recall is measured on it; a runner reviewer cell answers a blinded request in cell mode with no host fallback; a runner matcher cell answers the check's matcher request, and a runner proposer cell the check's packet once the verdicts are locked, each in cell mode with no host fallback, and the judging moves on from `awaiting_matcher` or `awaiting_proposer` as it does for a host cell's answer; a scored code or writing copy's clone is bundled, then deleted, and a copy not yet scored keeps its clone
   - needs: 5, 8, 10, 11
   - acceptance: ac-5's power estimate sizes the run rather than only qualifying its report; ac-14's reviewer recall rests on seeded-defect copies the run makes; the shape's "bundled and deleted once scored" holds for code and writing copies as it does for check copies.
   - These are the four parts dcn-2611211433133728 named and left for a later
     change, and the runner matcher and proposer step 9's judging waits for
     (dcn-2611251953231215): the same cell-mode answer the runner reviewer
     takes, on the matcher's request and the packet. The step deletes files outside the tree, so the change takes a
     security review before it lands.
14. The judging session people join by code, and the close
   - packages: internal/core/matrix, internal/surface/cli, commands/lab.md, .abcd/development/brief/04-surfaces/31-lab.md
   - tests: a judge who joins the run's session by its code calibrates on one good and one bad example before seeing a copy; a judge sees at most eight copies under random codes in an order drawn for that judge; each copy is rendered as inert text in an isolated frame; a verdict is saved to the run's record on the host's machine as it is given and nothing is sent to a forge; a judge who leaves and rejoins finds their ranking where they left it; a check's two phases run in the same session kind and produce the verdict file step 9 reads; the session's verdicts are validated as untrusted input, one ranking per judge identity, and refused whole when malformed
   - needs: 6, 7, 9
   - waits on: itd-2610091016262335 and iss-2610161928115039 (the session and its kinds)
   - acceptance: ac-3, with ac-19 and ac-21 holding for the hosted session as they do for a verdict file.
   - Not planned further here: the session kind's page and relay rules belong
     to itd-2610091016262335, and this step's files are named when that intent
     lands. The packages line is a placeholder until then.
   - The change that lands this step closes the spec and ships the intent, and
     no earlier one does: it carries `Delivers: itd-2610161733316518` and
     `Resolves: iss-2611211344505614`, and steps 9 to 13 carry neither.
