---
id: spc-2611241502223513
slug: abcd-decides-for-each-change-which
intent: itd-2611241456272313
origin: researcher-authored
production_mode: hand-written
---
# Each change gets one check budget, and the check on what lands is never cut

## Summary

Delivers itd-2611241456272313. A project declares its checks once, in `.abcd/checks.json` on the default branch: each check's name, the command that runs it, the files it reads, the points at which it may run (the contributor's machine, the proposal, the landing, a schedule, an independent checker) and the platforms it covers. One classifier in the core reads a change's diff against that declaration and one planner turns the result into a plan: which check runs where for this change, which are left to the landing and why, which platforms nothing covers before landing, and whether waiting time or spend is being saved first. The plan may move a check later and may never drop it, because the landing runs every declared check for every change (adr-2611241456284353). A new verb, `abcd checks`, shows the layout, the plan and the measured figures. The intent claims the plan is right and the landing run whole; the saving is measured by the intents that cut a run. Five steps, each one pull request.

## Approach

### What exists today

The decision "is this change light" is written twice by hand in abcd's own repository, with two different lists: `scripts/records-only.sh` on the contributor's machine and the `changes` job in `.github/workflows/ci.yml` on the proposal. Both fail closed and neither reads the other. `abcd land` (`internal/core/landing/land.go`) runs one gate by name and knows nothing of which run a change needs. `abcd landings` (`internal/core/landings`) reads how each change landed and not what it cost. A project abcd manages has no declaration, no classifier and no plan.

### The measure (step 1)

`abcd checks measure [--since <tag|YYYY-MM-DD>]` reads the forge's run history through the existing forge client and reports, per landed change: the runs it had at each point, their duration, the time from push to landing, and the landing run's outcome at its first attempt. For each failed run it names the failing check, whether a rerun of the same commit passed (an intermittent test) and whether the check could have run before the push (an escape), and it counts escapes per check since the stated date. It writes only a cache in the local tier, as `landings` does. It needs no declaration, so it lands first and records the baseline in a research note before any behaviour changes. A forge it cannot read, or a project with no hosted runs, is reported in one sentence and nothing is estimated.

### The declaration (step 2)

`.abcd/checks.json` is strict JSON read through `core/jsonstrict`, with a `schema_version`. A check with no `inputs` reads everything. `definition` lists the files that define the checks themselves (build files, scripts, hosted workflow files); a change that touches one is planned in full. `budget` holds `priority` (`speed`, `spend`, or absent for the default), `late_failure_limit` (a pair, failures of landings, or absent for the derived default) and `early_run` (`quick` or `full`). Every reader takes the file from the default branch's tip (`git show <remote>/<default>:.abcd/checks.json`), never from the working tree, so a change cannot plan its own checks away (the principle evaluator-outside-the-loop, as itd-2611011325421304 does for its always-rerun list). The record lint gains a rule that the committed file parses, names no check twice and names a landing point for every check. abcd's own repository declares its gates in the same change, and the brief gains the invariant of adr-2611241456284353.

### The classifier and the plan (step 3)

`checks.Classify` takes a base, a head and the declaration and returns, per check, whether the diff can reach it. It fails closed: an unreadable diff, a missing base, a path outside every declared input set of a check that declares inputs, a change to a `definition` file or to the declaration, or any error, classes every check as reached and carries the reason. `checks.Plan` then places each check: at the landing always; at the contributor's machine when reached; on the proposal only when it is quick, or reads something only a proposal has, or `early_run` is `full`, or the proposer has no write access (read from where the head branch lives, never from anything the author sends). `abcd checks plan [<ref>]` prints it, and `abcd land` states the plan's one-line form in the proposal's text, which is what the report in step 5 later compares against.

The two hand-kept lists become readers. `scripts/records-only.sh` asks the binary. The hosted `changes` job keeps a list in the workflow, because the binary does not run inside a hosted run (a download to trust and a version to pin in every project), and a test holds that list equal to the declaration. A second test reads the workflow and fails unless every declared check with a landing point runs unconditionally on the landing event; `abcd lint` reports the same for a project's own workflows as far as it can prove it (the trigger lacks the landing event, or a job's condition excludes it), and says "unproven" for a condition it cannot read, never "fine".

