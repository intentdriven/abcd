---
id: spc-2611241502296629
slug: a-person-can-set-an-old-linux-pc-or-mac
intent: itd-2611011325430933
origin: researcher-authored
production_mode: hand-written
---
# A machine the person owns checks each change independently

## Summary

Delivers itd-2611011325430933 under adr-2611241456294452. A new verb, `abcd checker`, turns a spare machine into a pull-only checker: it polls the shared remote, takes the newest commit of each of the project's own branches, exports it to a fresh directory, lays the default branch's copy of the check definitions over it, runs the declared checks under a second operating-system account that holds no credential, and reports the result through the forge under an identity whose key only a separate signer account on that machine can read. The verdict is never required: no forge rule names it, `abcd land` never waits for it, and a change the checker has not reached lands and is checked afterwards, with the late verdict reported. Several checkers share work by claiming a commit before running it, so the first one free takes it. The person's working machine learns a checker's state (reachable, free, how far behind) from a declared list or, opt-in, by discovery on the local network, and uses that state only to report. Six steps: five pull requests and a proof on a real machine.

## Approach

### What exists today

Nothing of the checker is built. The pieces it stands on are: the declaration of checks and of the files that define them, read from the default branch (itd-2611241456272313); the fixed check directory and reuse of unchanged results (itd-2611011325421304), without which an old machine cannot keep pace; the `behind` field of a landing's result, whose independent source is an interface returning "none" (itd-2610261025516272); the independent point in a change's plan, likewise "none available"; and the settings baseline that pins required checks to their reporting application (iss-2610290913450522), which this spec reads for the identity's shape and does not extend, because nothing is required.

### The checker core, with no forge (step 1)

`abcd checker run` is the loop; `--once` runs one pass for tests. Each pass:

1. Fetches `refs/heads/*` of the project's repository and nothing else. Pull-request refs of forks live elsewhere and are never fetched. The default branch, the reservation and status prefixes and work-in-progress branches are left out of the per-branch work.
2. Picks the newest commit of each branch it has no result for, newest push first. An older commit of a branch pushed again is marked superseded, never passed. When no branch head is waiting it takes the default branch's tip if it has no result for it, which is how a change that landed before the checker reached it is still checked.
3. Exports the commit with `git archive` into a fresh directory outside every working tree (the verifier-works-on-a-copy rule), then overwrites every file the default branch's declaration lists under `definition`, and the declaration itself, with the default branch's copies. The result records each definition file the change had edited.
4. Runs the declared checks that name the independent point, one run at a time under the load gate, as the runner account (step 3), and reads the exit status as the parent process.
5. Stores the result in its own store before anything is reported, so a result survives a lost connection.

A project with no declaration, or whose declaration gives no check the independent point, makes `checker setup` and `checker run` say what is missing and install or run nothing. The tests the checker runs are still the change's own: independence proves the run happened elsewhere with that result, not that the tests are good.

### The identity and the report (step 2)

On the largest forge only an application may create a check, so the identity is one. `abcd checker setup` runs on the checker and walks the one browser step in which the person registers it (dcn-2610180720389628) with the least permission that lets it report a check and nothing that lets it read or write contents; a public repository needs no fetch key, a private one gets a read-only one. Each machine holds its own key for the identity, so one machine can be withdrawn without the others. Tokens are signed with the standard library (RS256), so there is no new dependency. The signer posts each stored result as a check on its commit, naming the machine, its operating system, the commit, the time and any definition file the change had edited. No conclusion that a forge counts as passing is ever posted for a run that did not pass or did not happen: a superseded commit and a withdrawn claim are posted as cancelled. After an absence the signer posts every result it owes, oldest first. Behind the forge seam, a forge that cannot show which identity reported a check gets the same report labelled unpinned, and `setup` says so before anything is installed. Whether the other forges abcd reads can show a reporting identity is established in this step from their documentation and recorded in the brief chapter; the fallback needs no answer.

### The service and the two accounts (step 3)

`checker setup` installs a system service (a systemd unit, a launchd daemon) that runs the released binary, never a checkout's build, and creates two accounts. The signer fetches, exports, starts the runner, reads its exit status and reports; the fetch key and the identity's key are readable by the signer alone. The runner holds nothing, and is denied the private address ranges where the system allows it (a per-unit rule under systemd, a per-user packet-filter rule on macOS), so a change's code cannot reach the person's other machines. A virtual machine for the runner is offered as the stronger option and not required. `setup` checks that the project's declared toolchain runs on the machine and refuses otherwise. `abcd checker status`, run on the checker, prints its queue and its last results.

### Several machines, and how they are found (step 4)

