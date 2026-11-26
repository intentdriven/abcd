---
id: spc-2611251757236507
slug: one-abcd-command-starts-a-session-on
intent: itd-2611251725203320
origin: researcher-authored
production_mode: hand-written
---
# one-abcd-command-starts-a-session-on

## Summary

`abcd start [--profile <name>] [--permission-mode <mode>] [prompt]` is the one command a person types, in a plain terminal in a checkout abcd manages, to work in a session on abcd alone (itd-2611251725203320; the name is dcn-2611251757251202's). A run has one of two outcomes, never both:

- **No profile on the account.** It makes one directory under `~/.abcd.noindex/profiles/`, writes abcd's settings there and nothing of the person's, prints the host's own login command, and starts nothing. Logging in stays the person's act, in their own shell.
- **A profile that is logged in.** It runs one short launch with the session's arguments on that profile, written to the launch ledger, refuses naming anything that launch reports which the printed list does not name, and otherwise replaces the abcd process with the host: the tools on the printed list, abcd's hooks handed over inline, the repository's conventions file whole as a private file the session reads, no settings source, no server and no server's tool, no browser tool, no plugin but the host's own, the person's home and none of a stage's files.

The approach is the drive's wall turned to a session, not a second wall. `internal/core/runner` gains one file, `session_start.go`, beside `profile.go`: making a profile, the session's arguments, environment and inline settings, the copy of abcd its hooks name, and the judge of the short launch. `internal/surface/cli` gains one verb file, `start.go`, which orders the refusals, prints the lines and execs in place as `drain_drive_update.go` does. The drive's `wallHooks`, its settings file and its stage files are unchanged, `drain_drive.go` is not edited, and `wall.go` is not edited in step 1. A session is handed its settings at the start and reads no settings source, so no file a drive reads is written.

It is built as two steps. Step 1 is the whole of the code, the pages and every list that pins a verb; it lands on the main branch with no `Delivers:` trailer. Step 2 is the person's live proof on their own login; the spec stays open on it and the intent stays planned (the intent's decision 2). The writer of this spec launched nothing. What was tested by hand on 2026-11-25 on host 2.1.332 is said where it is relied on; every other claim about the host is listed under "What only the live proof can show".

## Footprint

- packages: internal/core/runner/session_start.go (new), internal/core/runner/session_start_test.go (new),
  internal/core/runner/live_session_test.go (new), internal/core/runner/main_test.go,
  internal/core/runner/profile.go, internal/core/runner/profile_test.go, internal/core/runner/wall.go,
  internal/core/runner/testdata, internal/core/refusal/refusal.go,
  internal/core/surface/sentences.go, internal/core/surface/examples.go,
  internal/surface/cli/start.go (new), internal/surface/cli/start_test.go (new),
  internal/surface/cli/start_hooks_parity_test.go (new), internal/surface/cli/cli.go,
  internal/surface/cli/barerender.go, internal/surface/cli/facts_test.go,
  internal/surface/cli/helpgroups.go, internal/surface/cli/helpgroups_test.go,
  internal/surface/cli/off_profile.go, internal/surface/cli/off_profile_test.go,
  internal/surface/cli/refusal_raisers_test.go, internal/surface/cli/refusal_codes_test.go, internal/surface/cli/note_line_test.go, internal/core/frontmatter/delimiter_canonical_test.go, Makefile,
  commands/start.md (new), commands/drain.md, commands/mode.md,
  .abcd/development/brief/04-surfaces/43-start.md (new), .abcd/development/brief/04-surfaces/README.md,
  .abcd/development/brief/04-surfaces/01-ahoy.md, .abcd/development/brief/04-surfaces/26-mode.md,
  .abcd/development/brief/04-surfaces/35-drain.md, .abcd/development/brief/05-internals/03-configuration.md,
  .abcd/development/release-gate/manifest.json, .abcd/rules.json,
  docs/reference/cli/commands.md, docs/reference/cli/commands/start.json (new),
  docs/reference/cli/commands/abcd.json, .abcd/development/release/surface.json,
  .abcd/development/release/refusal-codes.json,
  .abcd/development/specs/open/spc-2611251757236507-one-abcd-command-starts-a-session-on.md,
  .abcd/development/specs/closed/spc-2611251757236507-one-abcd-command-starts-a-session-on.md,
  .abcd/development/intents/planned/itd-2611251725203320-one-abcd-command-starts-a-session-on.md,
  .abcd/development/intents/shipped/itd-2611251725203320-one-abcd-command-starts-a-session-on.md
- tests: internal/core/runner/session_start_test.go, internal/core/runner/live_session_test.go,
  internal/core/runner/profile_test.go, internal/surface/cli/start_test.go,
  internal/surface/cli/start_hooks_parity_test.go, internal/surface/cli/off_profile_test.go,
  internal/surface/cli/helpgroups_test.go, internal/surface/cli/facts_test.go,
  internal/surface/cli/refusal_raisers_test.go, and the standing tests named under
  "The lists that pin a verb"; step 2 is the person's terminal and the open session's own listings

## Steps

1. **`abcd start`, whole: the core file, the verb, the pages and every list that pins a verb** (MUST, tonight, one implementer)
   - priority: MUST, before the unattended run starts (dcn-2611251726008978)
   - login: none of the writing session's; the live proof of the short launch runs on a profile the person has logged in, and skips where there is none
   - needs: none
   - packages: internal/core/runner/session_start.go (new), internal/core/runner/session_start_test.go (new), internal/core/runner/live_session_test.go (new), internal/core/runner/main_test.go, internal/core/runner/profile.go, internal/core/runner/profile_test.go, internal/core/refusal/refusal.go, internal/core/surface/sentences.go, internal/core/surface/examples.go, internal/surface/cli/start.go (new), internal/surface/cli/start_test.go (new), internal/surface/cli/start_hooks_parity_test.go (new), internal/surface/cli/cli.go, internal/surface/cli/barerender.go, internal/surface/cli/facts_test.go, internal/surface/cli/helpgroups.go, internal/surface/cli/helpgroups_test.go, internal/surface/cli/off_profile.go, internal/surface/cli/off_profile_test.go, internal/surface/cli/refusal_raisers_test.go, internal/surface/cli/refusal_codes_test.go, internal/surface/cli/note_line_test.go, internal/core/frontmatter/delimiter_canonical_test.go, Makefile, commands/start.md (new), commands/drain.md, commands/mode.md, .abcd/development/brief/04-surfaces/43-start.md (new), .abcd/development/brief/04-surfaces/README.md, .abcd/development/brief/04-surfaces/01-ahoy.md, .abcd/development/brief/04-surfaces/26-mode.md, .abcd/development/brief/04-surfaces/35-drain.md, .abcd/development/brief/05-internals/03-configuration.md, .abcd/development/release-gate/manifest.json, .abcd/rules.json, docs/reference/cli/commands.md, docs/reference/cli/commands/start.json (new), docs/reference/cli/commands/abcd.json, .abcd/development/release/surface.json, .abcd/development/release/refusal-codes.json, .abcd/development/specs/open/spc-2611251757236507-one-abcd-command-starts-a-session-on.md
   - tests: internal/core/runner/session_start_test.go, internal/core/runner/live_session_test.go, internal/core/runner/profile_test.go, internal/surface/cli/start_test.go, internal/surface/cli/start_hooks_parity_test.go, internal/surface/cli/off_profile_test.go, internal/surface/cli/helpgroups_test.go, internal/surface/cli/facts_test.go, internal/surface/cli/refusal_raisers_test.go
   - done: every test under "The tests, in the order they are written" passes and was watched failing on an assertion before its code was written; the live proof of the short launch ran on a logged-in profile and its log is in the local tier, or it skipped for want of a login and the return says so in one line; `make precheck` is clean; the three generating commands leave the tree unchanged when run again; `scripts/abcd.sh start` from an agent's shell refuses with `start/outside-a-terminal` and makes nothing beyond what every abcd command lays in abcd's home (`.gitignore` and `gitignore-laid`); this step carries its `landed:` line; the change is on a branch with no `Delivers:` trailer and this spec still open.
   The order of work inside the step: the core file with its tests; the verb with its tests; the live proof of the short launch; the reworded lines and the rosters; the pages and the brief; the three generating commands; `make precheck`. Each section's tests are run by name with `-run`, one red run and one green: one run of the whole `internal/surface/cli` package took about twenty minutes on 2026-11-25 with other lanes queued at the load gate, so the whole package is run once, at the end. No stand-in test starts the host: the runner's tests play the harness from the test binary through `fakeHarness`, and the verb's tests stand a script named `claude` on `PATH` as `newDriveWorld` does, one that answers `auth status --json`, prints a canned event stream for a print launch and writes its own arguments and environment to a file the test reads. A test that fails only because a name does not compile has not been watched failing: add the signature with an empty body first, run the test, read its failure, then write the body.
   The live proof of the short launch is the writing session's, straight after the verb's tests pass and before any page is written, because the pages describe what that run may change. It is `TestTheShortLaunchHoldsOnALoggedInProfile` in `live_session_test.go`, behind the `live` build tag as `live_effort_test.go` is, run as `GOFLAGS=-trimpath go test -tags live -count=1 -v -run TestTheShortLaunchHoldsOnALoggedInProfile ./internal/core/runner` with its output saved to the main checkout's `.abcd/.work.local/logs/`. It sits in `package runner_test` and calls only what the verb calls: it finds the account's profile with `DriveProfile` (the name from `ABCD_LIVE_PROFILE` where the account holds several), asks `CheckLogin`, keeps a copy of the checkout's own build with `KeepSessionBinary` (`.abcd/.work.local/bin/abcd`, which any `scripts/abcd.sh` call has made; the test's own binary is not abcd), takes the host's path from `SessionHarness` and the ledger from `runner.Load` over the person's home and this checkout, builds the `Session` the verb would build for this checkout with no mode and no prompt, and calls `ProveSession`. It skips loudly, on stderr too and through a helper of its own (`skipLoudly` in `live_effort_test.go` words its line for another check, and that file is not edited), where no host is on `PATH`, the account holds no profile, or the profile is not logged in: the login is the person's, and the writing session never performs it and never runs `abcd start`, which refuses in its shell. It spends a little of the subscription and writes one ledger entry. A wall it names is fixed in `session_start.go`, with a test, before the pages are written.