### Priority, limit and restore (step 4)

With no `priority` set, the planner picks `speed` for a public repository and `spend` for a private one, from `repo.visibility` in `.abcd/config.json` and else from the forge, and every plan names the pick and its source. Under `spend` an optional early point on a platform the forge prices above the project's cheapest is left out unless the declaration asks for it; under `speed` every optional early point that can answer before the landing is taken.

`checks.Limit` returns the project's late-failure limit: the declared pair, or the derived default. The default is the share of changes that would fail the landing run if every early test-stage failure moved to it, measured over the last 300 proposal runs and 200 landing runs and rounded to the next whole "1 in N" towards more failures; fewer than a hundred landings gives 1 in 20. When the landing run's first-attempt failures over the last hundred landings pass the limit and `early_run` is `quick`, `abcd checks restore` writes `early_run: full` with the figures and the date into the declaration as a one-file change on its own branch, composed from those facts, and lands it by the ordinary route; `land` and `checks plan` report the restore as owed until it is on the default branch. `abcd checks cut` sets `early_run: quick` again and is a person-only act: the guard refuses it in an agent's session, and it asks at a terminal. After a cut the count starts from the cut's date, so the restore does not flap.

### The checker's point and the report (step 5)

The declaration admits the point `independent`. The planner shows it for a check that names it, marked optional and never waited for, and asks a small interface whether a checker is available; until itd-2611011325430933 supplies one the answer is "none available" and the plan says the platform is not covered before landing. `abcd checks measure --plans` sets each landed change's stated plan beside the runs it had, and the fifty-landing report is recorded in the research note.

### Wiring

`checks` is reachable from the CLI and the plugin page with `--json`, and is added to the surface manifest, the command reference, the release-gate manifest, the rules loader's verb list and the package map. It never writes to a project's CI.

## How the Acceptance Criteria are met

1. The plan lists each check, its point, the reasons and the uncovered platforms: `checks.Plan` and `abcd checks plan` (step 3), with the priority line from step 4.
2. Every declared check runs in full at the landing: the invariant and the declaration's rule that every check names a landing point (step 2), and the workflow test and lint finding (step 3).
3. An unprovable change is planned in full with the reason: `checks.Classify` failing closed (step 3).
4. A change to the declaration is planned from the default branch's copy, in full: the default-branch read (step 2) and the `definition` rule (step 3).
5. A change from someone without write access gets everything everywhere: the proposer rule in `checks.Plan` (step 3).
6. One answer on both sides: `scripts/records-only.sh` asks the binary and the hosted list is held to the declaration by test (step 3).
7. Speed for public, spend for private, named, and overridable: `checks.Priority` (step 4).
8. The limit, the derived default, the restore and the person-only cut: step 4.
9. One check, or no check of the merged result, cuts nothing and says so in one sentence: `checks.Plan`'s "nothing to do" results (step 3).
10. The independent point, optional and never waited for: step 5.
11. Failing check, intermittent or not, escape or not, counted per check: `abcd checks measure` (step 1).
12. The fifty-landing report: `abcd checks measure --plans` and the research note (steps 1 and 5).

## Open Questions

1. resolved: the declaration is one file under `.abcd/`, read from the default branch, not keys in `.abcd/config.json`; a per-project budget setting a change could edit for itself would defeat it.
2. resolved: the verb is `checks`, new, because no existing verb owns how a project's checks are laid out; `landings` keeps what admitted a change.
3. resolved: the restore is a committed change to the declaration, so every machine and the hosted run read one state, and it does not flap.
4. resolved: the binary does not run inside a project's hosted run; the hosted list is held to the declaration by a test.
5. resolved: the saving is measured by itd-2610110940216682 and itd-2610261025516272, so this spec closes on the fifty-landing comparison of plans with runs.

## Footprint