A checker answers one fixed question on the local network: a read-only request for its state (its name, free or busy, the number of commits waiting, the oldest one's age). The listener binds only to private addresses, takes no body and no parameter, and nothing it answers is a verdict. It can be switched off, in which case the checker is known only by what it reports through the forge.

The person's working machine finds checkers in two ways. `checker.hosts` in the person's own `~/.abcd.noindex/config.json` lists them; a repository's file may not set it, and the layered reader refuses the key there. `checker.discover: true`, in the same file only and off by default, makes abcd browse for the checker's service name with the operating system's own discovery tool, on the local link only; where the tool is absent abcd says so and uses the declared list. With neither set abcd looks for nothing.

First one free: before running a commit a checker posts an in-progress check under the shared identity, then reads the commit's checks back; the earliest in-progress check wins, and any other checker withdraws its own as cancelled and moves on. A claim carries the time of its last renewal, and one not renewed within a stated interval is taken over by the next free checker, so a machine that stops mid-check does not strand the commit. On a forge with no such check, each checker runs independently and a commit may be run twice, which costs time and nothing else; `setup` says so.

### A landing reports it and never waits (step 5)

The independent source of `land`'s `behind` field is now real: `land` reads the commit's checks from the forge once, and if a pass under the checker's identity is there, names it. If not, it asks each known checker's state with a short bound and in parallel with its other reads, and its result says one of: no checker is known; the checker is reachable and has this commit waiting, with how far behind it is and since when; or the checker did not answer, with when it last reported. `land` proceeds identically in every case. A test holds that no path through `land` blocks on, retries or reads a setting about the checker, and `abcd checker` offers no flag and no setting that makes its verdict required; `setup` writes no forge rule. A failed verdict that arrives after a landing is shown by `abcd status` and by the next `land` in that project, naming the commit, the check and the machine, until a later commit passes that check on a checker or the person dismisses it with `abcd checker ack <commit>`. `abcd checks plan` shows the independent point as available or not from the same state, and `abcd landings` carries the checker's name, operating system, commit and time on the rows it judged.

### The proof (step 6)

On a Mac, with abcd's own repository as the project: twenty consecutive landings with the checker switched off, showing no landing delayed or refused; then twenty with it on, giving the share of changes with a verdict within an hour of the push and the backlog at the end of each day; and one hostile change, run on the real machine, whose tests try to read the key, post a check and reach a private address. The Linux service is built and tested in steps 1 to 4; its proof on a real machine is welcome and is not a condition of closing.

## How the Acceptance Criteria are met

1. Fetched with no agent's action, run as the default branch declares, reported under its own identity: steps 1 and 2.
2. The default branch's copy of an edited definition file, and the verdict names the edit: the overlay (step 1) and the report (step 2).
3. The runner holds no credential and no route, and only the signer reports: step 3, proved on a real machine in step 6.
4. Off, unreachable, busy or behind, the landing proceeds and names the machine: step 5.
5. A change that landed first is checked later, and a late failure is reported: the default-tip rule (step 1) and the status line (step 5).
6. Several machines, first one free, no commit run twice, a stopped machine loses its claim: step 4.
7. Nothing is looked for unless set, discovery is local only, and state is never a verdict: step 4.
8. Nothing about landing differs and no setting requires the verdict: the test and the absence of the setting (step 5).
9. The newest commit is checked and the older one marked superseded: step 1.
10. Every owed result is reported after an absence: the store (step 1) and the catch-up (step 2).
11. An unpinned forge is said before installing and labelled on every verdict: step 2.
12. A fork's change is never fetched: the ref rule (step 1).
13. No declared check installs nothing and says what is missing: step 1.
14. Every report of a landing names the machine, its system, the commit and the time, apart from a hosted check and the lander's own word: step 5.

## Open Questions

1. resolved: nothing requires the verdict and nothing waits for it (adr-2611241456294452); the earlier design's required check, its carry-over onto queue entries and the hold in `land` are not built.
2. resolved: checkers pull and claim among themselves; the working machine never hands a checker work, since that would let an agent choose what it runs.
3. resolved: the state answer is the only listener, read-only, private addresses only, and can be switched off.
4. resolved: hosts are declared in the person's own configuration and never in a repository; discovery is opt-in and uses the operating system's own tool, so no dependency is added.
5. resolved: one identity with a key per machine.
6. resolved: the proof is on a Mac; the Linux service is built and tested but not a condition of closing.

## Footprint

- packages: internal/core/checker/poll.go, internal/core/checker/pick.go, internal/core/checker/export.go, internal/core/checker/overlay.go, internal/core/checker/run.go, internal/core/checker/store.go, internal/core/checker/identity.go, internal/core/checker/token.go, internal/core/checker/report.go, internal/core/checker/setup.go, internal/core/checker/service.go, internal/core/checker/service_linux.go, internal/core/checker/service_darwin.go, internal/core/checker/accounts_linux.go, internal/core/checker/accounts_darwin.go, internal/core/checker/state.go, internal/core/checker/hosts.go, internal/core/checker/discover.go, internal/core/checker/claim.go, internal/core/checker/verdict.go, internal/core/github/checkrun.go, internal/core/landing/behind.go, internal/core/landing/land.go, internal/core/landings/landings.go, internal/core/checks/plan.go, internal/core/surface/sentences.go, internal/ghcontract/rest-fields.txt, internal/surface/cli/checker.go, internal/surface/cli/cli.go, internal/surface/cli/land.go, internal/surface/cli/status.go, internal/README.md, commands/checker.md, commands/land.md, commands/status.md, docs/reference/cli/commands.md, docs/reference/cli/commands/checker.json, .abcd/checks.json, .abcd/rules.json, .abcd/development/release/surface.json, .abcd/development/release-gate/manifest.json, .abcd/development/brief/04-surfaces/44-checker.md, .abcd/development/brief/04-surfaces/38-land.md, .abcd/development/brief/04-surfaces/39-status.md, .abcd/development/brief/04-surfaces/README.md, .abcd/development/brief/05-internals/03-configuration.md, .abcd/development/research/notes/independent-checker-proof.md
- tests: internal/core/checker/poll_test.go, internal/core/checker/pick_test.go, internal/core/checker/overlay_test.go, internal/core/checker/run_test.go, internal/core/checker/store_test.go, internal/core/checker/token_test.go, internal/core/checker/report_test.go, internal/core/checker/setup_test.go, internal/core/checker/service_test.go, internal/core/checker/state_test.go, internal/core/checker/hosts_test.go, internal/core/checker/discover_test.go, internal/core/checker/claim_test.go, internal/core/checker/verdict_test.go, internal/core/github/checkrun_test.go, internal/core/landing/behind_checker_test.go, internal/core/landing/land_checker_never_waits_test.go, internal/surface/cli/checker_surface_test.go, internal/surface/cli/status_checker_test.go

## Steps

1. The checker core, with no forge
   - criteria: 1, 2, 5, 9, 12, 13
   - needs: none
   - packages: internal/core/checker/poll.go, internal/core/checker/pick.go, internal/core/checker/export.go, internal/core/checker/overlay.go, internal/core/checker/run.go, internal/core/checker/store.go, internal/surface/cli/checker.go, internal/surface/cli/cli.go, internal/core/surface/sentences.go, internal/README.md, commands/checker.md, docs/reference/cli/commands.md, docs/reference/cli/commands/checker.json, .abcd/rules.json, .abcd/development/release/surface.json, .abcd/development/release-gate/manifest.json, .abcd/development/brief/04-surfaces/44-checker.md, .abcd/development/brief/04-surfaces/README.md
   - tests: internal/core/checker/poll_test.go: against a bare fixture remote with branches, a fork's pull-request ref, a reservation branch and a work-in-progress branch, only the project's own branches are fetched. internal/core/checker/pick_test.go: the newest commit of each branch is picked, newest push first; a branch pushed again marks its older commit superseded and not passed; with no branch waiting the default tip is picked once. internal/core/checker/overlay_test.go: a change that edits a declared definition file is run with the default branch's copy and the result names the file; a change that edits the declaration is run with the default branch's. internal/core/checker/run_test.go: the export is outside every working tree and is removed afterwards; no declaration, and no check at the independent point, each run nothing and say what is missing. internal/core/checker/store_test.go: a result is stored before any report and survives a restart. internal/surface/cli/checker_surface_test.go: `checker run --once` on the CLI and the plugin page.
2. The identity and the report
   - criteria: 1, 2, 10, 11
   - needs: 1
   - packages: internal/core/checker/identity.go, internal/core/checker/token.go, internal/core/checker/report.go, internal/core/checker/setup.go, internal/core/github/checkrun.go, internal/ghcontract/rest-fields.txt, internal/surface/cli/checker.go, commands/checker.md, .abcd/development/brief/04-surfaces/44-checker.md
   - tests: internal/core/checker/token_test.go: a token signed from a fixture key verifies against its public half and carries the claims the forge documents; a key the process cannot read is an error naming the file. internal/core/github/checkrun_test.go: the request to create and to conclude a check matches the recorded contract. internal/core/checker/report_test.go: a pass, a failure, a superseded commit and a withdrawn claim are each posted with the stated conclusion, and none of the last two as a conclusion the forge counts as passing; results owed after an absence are posted oldest first and once each; a forge with no reporting identity gets the unpinned label on every verdict. internal/core/checker/setup_test.go: `setup` on such a forge says so before writing anything, and asks for no permission to read or write contents.
3. The service and the two accounts
   - criteria: 3
   - needs: 1
   - packages: internal/core/checker/service.go, internal/core/checker/service_linux.go, internal/core/checker/service_darwin.go, internal/core/checker/accounts_linux.go, internal/core/checker/accounts_darwin.go, internal/core/checker/setup.go, internal/surface/cli/checker.go, commands/checker.md, .abcd/development/brief/04-surfaces/44-checker.md
   - tests: internal/core/checker/service_test.go: the rendered systemd unit and launchd daemon run the installed binary by its absolute path as the signer, start the runner as the second account, and carry the rule denying it the private address ranges; the key files' modes admit the signer alone; a toolchain the machine cannot run is refused at setup. A hostile fixture change, run through the two accounts on a Linux runner in CI, cannot read the key file and cannot post a result.
4. Several machines, and how they are found
   - criteria: 6, 7
   - needs: 2
   - packages: internal/core/checker/state.go, internal/core/checker/hosts.go, internal/core/checker/discover.go, internal/core/checker/claim.go, internal/core/checker/report.go, internal/surface/cli/checker.go, commands/checker.md, .abcd/development/brief/04-surfaces/44-checker.md, .abcd/development/brief/05-internals/03-configuration.md
   - tests: internal/core/checker/state_test.go: the listener refuses to bind a public address, answers the one request with the four fields, and answers every other method, path and body with a refusal and no state change. internal/core/checker/hosts_test.go: `checker.hosts` and `checker.discover` are read from the person's own file and refused, naming the file, in a repository's. internal/core/checker/discover_test.go: with neither key set nothing is asked on the network; with discovery on, the operating system's tool is run for the one service name, and its absence is said and falls back to the declared list. internal/core/checker/claim_test.go: of two checkers claiming one commit the earlier claim runs and the other withdraws as cancelled; a claim not renewed within the interval is taken over; on a forge with no such check both run and the result says so.
5. A landing reports it and never waits
   - criteria: 4, 5, 8, 14
   - needs: 2, 4
   - packages: internal/core/checker/verdict.go, internal/core/landing/behind.go, internal/core/landing/land.go, internal/core/landings/landings.go, internal/core/checks/plan.go, internal/surface/cli/land.go, internal/surface/cli/status.go, commands/land.md, commands/status.md, .abcd/checks.json, .abcd/development/brief/04-surfaces/38-land.md, .abcd/development/brief/04-surfaces/39-status.md
   - tests: internal/core/landing/land_checker_never_waits_test.go: with no checker, a checker that passed, one that is busy, one that never answers and one that answers after the bound, `land` takes the same steps to the same result and differs only in the report; no setting or flag changes that, watched fail against a `land` that waits. internal/core/landing/behind_checker_test.go: a pass under the checker's identity on the exact commit is named with machine, system, commit and time; a pass under any other identity, or on another commit, is not. internal/core/checker/verdict_test.go and internal/surface/cli/status_checker_test.go: a failure reported after the landing is shown by `status` and by the next `land`, naming commit, check and machine, until a later commit passes that check or `checker ack` dismisses it.
6. The proof on a real machine
   - criteria: 3, 4, 5
   - needs: 3, 5
   - packages: .abcd/development/research/notes/independent-checker-proof.md
   - tests: the note's twenty landings with the checker off and twenty with it on, the hostile change run on the real machine, and each of the Mechanism's five falsifiers answered from them; the note names no machine, host or person

## Size

Five pull requests and a proof. Step 1 is the largest piece of new code and needs no forge, so it is tested entirely against a local bare repository. Step 2 depends on the one act only the person can take, registering the identity in a browser, so it cannot be finished unattended. Step 3 is the whole of the claim that no agent can produce the verdict, and it differs per operating system: the rendered units are tested everywhere, the Linux accounts on a hosted runner, and the macOS accounts only on a real machine in step 6, which is the main risk. Step 4 adds the only listener in the design and must stay read-only. Step 5 is small and is where the product thinker's ruling is enforced by test. Throughput on an old machine is outside this spec's control and is why itd-2611011325421304 comes first; the Mechanism's backlog falsifier is answered in step 6. No step runs a preflight in this planning lane; each step's builder runs the repository's definition of done.