2. **The person's live proof, on their own login** (live; the spec stays open on it)
   - priority: MUST, when the person is back at their terminal; not a reason to hold step 1 off the main branch (the intent's decision 2)
   - login: the person, once, with the host's own login command, in their own shell
   - needs: 1
   - packages: internal/core/runner/session_start.go, internal/core/runner/session_start_test.go, internal/core/runner/testdata, internal/core/runner/wall.go, internal/surface/cli/start.go, internal/surface/cli/start_test.go, .abcd/development/specs/open/spc-2611251757236507-one-abcd-command-starts-a-session-on.md, .abcd/development/specs/closed/spc-2611251757236507-one-abcd-command-starts-a-session-on.md, .abcd/development/intents/planned/itd-2611251725203320-one-abcd-command-starts-a-session-on.md, .abcd/development/intents/shipped/itd-2611251725203320-one-abcd-command-starts-a-session-on.md
   - tests: the person's terminal, the open session's own listings and its transcript read afterwards; the receipt's named check `a-started-session-names-no-mcp-tool`, read from the open session's own tool list; a dated receipt in the main checkout's `.abcd/.work.local/scratch/`; the short launch's first event line saved as a fixture under `internal/core/runner/testdata/` and read by the judge's test; `TestStartRefusesWhenTheCopyOrTheHostChangedSinceTheLaunch` in `start_test.go`
   - done: the person has logged in once, a session opened by `abcd start` shows from inside each item under "What only the live proof can show", the receipt is written, the two changes below are made with their tests, this step carries its `landed:` line, and the spec is closed with `scripts/abcd.sh spec close spc-2611251757236507` in a change carrying `Delivers: itd-2611251725203320`.
   The person runs `abcd start` (it makes the profile and prints the login command), runs that command, and runs `abcd start` again. A wall the short launch names, or an item below that the open session does not show, is a finding: it is fixed in `session_start.go` or `start.go` in this step with a test, or it stops the route and goes back to the product thinker. The Mechanism's falsifier is read here too: the session's transcript, read afterwards, holds no tool outside the printed list, no plugin but the host's own, none of the person's instructions or memory, no connected service, and shows abcd's guard and rules at work.
   Two changes wait for this step. The first is the re-read before the exec: the host's file and the kept copy are read again by hash immediately before the exec, and a difference from what the launch ran is refused as `start/wall-not-held` (`SessionBinaryHolds`, and the test named above). The second is one string in `wall.go`: once the login command has been seen to work, the drive's logged-out wall prints `LoginCommand(p)` in place of the by-hand route; `wall_reprove_test.go` reads only the words "log in" and stands. Until then the drive's present line is the one login route known to work.

## Decisions this spec makes

Each is the writing session's, with no question put, for the product thinker's review on return; each says how to reverse it.

1. **The default profile name is `drive`.** Every shipped page and the drive's no-profile refusal already name it, `DriveProfile` chooses the one directory an account holds whatever it is called, and it names no provider and no account. `runner.DefaultProfileName` holds it. The name list of itd-2611040344170206's decision 15 replaces it; to reverse, change the constant before the release.
2. **A profile is made only on an account that holds none.** With no profile, `abcd start` makes `drive`, or the plain name `--profile` gives. With one or more profiles, a `--profile` name the account does not hold is refused listing the ones it holds, and makes nothing: a mistyped name must not make a second login. Several profiles and no name is refused listing them, as the drive refuses. A second profile is still made by hand, as `commands/drain.md` shows. To reverse, let a name not held be made; criterion 1's given is "an account with no profile", so this spec stops there.
3. **What the making writes.** The directory at `0o700` through `fsutil.EnsureHomeDeclarationScope`; `settings.json` at `0o600`, byte for byte `wallSettings(<kept copy>)`, so a drive's start finds a file abcd rendered and reports nothing replaced; and an empty `skills` folder at `0o500`, the state `SetAsideSyncedSkills` leaves, so the host's login cannot sync the account's skills in. Nothing else. Without the closed folder, criterion 2 (no file in the profile is changed on a later run) and criterion 5 (none of the person's skills) cannot both hold for a profile the host has synced skills into.
4. **A profile already there is only read.** No `WriteWallSettings`, no `SetAsideSyncedSkills`. `abcd start` reads the entries `wallLoaded` names by the drive's own rule (`checkStageFiles`: present by `Lstat`, unless it is an empty directory) and refuses naming the first; a hand-made profile holding synced skills is refused with the two ways on, a drive's start (which sets them aside) or moving them out by hand.
5. **The permission modes admitted are the six host 2.1.332 lists:** `manual`, `acceptEdits`, `plan`, `auto`, `dontAsk`, `bypassPermissions`. Any other value is refused as a usage error and not repeated. None named means no `--permission-mode` argument at all, which is the host's default; a named one goes as the single argument `--permission-mode=<mode>`. No file is read for it, a repository's or the machine's.
6. **The environment.** The session carries the person's environment, `HOME` and `PATH` included, less: the eight names in `credentialVars`; every other variable whose name opens `ANTHROPIC_` or `CLAUDE_`; every `ABCD_STAGE_` variable and `ABCD_PERSON_HOME`. It then sets `CLAUDE_CONFIG_DIR=<the profile>`, `CLAUDE_CODE_DISABLE_CLAUDE_MDS=1`, `CLAUDE_CODE_DISABLE_AUTO_MEMORY=1`, `DISABLE_UPDATES=1`, and the web-connector switch `ENABLE_CLAUDEAI_MCP_SERVERS=false`, as the hand-written script of 2026-11-25 sets it. It sets none of the drive's `CLAUDE_ENV_FILE`, `ABCD_STAGE_HOME` or `ABCD_STAGE_PROTECT`. A credential variable is cleared and named, where the drive refuses: the terminal is the person's and often holds one for other tools, and the short launch then reads `apiKeySource`. "No key in its environment" is read as none of the variables the host signs in or bills through; any other secret of the person's shell stays, since the shell is theirs.
7. **`StageProtectEnv` does not apply to a session.** `guard.Files.WithStage` changes nothing unless the stage home is absolute, and a session has no stage home; giving it one would make every hook treat the person's session as a stage. The launch argument `--disallowedTools=` carries `walledDirRule(dir)`, a `Read(...)` rule: the Read tool is kept out of the profile directory; Edit, Write and the shell are not. The lines and pages say so: a profile is not a sandbox.
8. **The hooks name a copy of abcd kept under abcd's home, addressed by content.** `~/.abcd.noindex/session-bin/<sha256 of the running binary>/abcd`, made after the login check and before the short launch. The bytes the short launch ran are then the bytes the session's hooks start with for its whole life, whatever `abcd update`, a drive's self-update, a rebuild by `scripts/abcd.sh` or a worktree's removal does to the file the person typed; `scripts/abcd.sh` rebuilds the binary from the very tree the session edits, so naming that file would let the session change its own hooks. One rule covers every binary, a build under a checkout's `.abcd/.work.local/bin` included: nothing is refused, the copy is what is named. The profile's own `settings.json` names the same copy when `abcd start` makes it. Nothing here reclaims a copy, since a live session cannot be told from an ended one; the cost is one copy for each distinct build a session is started from, not for each start, and the page names the folder. The re-read of the copy and of the host's file immediately before the exec is step 2's.
9. **The short launch runs in an empty repository of its own, in `dontAsk`, with abcd's prompt.** It cannot carry the session's arguments unchanged: a print launch needs `--print`, an output format and a prompt of its own, `plan` would keep the model from the one call the proof reads, and a proof must not run in the bypass mode. So it is the session's arguments less the permission mode and the person's prompt, plus the print flags, and the start's line says exactly that.
10. **The launch-ledger entry** is one `RecordLaunch` with `LaunchStart{Role: "session-proof", Runner: Claude, Repo: <checkout root>, Fallback: "session-proof"}`. The session itself is never written. `runner.claude` need not be enabled on the machine: the session is the person's own, not a route.
11. **"No terminal on all three streams" is read as: each of standard input, output and error must be a terminal**, as `ahoy connect` reads it (`connectIsTerminal`). With the session markers, those are the only two things checked about where the command runs.
12. **The conventions file is `AGENTS.md` at the root of the checkout the person stands in** (`loop.ConventionsFile`), refused when absent, not a regular file, holding a NUL byte, or over 100 KiB, so "whole" has a bound and the file is never cut. It is read through `os.OpenRoot(<checkout root>)` and `fsutil.ReadGuardedInRoot(root, loop.ConventionsFile, 100<<10)`; a link at that name is refused, never followed, since following one would hand a file from outside the checkout to the host as its instructions. It is the file as it stands at the start; a later edit does not reach an open session. It goes over by path, not as text: `KeepSessionConventions` writes one line naming the file, a blank line and the file whole to `~/.abcd.noindex/session-conventions/<sha256 of that text>/conventions.md`, through `fsutil.EnsureHomeDeclarationScope(home, abcdhome.Rel("session-conventions", digest), 0o700)` and an atomic write at `0o600`, read back by hash; a copy there with that text is reused, and a link or another account's file there is refused. The short launch and the session take `--append-system-prompt-file <path>`, so no argument carries the text. A host fact, tested by hand on 2026-11-25 on host 2.1.332: an interactive session started with `--setting-sources ""` and `--append-system-prompt-file <path>` answered with a word only that file held, and print mode takes the same flag. abcd reclaims no copy; one may be cleared only when no session started from it can still be open.
13. **`--json` adds nothing of this verb's.** A refusal is the framework's envelope; the lines are text on standard output whatever the flag. The terminal check puts every run that gets past it at a terminal, so only the refusal envelope is ever read by a program.
14. **iss-2611032326573729 stays open.** Its remedy asks for more than this delivers (a registry, random names, ahoy's report of each profile), which stays with itd-2611040344170206. The landing change carries no `Resolves:` trailer for it.
15. **Every server's tool is denied by rule, and the code names four plugins and no more.** The inline settings carry `"permissions":{"deny":["mcp__*"]}`, in the short launch and in the session alike. `--tools` bounds the host's own tools; the deny rule bounds every server's, a built-in plugin's registered tool included, without naming one. Tested by hand on 2026-11-25, host 2.1.332, print mode: with the rule in the inline settings a probe server stayed connected and its tool left the reported tool list, and `mcp__<server>` alone did the same. `enabledPlugins` switches off the four built-in plugins the committed fixture reports, and no other; no file is read for a name and no flag adds one.

## The command

`start.go` holds `newStartCommand()`, registered in `cli.go` beside `newSweepCommand`, placed in `helpgroups.go` under the set-up group (a person's verb, `block: people`). It takes no `asJSON`, since it renders no object.

- `Use: "start [prompt]"`, `Args: cobra.MaximumNArgs(1)`. An empty prompt is no prompt.
- `--profile <name>`: the profile to start on; on an account with none, the name the one made is given. Held to `runner.PlainProfileName`; a value that is not plain is refused with `usage/invalid-flag-value` and not repeated.
- `--permission-mode <mode>`: declared with `enumFlag` over `runner.SessionPermissionModes()`, so the help, the membership check and the command facts read one slice. `closedSets()` in `facts_test.go` gains `{"abcd start", "permission-mode"}: runner.SessionPermissionModes(),`, which `TestEnumFlagAnnotationsAreTheReadersSlices` counts.
- The long help says what the two outcomes are, that the login is the person's and abcd never reads, stores or forwards it, that a profile is a separate login and settings on this one account and not a sandbox, that the short launch spends a little of the subscription and is written to the ledger, and that `abcd session` is another thing (the live brief sessions). It spells no host binary's name beyond what the reference already holds; the login command is printed at run time.
- Bare `abcd start` is the start itself, not a render. `bareRenderExceptions` in `barerender.go` gains `"start": "bare is the start itself: it makes the account's first profile or hands the terminal to the host, so it is not a render; anywhere but a person's terminal it refuses (exit 2) and writes nothing",`, and the `## Bare invocation` section of the surfaces README gains: "Bare `abcd start` is the start itself, so it acts rather than renders; anywhere but a person's terminal it refuses and writes nothing." Held by `TestEveryTopLevelVerbRendersStateBareOrIsAnException` and `TestTheBriefEnumeratesEveryBareRenderException`.

## The order of the refusals

Each refusal is written `refused(2, refusal.Start…, msg)` at its own call site: `TestExitErrorIsBuiltOnlyThroughItsConstructors` wants a literal `2` and a `refusal.` selector, so a helper taking the code as a parameter must call `exitWith`. Its text goes through `termsafe.Sanitize(fsutil.RedactHome(...))`, opens `abcd start: ` and closes by saying that no session was started. `start.go` adds no `exitError` literal, so `.abcd/development/release/refusal-uncoded.txt` does not grow. Nothing of this verb's is written anywhere before item 10, except in the profile-making outcome of item 5, which writes the copy of abcd and the profile and nothing else; every abcd command lays `.gitignore` and `gitignore-laid` in abcd's home, and this one is no exception.

1. A session marker is set (`CLAUDECODE`, `CLAUDE_CODE_SESSION_ID`, `CLAUDE_CODE_ENTRYPOINT`; the list is `driveSessionMarkers`, read, not copied). `start/outside-a-terminal`, naming the variable.
2. Standard input, output or error is not a terminal. `start/outside-a-terminal`, naming the stream.
3. A flag value: `--permission-mode` outside the six, `--profile` not a plain name. `usage/invalid-flag-value`. A second positional is the framework's `usage/extra-argument`.
4. No absolute home is known. A fault, `exitWith(1, refusal.None, msg)`, saying so.
5. The profile. None on the account: keep the copy of abcd (decision 8), make the profile (decisions 1 to 3), print the lines, exit 0, start nothing; the checkout is not looked at, since the profile is the account's. Several and no name; a name the account does not hold; a directory `DriveProfile` does not trust (a link, another account's, writable by group or other, a path a deny rule cannot hold): `start/profile-unusable`, with `DriveProfile`'s own words where it has them.
6. The profile holds something a launch would load (`runner.ProfileLoads`). `start/profile-unusable`, naming the entry.
7. The checkout: not inside a git checkout (`gitutil.Toplevel`); not one abcd manages (`ahoy.ManagedContext`); the conventions file absent, a link, not regular, holding a NUL byte or over the bound. `start/repository-unready`.
8. The host is not on `PATH` or lies where `runner.SessionHarness(p, repoRoot)` does not admit it; abcd cannot name its own binary (`runner.SelfBinary`); the machine's runner configuration cannot be read, so no ledger opens (`loadRunners`). `start/host-not-runnable`. The path `SessionHarness` answers is the one the short launch runs and the one the exec replaces abcd with.
9. The login, asked as the drive asks it (`runner.CheckLogin`): logged out is `start/not-logged-in`, with the login command on the line; no answer is `start/no-verdict`. Only the kind of `CheckLogin`'s answer is read; the line is this verb's own.
10. The copy of abcd cannot be kept or does not read back as the running binary. `start/host-not-runnable`.
11. The short launch: the ledger cannot record it, a limit was met, it did not finish, or an interrupt ended it: `start/no-verdict`, with the reset time when the host named one; nothing waits and nothing retries. A wall not held: `start/wall-not-held`, naming the tool, server, plugin or key, or saying the guard's refusal was not seen, with the host's version.

Then the exec. The run holds a `signal.NotifyContext` from its first launch, so an interrupt ends the launch through the launcher's own kill of its process group, and is refused under item 11; the handling is stopped before the exec, so the host starts with default dispositions. An exec that returns is a fault, `exitWith(1, refusal.None, msg)`, printing `abcd start: the host could not be started (<error>); no session is open`.

The seven codes are registered in `internal/core/refusal/refusal.go`, each with its summary and explanation and `RaisedBy: []string{"abcd start"}`. `usage/invalid-flag-value` is left as it is: a usage code names no raiser (`TestEveryEntryIsComplete`). A next command must name a command the tree has (`TestEveryNextCommandAndRaiserIsACommandTheTreeHas`), and `abcd explain` prints it as "Try next:", so the first row must not send an agent back to `abcd start`:

| Constant | Code | Next |
|---|---|---|
| `StartOutsideATerminal` | `start/outside-a-terminal` | `NoNext` |
| `StartProfileUnusable` | `start/profile-unusable` | `NoNext` |
| `StartRepositoryUnready` | `start/repository-unready` | `"abcd ahoy"` |
| `StartHostNotRunnable` | `start/host-not-runnable` | `NoNext` |
| `StartNotLoggedIn` | `start/not-logged-in` | `NoNext` |
| `StartNoVerdict` | `start/no-verdict` | `"abcd start"` |
| `StartWallNotHeld` | `start/wall-not-held` | `NoNext` |

Codes are US English and named for what the refusal means (dcn-2611222121290312). Another lane is at work on refusals: run `scripts/abcd.sh peers` before editing `refusal.go`.

The probes in `refusal_raisers_test.go` need seven worlds, not one fixture. `startRefusalWorlds` in `start_test.go` maps each of the verb's codes to the setup of its world, `raiserProbe` is left as it is, and a probe whose code the map holds runs inside `t.Run(p.want, …)` after that setup. The seven start probes all run `{"start"}` and differ in the world `newStartWorld(t)` (defined in `start_test.go`) is set to: a session marker set; two profiles and no name; a checkout abcd does not manage; no `claude` on `PATH`; a stand-in answering logged out; a stand-in whose print launch ends with no event and exits non-zero; a stand-in whose first event reports a server.

## The profile

`MakeProfile` is the only writer. It refuses a name that is not plain and an account that already holds a profile, makes the directory, writes the settings and the closed `skills` folder, and returns what `DriveProfile` then returns for it, so the two can never disagree about what a profile is. `ProfileNames` reads the folder as `DriveProfile` reads it (directories and links, sorted), which is how the verb tells "none" from "a name not held" without reading an error's words. `LoginCommand` renders the one line, in its `~` form:

```text
CLAUDE_CONFIG_DIR=~/.abcd.noindex/profiles/<name> claude auth login
```

abcd never runs it. The login check is `claude auth status --json`, one field read (`loggedIn`), nothing else kept, exactly the drive's.

`profile.go` changes in one place: the no-profile refusal's `makeOne` text names `abcd start` in a plain terminal as what makes the profile and prints the host's login command, in place of the `mkdir` line, and still says abcd never logs a host in. `TestNoWalledDirectoryRefusesNamingTheOneToMake` changes with it.

## The session's arguments, environment and settings

The arguments, in order, each one element unless marked:

1. `--settings`, then the inline JSON (two elements, as tested on 2026-11-25).
2. `--setting-sources`, then the empty string (two elements, as tested).
3. `--strict-mcp-config` and `--mcp-config={"mcpServers":{}}`.
4. `--no-chrome`.
5. `--tools=Bash,Read,Edit,Write,Agent,TaskStop,AskUserQuestion,Monitor,ScheduleWakeup,CronCreate,CronList,CronDelete`.
6. `--disallowedTools=` with `walledDirRule(dir)`.
7. `--append-system-prompt-file` and the path of the kept copy of the conventions text, as two arguments.
8. `--permission-mode=<mode>`, only when the person named one.
9. `--` and the prompt, only when the person gave one.

The inline settings are one JSON object, the same in the short launch and in the session:

- `hooks`: the session's set below, each command the kept copy's path in single quotes and the verb, as `wallHooks` spells it.
- `attribution`: `commit` and `pr` empty, as the drive's.
- `enabledPlugins`: the four built-in plugins the committed fixture `internal/core/runner/testdata/claude-init-2.1.319.jsonl` reports (`cc-plugin-agents-md`, `cc-plugin-telemetry`, `cc-plugin-privacy`, `cc-plugin-plugin-authoring`), each set to false under `<name>@builtin`, and no other. The first is the one that loads the conventions file by itself, so switching it off is what makes the kept copy the only one.
- `permissions`: `deny` holding the one rule `mcp__*` (decision 15).

The environment is one function over a parent environment (decision 6), used twice: over the launcher's carried, git-scrubbed environment for the short launch, which runs in a repository of its own where an inherited `GIT_DIR` would misdirect it, and over `os.Environ()` for the exec. It returns the names it cleared, sorted, which the verb prints, names only.

## The hook set and the manifest

`SessionHooks()` is a table of event, matcher and verbs in `session_start.go`. It holds every event the plugin's manifest wires, and the one at a sub-agent's start:

| Event | Matcher | Verbs |
|---|---|---|
| `UserPromptSubmit` | | `hook prompt-router` |
| `SessionStart` | | `hook session-start`, `hook prompt-router-reset` |
| `PreToolUse` | `Bash\|AskUserQuestion\|Write\|Edit\|MultiEdit\|NotebookEdit` | `guard hook` |
| `PostToolUse` | `AskUserQuestion` | `hook question-answered` |
| `PermissionRequest` | `Bash` | `hook permission-request` |
| `PreCompact` | | `hook prompt-router-reset` |
| `SessionEnd` | | `hook session-end` |
| `SubagentStop` | | `hook subagent-stop` |
| `Stop` | | `hook stop` |
| `StopFailure` | | `hook stop-failure` |
| `SubagentStart` | | `hook subagent-start` |

Two of these, `PostToolUse` and `PermissionRequest`, the drive's `wallHooks` does not hold; that table is left as it is. Timeouts and status messages are the manifest's own, for a binary it may have to download, and are not copied: the session's hooks take the host's default, as the drive's do.

`hook subagent-start` is built in parallel by spc-2611251757228271 and is not in this tree. So the verb filters the table before it renders the settings: an event whose verb the binary lacks is left out and said in one line, never wired to a verb that would fail. The filter takes the command tree as an argument (`findByPath` over it), and `TestStartLeavesOutAHookVerbTheBinaryLacks` passes it a tree without the verb, so the test still tests something once the parallel spec has landed.

The parity test, `TestSessionHooksWireEveryManifestEventToTheSameVerb` in `start_hooks_parity_test.go`, decodes `hooks/hooks.json` itself for each event's matcher and command, reads each command's script through `resolveHookCommand`, and takes as the event's verbs every match of `"\$r/abcd" ((?:hook|guard) [a-z][a-z-]*)` in the script, in the order written. It holds: every manifest event is in the table with the same matcher and the same verbs in the same order; the one table event the manifest may lack is `SubagentStart`; and when the manifest holds it, matcher and verb must agree. It passes before the parallel spec lands and after. The manifest is the authority on that event's matcher.

The session's hooks run the kept copy and nothing else. The plugin's shims are what hand a hook over to a dogfood build, and this session runs no shim, so in a checkout listed in `dogfood-roots` its hooks are the build that typed `abcd start`, not the reviewed branch's. The page says so.

## The short launch

`ProveSession` makes a temporary directory, makes it a repository with `wallProofRepository`, and lays down three things a launch must not read: `.claude/settings.json` and `.claude/settings.local.json` from `workingTreeSettings(marker)`, and a `.mcp.json` naming one server under a fixed name, whose command does nothing. It runs the host there once, through a launcher with `gateWaiting = nil` as `ProveWall`'s is, with the session's arguments less the permission mode and the person's prompt, plus `--print --output-format stream-json --verbose --include-hook-events --no-session-persistence --permission-mode=dontAsk`, then `--` and abcd's prompt: use the Bash tool once for `guardProofCommand`, exactly as written, do nothing else, then answer with the one word done. The bound is `wallProofTimeout`. The whole call sits inside `RecordLaunch` (decision 10), so a launch the ledger cannot hold is not started.

The guard's refusal is read as `judgeWallProof` reads it: a `system` event of subtype `hook_response` whose `hook_event` is `PreToolUse` and whose `stdout` holds `guardRefusalText`. The hook was run by hand on that payload, from a copy under a scratch home's `session-bin`, in an empty unmanaged repository and in a managed one, with no stage variable set: it answered exit 0 with a deny decision whose reason opens with that text, so the reading holds outside a stage.

`judgeSessionProof` reads every line leniently and keeps only these fields of the `init` event:

- `tools`: each must be on the list, with `Task` admitted for `Agent` (the host reports the sub-agent tool under that name). One that is not is `WallTools`, named; a server's tool that the deny rule failed to remove is caught here. A listed tool the launch does not report is returned as unreported and said, not refused: a print launch does not report `AskUserQuestion`.
- `mcp_servers`: must be empty. Any entry is `WallServers`, named; the planted one above all. The deny rule does not loosen this: a reported server is refused whether or not its tools are gone.
- `plugins`: each must be `harnessOwn()`. Any other is `WallPlugins`, named.
- `slash_commands`, `agents`, `skills`: an entry opening `abcd:` is `WallPlugins`, named, since abcd's own pages and agents are not in the session. The rest of `skills` and `agents` are returned by name for the line; they are the host's built-in ones.
- `apiKeySource`: must be `none`. Empty is `WallKey` (nothing shows no key is in use); anything else is `WallKey`.
- No `init` event at all, from a launch that exits clean, is `WallTools`; a launch that ends with no event at all and exits non-zero did not finish, and is no verdict.

After the stream: no guard refusal seen is `WallGuard`; the marker file present is `WallWorkingTree`. A limit (`claudeRateLimit`) is a `LimitError` and a launch that failed or whose result reports an error is a `NoVerdictError`, both before any wall is judged, as in `ProveWall`. Every `WallError` carries the host's version (`withHarnessVersion`).

The runner's tests play the harness from the test binary. In `fakeHarness` in `main_test.go`, beside the `wall-` line, add `if strings.HasPrefix(mode, "session-") { return fakeSession(mode) }`. `fakeSession` lives in `session_start_test.go`; in the mode the working-tree test names, it runs the command the planted `.claude/settings.json` names, as `fakeWall` does. `fakeWall` cannot serve, because its stream reports the approver server.

## The lines

Standard output, one line each, opening `abcd start: `, home paths in their `~` form, every value from the host through `termsafe.Sanitize` and `safeWord`. One `say` writes each line as it is known.

On an account with no profile:

```text
abcd start: this account held no profile, so one was made: ~/.abcd.noindex/profiles/<name>
abcd start: a profile is a separate login and settings on this one account, not a sandbox: a command a session runs can still read whatever the account can
abcd start: abcd wrote its own settings there (settings.json, and an empty skills folder closed to writes) and nothing of yours
abcd start: abcd never performs the login and never reads, stores or forwards its token; log the host in once, in your own shell (this opens no session):

  CLAUDE_CONFIG_DIR=~/.abcd.noindex/profiles/<name> claude auth login

abcd start: no session was started; run abcd start again once the login is done (abcd drain --drive runs on the same profile)
```

On a profile that is logged in, in this order; a line marked optional is printed only when it has something to say:

```text
abcd start: found the profile ~/.abcd.noindex/profiles/<name> as it was; nothing in it is changed (a separate login and settings on this one account, not a sandbox)
abcd start: the host reports a login there; abcd asked for that one state and kept nothing else
abcd start: tools: Bash, Read, Edit, Write, Agent, TaskStop, AskUserQuestion, Monitor, ScheduleWakeup, CronCreate, CronList, CronDelete; no others; no server and no server's tool; no browser tool; no plugin but the host's own
abcd start: hooks: abcd <version>, a copy kept at ~/.abcd.noindex/session-bin/<digest>/abcd, so an update, a rebuild or a removed worktree changes nothing this session runs
abcd start: conventions: AGENTS.md, <n> bytes, handed over whole as a copy only this account can read, kept at <path>, as it stands at this moment
abcd start: permission mode: the host's default (name one with --permission-mode)
abcd start: left out of the session's environment: <NAMES>                                   (optional)
abcd start: this binary has no hook for a sub-agent's start, so the session's settings leave that event out   (optional)
abcd start: one short launch with the session's arguments is starting; it spends a little of the subscription and is written to the launch ledger
abcd start: the launch reported: tools <...>; named and not reported: <...>; no server; no key in use; the guard refused the command it is meant to refuse; plugins, the host's own: <...>; built-in skills (<n>): <...>; built-in agents (<n>): <...>; host <version>
abcd start: that was read from one launch with the session's arguments (in print mode, with abcd's own prompt and without a permission mode of yours), not from the open session; what the host adds after the start abcd cannot see
abcd start: opening the session; abcd says nothing after this line
```

With a mode named, the sixth line reads `permission mode: <mode>, as you named it`. The logged-out refusal carries the same login command as the first block, on its own line.

## The core file, `internal/core/runner/session_start.go`

Exported, each with a doc comment saying what it is for:

- `const DefaultProfileName = "drive"`.
- `func ProfileNames(home string) ([]string, error)`.
- `func MakeProfile(home, name, bin string) (Profile, error)`: `bin` is the kept copy the settings name.
- `func ProfileLoads(p Profile) string`: the first entry of `wallLoaded` that is present by `Lstat` and is not an empty directory, or the empty string: the drive's rule in `checkStageFiles`, so an empty `CLAUDE.md` is named as the drive would refuse it. It reads only.
- `func LoginCommand(p Profile) string`.
- `func SessionHarness(p Profile, repoRoot string) (string, error)`: `defaultLauncher().admit(Claude, "claude", append(p.harnessGuards(), repoRoot)...)`. Item 8 calls it; its answer is the `harness` `ProveSession` runs, the path `startExec` execs, and the file step 2's re-read hashes. `AdmitClaude` discards the path, so it cannot serve.
- `func KeepSessionBinary(home, self string) (kept string, err error)`: the content's SHA-256 names the folder; a copy already there with that content is reused and not rewritten; otherwise the folder is made through `fsutil.EnsureHomeDeclarationScope(home, abcdhome.Rel("session-bin", digest), 0o700)` and the file written beside its final name and renamed into place at `0o500`, then read back by hash.
- `func SessionTools() []string`, `func SessionPermissionModes() []string`: copies of the two lists.
- `type SessionHook struct { Event, Matcher string; Verbs []string }` and `func SessionHooks() []SessionHook`.
- `type Session struct { Profile Profile; Bin string; Hooks []SessionHook; ConventionsName string; Conventions []byte; Mode, Prompt string }`, with `Settings() []byte`, `Args() []string` and `Env(parent []string) (env, cleared []string)`.
- `type SessionProof struct { Reported, Unreported, Plugins, Skills, Agents []string; HarnessVersion string }`.
- `func ProveSession(ctx context.Context, s Session, harness, repo string, ledger *Ledger) (SessionProof, error)`, and unexported `judgeSessionProof(out []byte) (SessionProof, error)` and `proofArgs(s Session) []string`, the short launch's arguments.

It reuses from the package, unchanged: `wallSettings`, `hookCommand`, `hookGroup`, `wallAttribution`, `wallLoaded`, `walledDirRule`, `toolRe`, `credentialVars`, `sealedEnv`, `guardProofCommand`, `guardRefusalText`, `workingTreeSettings`, `wallProofRepository`, `wallPlugin`, `wallServer`, `WallError` and its wall names, `NoVerdictError`, `LimitError`, `claudeRateLimit`, `claudeResultFailed`, `claudeSeen`, `withHarnessVersion`, `fileSum`, `RecordLaunch`, the launcher's `admit` and `run`, `harnessGuards`, `wallProofTimeout`. It adds no dependency.

## The verb file, `internal/surface/cli/start.go`

`runStart` walks the order above and nothing else. Its seams, variables the tests replace: `connectIsTerminal` (already one), `startSelf` for `runner.SelfBinary`, and

```go
var startExec = func(path string, argv, env []string) error {
	// #nosec G204 -- path is the host's file as runner.SessionHarness admitted it.
	return syscall.Exec(path, argv, env)
}
```

with `argv[0]` the host's resolved path. The verb imports `runner`, `ahoy`, `loop` (for `ConventionsFile`), `gitutil`, `layered`, `fsutil`, `abcdhome` and `termsafe`, and holds no rule of its own that the core file could hold: the lists, the arguments and the judge are the core's, the order and the words are the surface's.

`off_profile.go` changes in its three texts: the sentence saying no abcd command starts a session on a profile is replaced, in `offProfileHelp` and in `offProfileLine`, by one saying that `abcd start`, typed by the person in a plain terminal, starts a session on a profile; the comment above `offProfileLine` follows. The warning still carries no path, no variable and no step, and still names the drive as the unattended start.

## The lists that pin a verb

Registering the verb turns each of these red until its list is brought up to date; each is a test watched failing first. The plugin page cannot be put off: the verb is refused by the first of these with no `commands/start.md` and no scoping note.

| List | File | Held by |
|---|---|---|
| The plugin page | `commands/start.md` (new): frontmatter `name`, `description` (the sentence), `argument-hint`, `block: people`; the generated ask-the-binary and command-facts blocks; the binary-resolution paragraph every page carries; and the body below | `TestPluginSurfaceReachesEveryBinaryVerb`, `TestCommandPagesDeclareTheirBlock`, `TestEveryCommandPageOpensWithTheAskTheBinaryCall` |
| The help group | `helpgroups.go` (`"start": {group: groupSetUp}`), and the expected list in `helpgroups_test.go`: ahoy, completion, dashboard, help, session, start, sweep, update | `TestEveryVisibleVerbHasAGroup`, `TestRootHelpListsThePersonsGroups`, `TestRootHelpAgentRendersBothBlocks`, `TestSurfaceSnapshotRecordsHelpPlacement` |
| The one-line sentence | `internal/core/surface/sentences.go`, key `abcd start`; a sentence within the cap of 160, for example "Start a session on abcd alone, making its separate login where none exists: Writes that profile once, a copy of abcd, a ledger entry; refuses an agent's shell." | `TestEveryVisibleVerbCarriesItsSentenceEverywhere`, `TestTheFrameworkCommandsAreTheOnlyExemption`, `TestSentencesAreCheckedByTheDocsLint`, `TestSentencePagesReproduceTheCommittedPages` |
| The worked example | `internal/core/surface/examples.go`, `"abcd start": "abcd start"` | `TestEveryVisibleCommandCarriesAWorkedExample` |
| The bare-render exceptions | `barerender.go` and the surfaces README's `## Bare invocation` section, as "The command" words them | `TestEveryTopLevelVerbRendersStateBareOrIsAnException`, `TestTheBriefEnumeratesEveryBareRenderException` |
| The closed-set flags | `facts_test.go`, `closedSets()` | `TestEnumFlagAnnotationsAreTheReadersSlices` |
| The DOGFOODING recall | `.abcd/rules.json`, `start` between `spec` and `status` | `TestDogfoodingRecallNamesEveryTopLevelVerb` |
| The release-gate rosters | `.abcd/development/release-gate/manifest.json`: `start` in `commands`, the new chapter in `briefDocs` | `TestReleaseGateManifestIsCurrent` |
| The brief's register | `.abcd/development/brief/04-surfaces/README.md`: a row `43`, `/abcd:start`, `shipped`, one line, the chapter link; the group table's row `start`, people, set-up. The `hook` row is not touched: the parallel spec rewrites that one long line | record-lint's `surface_coverage`; the generated index beneath it |
| The surface chapter | `.abcd/development/brief/04-surfaces/43-start.md` (new), in the shape of `40-sweep.md`: what it is, sub-verbs (none), the two outcomes, the order of refusals, what the session holds, what the short launch reads and does not, where it is named, and the appendix markers the generator fills | `TestSurfaceAppendixFromCobraTree`, `TestReleaseGateManifestIsCurrent` |
| The refusal registry | `internal/core/refusal/refusal.go` and the probes in `refusal_raisers_test.go` | `TestEveryEntryIsComplete`, `TestEveryNextCommandAndRaiserIsACommandTheTreeHas`, `TestExitErrorIsBuiltOnlyThroughItsConstructors`, `TestRaisedByIsWhatTheVerbsRaise` |

`surfaceparity_test.go` is not edited: `start` has a page, so it takes no entry in `cliOnlyVerbs`. The page tells an agent that the command is the person's to type and refuses in an agent's shell, and never to run it for them. It gives the line to type, what the first run makes and prints, that the login is the person's, and what the session holds and what it does not. It says that a profile is a separate login and settings on this one account, never a sandbox: a command the session runs can read whatever the account can, the profile directory included. It says the session's hooks are "every event the plugin's manifest wires", with no count, since the parallel spec changes the number; the chapter does the same. It says in one sentence that the short launch runs in an empty repository and does not read a checkout's own `.claude` folder. It says that in a checkout listed in `dogfood-roots` the session's hooks are the build that typed `abcd start`. And it says the kept copies live in `~/.abcd.noindex/session-bin/` and that one may be cleared when no session started from it is open and no profile's `settings.json` names it: a by-hand session on such a profile would otherwise run with every hook missing.

The other pages that name the profile or the missing command:

- `commands/drain.md`: the sentence saying the person makes the directory says `abcd start` makes it, or the person by hand, and the person logs in; the sentence "No abcd command starts one" names `abcd start` and its page, and says its settings are handed over at the start so the directory's files are not written. The by-hand block for two profiles stays.
- `commands/mode.md` and `.abcd/development/brief/04-surfaces/26-mode.md`: the same sentence, reworded as `off_profile.go` words it. In `commands/mode.md` the sentence wraps across two lines; both go.
- `.abcd/development/brief/04-surfaces/35-drain.md`: the sentence on what stays a draft becomes "Choosing by itself among several, a registry and key profiles are itd-2611040344170206, which stays a draft; `abcd start` makes the first profile on an account that holds none", closed by a link, in brackets, to the new chapter `43-start.md` beside it. The link is written there and not here: from this file it would not resolve.
- `.abcd/development/brief/04-surfaces/01-ahoy.md` and `.abcd/development/brief/05-internals/03-configuration.md`: the `profiles/` line says `abcd start` makes the first directory and the person logs in; a new line names `session-bin/`, the copies of abcd an open session's hooks run.

## Generated files, and the command that writes each

Run in this order, from the checkout root, after the hand edits; each is then run a second time to show it changes nothing. None of these files is edited or merged by hand.

1. `make refusal-codes-sync` writes `.abcd/development/release/refusal-codes.json` from the registry.
2. `go generate ./internal/surface/cli` writes `docs/reference/cli/commands.md` (the new verb, and the reworded off-profile help in the four verbs that carry it), `.abcd/development/release/surface.json`, the `description:` line of `commands/start.md` from the sentence, the generated index in the brief's surface README, and the appendix of `43-start.md`.
3. `make command-facts-sync` writes `docs/reference/cli/commands/start.json`, the root's `docs/reference/cli/commands/abcd.json` (its list of verbs), and the generated blocks of `commands/start.md`.

Held by `TestReferenceMatchesCommittedPage`, `TestSurfaceSnapshotMatchesCommittedBaseline`, `TestCommandFactsFilesMatchTheCommittedCopies`, `TestCommandPageBlocksMatchTheCommittedPages`, `TestExplainJSONIsTheCommittedFileSliceForEveryCommand` and `TestCommandFactsCoverTheSnapshotTree`.

## Criteria and the tests that hold them

The criteria are the intent's, numbered in its order. `R` is `internal/core/runner/session_start_test.go`, `S` is `internal/surface/cli/start_test.go`.

| Criterion | Tests |
|---|---|
| 1. No profile: one is made, abcd's settings only, the login command printed, nothing started | R `TestMakeProfileWritesAbcdsSettingsAndNothingElse` (mode `0o700`, the settings equal `wallSettings` for the copy and `renderedByAbcd` holds, the `skills` folder empty at `0o500`, no other entry, a second call refused); R `TestDefaultProfileNameIsDrive`; S `TestStartWithNoProfileMakesOneAndPrintsTheLoginCommand` (exit 0, the login line, the stand-in host never run, the exec seam never called); S `TestStartMakesTheProfileUnderTheNameGiven` |
| 2. A profile already there: no file changed, what was found is said | S `TestStartOnAProfileAlreadyThereChangesNoFileInIt` (names, modes, sizes, hashes and modification times of the whole directory equal before and after a logged-out refusal and after a full start; the refusal's code is read in the one and the exec seam is seen called in the other, so an empty body fails) |
| 3. Not logged in: refused, with the login command | S `TestStartRefusesALoggedOutProfileWithTheLoginCommand` (`start/not-logged-in`; the stand-in saw `auth status` and no print launch) |
| 4. Several profiles | S `TestStartWithSeveralProfilesRefusesAndListsThem`, S `TestStartWithProfileStartsOnTheOneNamed`, S `TestStartRefusesANameTheAccountDoesNotHold` (nothing made) |
| 5. What the session is started with | R `TestSessionArgsAreTheSettledOnes`, R `TestSessionEnvKeepsThePersonsHomeAndNoHostOrStageVariable` (the web-connector switch set among the five), R `TestSessionSettingsAreInlineAndSwitchOffTheFourNamedPlugins`, R `TestSessionAndProofSettingsBothDenyEveryServersTool` (the settings object in `Args()` and the one in `proofArgs` each hold the deny rule); S `TestStartExecsTheHostWithTheSessionsArguments` (the seam's path, arguments and environment; with the running-binary seam answering a path under the checkout's `.abcd/.work.local/bin`, every hook command names a path under the home's `session-bin`; and a variable set to a plain word, never a key-shaped literal, is named on the left-out line and absent from the environment), S `TestStartHandsTheConventionsFileOverWhole` (byte for byte inside the one argument), S `TestStartWritesNoFileADriveReads` (the profile directory and the stage-homes folder unchanged, and the exec seam seen called), S `TestStartRefusesOutsideAManagedCheckout`, S `TestStartRefusesAConventionsFileOverTheBound`, S `TestStartRefusesAConventionsFileThatIsALink` |
| 6. The hook set against the manifest | `TestSessionHooksWireEveryManifestEventToTheSameVerb` in `start_hooks_parity_test.go`; S `TestStartLeavesOutAHookVerbTheBinaryLacks` (given a tree without the verb, the settings leave the event out and the line says so; given the whole tree, every verb in the settings resolves) |
| 7. The short launch | R `TestJudgeSessionProof`, a table: a tool off the list, `Task` for `Agent`, a listed tool unreported, a server, a plugin not the host's own, an `abcd:` entry, a key, no key source, no guard refusal, no init line; R `TestProveSessionRunsTheSessionsArgumentsAndWritesTheLedger` (the stand-in's arguments are the session's with the print flags, one `session-proof` entry); R `TestProveSessionRefusesWhenAWorkingTreeFileWasRead`; S `TestStartRefusesNamingWhatTheShortLaunchReports` (`start/wall-not-held`, the exec seam never called), S `TestStartSaysANamedToolTheLaunchDidNotReport` (said, and the session opens), S `TestStartSaysItReadALaunchNotTheOpenSession`; and, live, `TestTheShortLaunchHoldsOnALoggedInProfile` |
| 8. An agent's shell, or no terminal | S `TestStartRefusesFromAnAgentsShell` (each of the three markers; no profile made), S `TestStartRefusesUnlessAllThreeStreamsAreATerminal` (each stream in turn), S `TestStartChecksThoseTwoThingsAndNoMore` (with `CI`, a dumb `TERM` and an SSH variable set, and neither refusal's cause, it goes on) |
| 9. The prompt, in place of abcd | S `TestStartOpensWithThePersonsPromptInPlaceOfAbcd` (`--` then the prompt, last), S `TestStartWithNoPromptOpensWaiting` (no `--`), S `TestStartSaysNothingAfterItsStartLine` (the opening line is the last byte written before the seam is called), S `TestStartSaysWhenTheHostCannotBeStarted` (a seam that returns an error: exit 1 and the one line), S `TestTheLedgerHoldsTheShortLaunchAndNotTheSession` |
| 10. The permission mode | S `TestStartPassesThePermissionModeNamedAndNoneOtherwise`, S `TestStartRefusesAModeOutsideTheSet` (`usage/invalid-flag-value`, the value not repeated) |
| 11. A repository file starts nothing and adds no tool | S `TestARepositoryFileChangesNothingStartPasses` (a checkout holding `.claude/settings.json` with a default mode and allow rules, a `.mcp.json`, and a start block in `.abcd/config.json`: the exec seam is seen called in both runs, and the arguments and environment it receives are identical to those of the same start without them) |
| 12. The drive runs on the same profile | R `TestADriveAcceptsTheProfileMakeProfileMade` (`DriveProfile` returns it, `WriteWallSettings` reports nothing replaced, `SetAsideSyncedSkills` moves nothing, the drive's check of what a launch would load passes; and `ProfileLoads` agrees with that check on an empty `CLAUDE.md`, which both name, and on an empty folder, which neither does) |
| 13. The warning and the no-profile refusal name the command | `TestNoWalledDirectoryRefusesNamingTheOneToMake` in `profile_test.go`, changed; the two assertions in `off_profile_test.go` that pin the old sentence, changed to the new one; `TestNoLineSaysAbcdHasNoCommandThatStartsASession` in `off_profile_test.go`, new: no `No abcd command starts`, after folding every run of whitespace to one space, in the off-profile texts, `commands/drain.md`, `commands/mode.md`, the reference or `26-mode.md` |
| 14. A separate login and settings, never a sandbox | S `TestEveryLineNamingTheProfileSaysASeparateLoginNeverASandbox`: the verb's own lines, `commands/start.md` and `43-start.md` each hold `separate login and settings`; in those, in `commands/drain.md`, `commands/mode.md`, `35-drain.md`, `26-mode.md` and the reference's `abcd start` section, every paragraph holding both `profile` and `sandbox` has `sandbox` only straight after `not a ` or `never a `. The reference as a whole cannot be held to it: the guard's help there uses the word of other things |
| The copy of abcd (decision 8) | R `TestKeepSessionBinaryIsAddressedByContent`; the exec test's assertion above; in step 2, S `TestStartRefusesWhenTheCopyOrTheHostChangedSinceTheLaunch` |

## The tests, in the order they are written

Step 1, new or changed, in the order of the work:

1. R `TestDefaultProfileNameIsDrive`
2. R `TestMakeProfileWritesAbcdsSettingsAndNothingElse`
3. R `TestADriveAcceptsTheProfileMakeProfileMade`
4. R `TestKeepSessionBinaryIsAddressedByContent`
5. R `TestSessionArgsAreTheSettledOnes`
6. R `TestSessionEnvKeepsThePersonsHomeAndNoHostOrStageVariable`
7. R `TestSessionSettingsAreInlineAndSwitchOffTheFourNamedPlugins`
8. R `TestSessionAndProofSettingsBothDenyEveryServersTool`
9. R `TestJudgeSessionProof`
10. R `TestProveSessionRunsTheSessionsArgumentsAndWritesTheLedger`
11. R `TestProveSessionRefusesWhenAWorkingTreeFileWasRead`
12. `TestNoWalledDirectoryRefusesNamingTheOneToMake` in `profile_test.go`, changed
13. S `TestStartRefusesFromAnAgentsShell`
14. S `TestStartRefusesUnlessAllThreeStreamsAreATerminal`
15. S `TestStartChecksThoseTwoThingsAndNoMore`
16. S `TestStartRefusesAModeOutsideTheSet`
17. S `TestStartWithNoProfileMakesOneAndPrintsTheLoginCommand`
18. S `TestStartMakesTheProfileUnderTheNameGiven`
19. S `TestStartWithSeveralProfilesRefusesAndListsThem`
20. S `TestStartRefusesANameTheAccountDoesNotHold`
21. S `TestStartRefusesOutsideAManagedCheckout`
22. S `TestStartRefusesAConventionsFileOverTheBound`
23. S `TestStartRefusesAConventionsFileThatIsALink`
24. S `TestStartRefusesALoggedOutProfileWithTheLoginCommand`
25. S `TestStartRefusesNamingWhatTheShortLaunchReports`
26. S `TestStartExecsTheHostWithTheSessionsArguments`
27. S `TestStartWithProfileStartsOnTheOneNamed`
28. S `TestStartHandsTheConventionsFileOverWhole`
29. S `TestStartPassesThePermissionModeNamedAndNoneOtherwise`
30. S `TestStartOpensWithThePersonsPromptInPlaceOfAbcd`
31. S `TestStartWithNoPromptOpensWaiting`
32. S `TestStartSaysNothingAfterItsStartLine`
33. S `TestStartSaysWhenTheHostCannotBeStarted`
34. S `TestStartSaysANamedToolTheLaunchDidNotReport`
35. S `TestStartSaysItReadALaunchNotTheOpenSession`
36. S `TestTheLedgerHoldsTheShortLaunchAndNotTheSession`
37. S `TestStartOnAProfileAlreadyThereChangesNoFileInIt`
38. S `TestStartWritesNoFileADriveReads`
39. S `TestARepositoryFileChangesNothingStartPasses`
40. S `TestStartLeavesOutAHookVerbTheBinaryLacks`
41. `TestSessionHooksWireEveryManifestEventToTheSameVerb` in `start_hooks_parity_test.go`
42. `TestTheShortLaunchHoldsOnALoggedInProfile` in `live_session_test.go`, behind the `live` tag, run once by the writing session
43. the two changed assertions and `TestNoLineSaysAbcdHasNoCommandThatStartsASession` in `off_profile_test.go`
44. the seven start probes under `TestRaisedByIsWhatTheVerbsRaise` in `refusal_raisers_test.go`
45. S `TestEveryLineNamingTheProfileSaysASeparateLoginNeverASandbox`, last, since it reads the pages
46. the standing tests of "The lists that pin a verb" and of "Generated files", each red on registration and green once its list or file is current

Step 2: the live proof's named check `a-started-session-names-no-mcp-tool` (the open session's own tool list, as the host lists it to the person and as the session answers when asked, holds no tool beginning `mcp__`; written into the receipt with the list itself); S `TestStartRefusesWhenTheCopyOrTheHostChangedSinceTheLaunch`; R `TestJudgeSessionProofReadsTheLiveLaunchsFirstLine`, the judge over the fixture saved from the live launch.

## What stays as it is

- `internal/surface/cli/drain_drive.go`: not edited. `internal/core/runner/wall.go`: not edited in step 1; step 2 changes one string of it, the drive's logged-out line, once the login command has been seen to work.
- `hooks/hooks.json`: read by the parity test, not edited. spc-2611251757228271 edits it.
- `internal/surface/cli/surfaceparity_test.go`: read, not edited.
- The drive still refuses a credential variable in its environment and still requires `runner.claude`; neither rule is `abcd start`'s.

## Beside the work in flight

spc-2611251757228271 is built at the same time. Its manifest entry for a sub-agent's start is one group, one command, no matcher, which is what the table above holds. Six paths are in both footprints:

- `docs/reference/cli/commands.md` and `.abcd/development/release/surface.json` are generated: never merged by hand; whichever lands second runs the three generating commands again on the merged tree and commits what they write.
- `.abcd/development/brief/04-surfaces/README.md`: that spec rewrites the `hook` row, one line of about 2,500 characters. This one adds its own row, its group row and one sentence under `## Bare invocation`, and does not touch that row.
- `internal/surface/cli/cli.go` (this spec registers the verb; that one adds a row to `applyHookPlaneFailOpen`), `.abcd/development/brief/04-surfaces/01-ahoy.md` and `.abcd/development/brief/05-internals/03-configuration.md` merge as text; that spec's rule on count words covers passages far from the lines this one edits.

When that spec lands, the line saying this binary has no hook for a sub-agent's start stops printing, and `TestStartLeavesOutAHookVerbTheBinaryLacks` keeps testing because it is handed a tree without the verb. No prose of this spec's pages counts the manifest's events. Started through `abcd build`, the second of the two would be refused at its reservation while the first holds those paths. Nothing here depends on the order the two land in. The later steps of spc-2611211659223744 add a ratchet list of command extents that a new command would have to enter if those steps land first; the test that holds it names the entry.

## What the short launch cannot see

abcd says each of these and closes none.

- A print launch with no settings source reports only the four plugins the fixture holds and no server (tested by hand on 2026-11-25, host 2.1.332). So the short launch cannot see what an interactive session alone loads.
- A plugin the host adds later can still draw on the person's screen: the code switches off the four it names and no other. A tool such a plugin registers reaches the session as a server's tool, which the deny rule takes out; `--tools` bounds the host's own. That is tested in print mode only, so the person's live proof (step 2) reads the interactive session's own tool list.
- A checkout's own `.claude` folder. The launch runs in an empty repository; the page says so in one sentence.
- What the host adds after the start. The start's last line but one says so.

## What only the live proof can show

Step 2 reads each of these from the open session or from the host, and writes what it saw into its receipt. None can be shown by a stand-in. The writing session's live run of the short launch in step 1 shows the second and third where a profile is logged in by then; the rest need the open session.

- The login command succeeds on a profile whose `skills` folder is closed to writes, and opens no session.
- A print launch in `dontAsk` with inline settings runs the `PreToolUse` hook and reports its response; the hand test of 2026-11-25 showed a `SessionStart` hook only.
- With the four built-in plugins switched off and the deny rule set, the host still starts, still reports its built-in skills and agents, and reports no server and no tool beyond the list.
- The open session's own tool list names no tool beginning `mcp__` (the named check `a-started-session-names-no-mcp-tool`), and holds `AskUserQuestion`, which no print launch reports, and the sub-agent tool.
- The session answers the conventions file's check word without reading a file, so the kept copy arrived as its instructions, once.
- The person's own instructions file is not loaded, though the checkout sits under the home folder (the hand run's notes of 2026-11-25 record that it loaded as a project file with user settings off; the start relies on `CLAUDE_CODE_DISABLE_CLAUDE_MDS=1`). The session's own listings show no memory file, no server, no plugin but the host's own.
- Whether the host loads a checkout's own `.claude` skills, agents or command pages with no settings source.
- The session's hooks run for a sub-agent's start once spc-2611251757228271 has landed.
- A named permission mode is the mode the session opens in; the host calls its default `manual` at 2.1.332, where an earlier fixture's first line says `default`.
- A drive's start on the same profile passes its own walls with no further step, as far as its consent screen.
- The ledger shows one `session-proof` entry and none for the session.
- Nothing a session leaves in the profile stops a later drive. The host's own menus can write the profile's `settings.json`; a drive rewrites that file at its start and stops at anything a launch would load.
- What the session held, read from its transcript afterwards: the Mechanism's falsifier.