- packages: internal/core/checks/measure.go, internal/core/checks/forge.go, internal/core/checks/escape.go, internal/core/checks/declaration.go, internal/core/checks/read.go, internal/core/checks/classify.go, internal/core/checks/plan.go, internal/core/checks/priority.go, internal/core/checks/limit.go, internal/core/checks/restore.go, internal/core/checks/report.go, internal/core/lint/checksdecl.go, internal/core/lint/checkslanding.go, internal/core/lint/config.go, internal/core/landing/land.go, internal/core/landing/prbody.go, internal/core/guard/personacts.go, internal/core/surface/sentences.go, internal/ghcontract/rest-fields.txt, internal/surface/cli/checks.go, internal/surface/cli/cli.go, internal/README.md, commands/checks.md, commands/land.md, docs/reference/cli/commands.md, docs/reference/cli/commands/checks.json, scripts/records-only.sh, .github/workflows/ci.yml, .abcd/checks.json, .abcd/rules.json, .abcd/development/release/surface.json, .abcd/development/release-gate/manifest.json, .abcd/development/brief/02-constraints/03-invariants.md, .abcd/development/brief/04-surfaces/43-checks.md, .abcd/development/brief/04-surfaces/38-land.md, .abcd/development/brief/04-surfaces/README.md, .abcd/development/brief/05-internals/03-configuration.md, .abcd/development/research/notes/check-budget-baseline.md
- tests: internal/core/checks/measure_test.go, internal/core/checks/escape_test.go, internal/core/checks/declaration_test.go, internal/core/checks/read_test.go, internal/core/checks/classify_test.go, internal/core/checks/plan_test.go, internal/core/checks/priority_test.go, internal/core/checks/limit_test.go, internal/core/checks/restore_test.go, internal/core/checks/report_test.go, internal/core/lint/checksdecl_test.go, internal/core/lint/checkslanding_test.go, internal/core/lint/checksreaders_test.go, internal/core/lint/recordsonly_test.go, internal/core/landing/land_plan_test.go, internal/surface/cli/checks_surface_test.go, internal/surface/cli/checks_cut_test.go

## Steps

1. The measure
   - criteria: 11, 12
   - needs: none
   - packages: internal/core/checks/measure.go, internal/core/checks/forge.go, internal/core/checks/escape.go, internal/ghcontract/rest-fields.txt, internal/surface/cli/checks.go, internal/surface/cli/cli.go, internal/core/surface/sentences.go, internal/README.md, commands/checks.md, docs/reference/cli/commands.md, docs/reference/cli/commands/checks.json, .abcd/rules.json, .abcd/development/release/surface.json, .abcd/development/release-gate/manifest.json, .abcd/development/brief/04-surfaces/43-checks.md, .abcd/development/brief/04-surfaces/README.md, .abcd/development/research/notes/check-budget-baseline.md
   - tests: internal/core/checks/measure_test.go: from a fixture of the forge's run history, each landed change's runs, durations, push-to-landing time and first-attempt landing outcome; a forge that cannot be read and a project with no hosted runs each give one sentence and no figure. internal/core/checks/escape_test.go: a failed run names its check; a rerun of the same commit that passed marks it intermittent; a check that could have run before the push is an escape; escapes are counted per check since a date. internal/surface/cli/checks_surface_test.go: `checks measure` prints the figures and `--json` carries them; the verb is on the plugin page and in the command reference.
2. The declaration
   - criteria: 2, 4
   - needs: none
   - packages: internal/core/checks/declaration.go, internal/core/checks/read.go, internal/core/lint/checksdecl.go, internal/core/lint/config.go, .abcd/checks.json, .abcd/development/brief/02-constraints/03-invariants.md, .abcd/development/brief/05-internals/03-configuration.md
   - tests: internal/core/checks/declaration_test.go: a well-formed file loads; a repeated key, an unknown key, a check named twice, a check with no landing point and an unknown point are each refused naming the file and the key; a check with no inputs reads everything. internal/core/checks/read_test.go: in a fixture repository the declaration is read from the default branch's tip while the working tree and the checked-out branch hold a different one; a missing file reads as "no declaration", a failed read is an error and never an empty declaration. internal/core/lint/checksdecl_test.go: the record lint refuses a committed declaration that does not load, and abcd's own loads.
3. The classifier and the plan
   - criteria: 1, 2, 3, 4, 5, 6, 9
   - needs: 2
   - packages: internal/core/checks/classify.go, internal/core/checks/plan.go, internal/core/lint/checkslanding.go, internal/core/landing/land.go, internal/core/landing/prbody.go, internal/surface/cli/checks.go, commands/checks.md, commands/land.md, scripts/records-only.sh, .github/workflows/ci.yml, .abcd/development/brief/04-surfaces/43-checks.md, .abcd/development/brief/04-surfaces/38-land.md
   - tests: internal/core/checks/classify_test.go: a diff inside one check's inputs reaches that check alone; an unreadable diff, a missing base, an undeclared path, a change to a definition file and a change to the declaration each reach every check and carry the reason; every case that abcd's two hand-kept lists class as light today is classed the same. internal/core/checks/plan_test.go: every check is placed at the landing whatever the diff; a proposer without write access gets every check at every point; a project with one point, and one with no check of the merged result, each give the one-sentence result and no cut. internal/core/lint/checksreaders_test.go: the list in the hosted `changes` job equals the declaration's, watched fail on a one-path difference. internal/core/lint/checkslanding_test.go: the workflow test fails when a declared check is conditional on the landing event, watched fail first; the lint finding reports a missing landing trigger, an excluding condition, and "unproven" for a condition it cannot read. internal/core/lint/recordsonly_test.go: the script's answer is the binary's. internal/core/landing/land_plan_test.go: `land` states the plan's one-line form in the proposal's text.
4. Priority, limit and restore
   - criteria: 7, 8
   - needs: 1, 3
   - packages: internal/core/checks/priority.go, internal/core/checks/limit.go, internal/core/checks/restore.go, internal/core/checks/plan.go, internal/core/landing/land.go, internal/core/guard/personacts.go, internal/surface/cli/checks.go, commands/checks.md, .abcd/development/brief/04-surfaces/43-checks.md, .abcd/development/brief/05-internals/03-configuration.md
   - tests: internal/core/checks/priority_test.go: public gives speed and private gives spend, each named with its source; a declared priority wins and is named; under spend a priced optional early point is left out unless declared. internal/core/checks/limit_test.go: a declared pair is used; the default from 19 failures in 300 proposal runs and 3 in 200 landing runs is 1 in 12; fewer than a hundred landings gives 1 in 20; the figures used are returned. internal/core/checks/restore_test.go: past the limit with `early_run` quick, the restore writes `full` with figures and date and nothing else; at or under the limit it writes nothing; after a cut only later landings count. internal/surface/cli/checks_cut_test.go: `checks cut` is refused in an agent's session and without a terminal.
5. The checker's point and the fifty-landing report
   - criteria: 10, 12
   - needs: 3, 4
   - packages: internal/core/checks/plan.go, internal/core/checks/report.go, internal/surface/cli/checks.go, commands/checks.md, .abcd/development/brief/04-surfaces/43-checks.md, .abcd/development/research/notes/check-budget-baseline.md
   - tests: internal/core/checks/plan_test.go: a check naming the independent point shows it as optional and never waited for when a checker is available, and as "not covered before landing" when none is; in neither case does it replace the landing point. internal/core/checks/report_test.go: from a fixture of fifty landed changes, each stated plan is set beside the runs the change had, and a difference is counted.

## Size

Five pull requests. Step 1 is a reader over a forge history the `landings` verb already fetches part of, plus the whole wiring of a new verb, which is most of its diff. Step 2 is a schema, a guarded read and a lint rule, and is small. Step 3 is the largest: the classifier must reproduce both hand-kept lists exactly before either becomes a reader, and it touches the hosted workflow and the preflight's short path, the two places a mistake blocks every landing; dcn-2608160000000002 already ruled that skipping the tests that read records is fail-open, and joining two lists must not loosen either. Step 4 is three pure functions and one person-only act. Step 5 is a slot and a comparison. The fifty landings accrue after step 3, so the spec closes some days after step 5 lands. No step runs a preflight in this planning lane; each step's builder runs the repository's definition of done.
