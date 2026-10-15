---
id: spc-2610120941441282
slug: a-person-hands-an-item-from-their
intent: itd-2610120738075547
origin: researcher-authored
production_mode: hand-written
---
# a-person-hands-an-item-from-their

## Summary

This spec delivers the first rung of
[itd-2610120738075547](../../intents/planned/itd-2610120738075547-a-person-hands-an-item-from-their.md):
handing one item that is ready to build to a hosted cloud session, and getting
its work and its report back, as **scripts** under
[`script-first-mvp`](../../principles/script-first-mvp.md), on abcd's own
repository. The Go verbs (`dispatch`, `collect`) are a later rung that absorbs
the contract these scripts prove. Its trust rule is
[adr-2610120937435856](../../decisions/adrs/2610120937435856-a-cloud-session-s-unchecked-work-goes.md)
and the brief's invariant 23.

Planned on 2026-10-12 by the technical facilitator on the product thinker's
delegation ("decide the deferred item so that you can plan the intent. I want
it ready to be build in a few days (while I'm away)"). A build session builds
it without the person as far as the steps marked **build-alone** reach; the
steps marked **needs the person** are acts only the person can perform (a
hosted session in their claude.ai account, a forge setting only an admin can
change, a public issue).

What a person does once it is built:

1. `scripts/cloud/send.sh <item-id>` on their own computer: abcd checks the
   item is ready and public, composes the brief and starts one hosted session
   through the host connector, and prints the session's id and link.
2. The session runs the setup (a non-root user, gitleaks, the toolchain, the
   committed hooks), checks its machine, reserves the item, works, and hands
   back: its commits and a report file pushed to
   `claude/abcd-wip/<item-id>`, or, if the push fails, the report as a comment
   on the tracking issue.
3. `scripts/cloud/list.sh` shows each session's item, branch, state and report.
4. `scripts/cloud/collect.sh <item-id>` re-runs preflight on a copy on the
   person's computer and, if it passes, offers a landing branch with a local
   receipt; nothing is pushed for them.

| Criterion (intent order) | Met by |
| --- | --- |
| C1, the machine is prepared, nothing runs as root | Steps 1 (the hypothesis), 4, 12 (proved live) |
| C2, the machine is checked before work, stops only for what the final checks need | Steps 2 (the gate-tool list), 3 |
| C3, a brief started in the wrong place stops | Step 3 |
| C4, one brief per item | Step 5 |
| C5, no quiet fallback, an expired login stops the run | Step 9 |
| C6, every session reports, on its branch | Step 8 |
| C7, finished work is not trapped and never reaches the default branch | Steps 6 (the hook entry), 7 (the required check and ruleset) |
| C8, one list of cloud sessions | Step 10 |
| C9, handed-back work lands through the person's own checks | Step 11 |
| C10, abcd starts the session | Steps 9, 12 (proved live) |
| C11, the cloud session runs the full abcd toolchain | Steps 4, 12 (proved live) |
| C12, everything host-specific sits behind the connector | Step 2 (the search test), held by every later step |
| C13, nothing private goes to the cloud | Step 5 |

## Footprint

- packages: scripts/cloud (new), scripts/cloud/connectors/claude (new, the host connector), .claude/settings.json (new, the connector's file at the host's fixed path), scripts/gate-tools.tsv (new), scripts/check-wip-source.sh and scripts/check-wip-source-cases.sh (new), .githooks/pre-push, .github/workflows/ci.yml, .abcd/work/rulesets/main-protection.json, internal/core/lint (static tests), internal/surface/cli (executed-script fixture tests)
- tests: per step below. Every script is exercised in a throwaway clone with a bare repository as its remote, as `internal/surface/cli/githook_prepush_test.go` does for the hook, and every host or forge call against a fake `claude` or `gh` on PATH that records its argv and serves fixture output. No test needs the network, a hosted session, root or gitleaks.

## Design

### Layout and the host connector

```text
scripts/cloud/
  cloud.conf                  # host-agnostic: tracking issue number, report dir
  connector.sh                # loads and validates connector declarations
  begin.sh                    # the brief's first step: place check, env check
  env-check.sh                # C2
  brief.sh, brief.md.tmpl     # C4, C13
  send.sh                     # C10, C5
  hand-back.sh                # C6, the session's last step
  list.sh                     # C8
  collect.sh                  # C9
  connectors/claude/          # everything that names the first host
    connector.conf            # wip_prefix=claude/abcd-wip/ ; host markers
    is-remote.sh              # exit 0 only when CLAUDE_CODE_REMOTE=true
    setup.sh                  # C1, run by the session-start hook
    launch.sh                 # starts a hosted session, maps its failures
.claude/settings.json         # the connector's, at the host's fixed path
```

A connector is a directory under `scripts/cloud/connectors/` holding a
`connector.conf` of `key=value` lines, read (never sourced) by
`connector.sh`. Keys: `name`, `wip_prefix`, `is_remote` (a script path inside
the connector directory), `setup`, `launch`. `connector.sh` refuses a
declaration whose `wip_prefix` does not match `^([a-z0-9][a-z0-9-]*/)?abcd-wip/$`
(the ADR's decision 2: it ends in the segment `abcd-wip/` and holds at most one
segment before it), an unknown key, a script path that leaves the connector
directory, or two connectors declaring one prefix. The first rung has one
connector; `send.sh` takes `--connector <name>` and defaults to the only one
declared, refusing to guess when there are two.

Everything outside `scripts/cloud/connectors/claude/` and `.claude/settings.json`
is host-agnostic (C12): the hook and the required check match only the
declared prefixes and the segment `abcd-wip`; the brief template takes the
branch name and the remote test from the connector.

### The machine (C1, C11)

`.claude/settings.json` (committed) does three things and no more:

- enables the abcd plugin from this repository's own marketplace (the
  `extraKnownMarketplaces` and `enabledPlugins` keys; the build session
  confirms the exact key names against the host's settings reference before
  writing them, and names the page it read in the commit);
- switches off the host's AI co-author line and session link
  (`attribution.commit`, `attribution.sessionUrl`, same confirmation);
- runs `scripts/cloud/connectors/claude/setup.sh` from a `SessionStart` hook.
  The script's first line exits 0 doing nothing unless `is-remote.sh` passes,
  so on the person's computer the hook does nothing (C11).

The hook calls a script, never abcd, so
[adr-2610050711287503](../../decisions/adrs/2610050711287503-abcd-reaches-the-harness-s-user-settings.md)
("hooks live only in the plugin", which governs the person's own configuration
folder) is not engaged: this file is the project's, committed and reviewed,
and the product thinker placed the setup there (the intent's decision 10). The
plugin cannot carry the setup, because the marketplace serves the last
release.

`setup.sh`, as root, idempotent:

1. Create a normal user `abcd` (no sudo), give it the checkout (`chown -R`),
   and refuse to go on if the user cannot be created; it says so and exits 1
   (the scope condition is then falsified, see step 1).
2. Install gitleaks at the version `.github/workflows/ci.yml` pins
   (`GITLEAKS_VERSION`, read from the file, so there is one pin), checksum
   verified exactly as the CI job does.
3. As `abcd`: resolve the declared toolchain through
   `scripts/pinned-toolchain.sh` (so `go run ./cmd/abcd` works) and set
   `core.hooksPath=.githooks` in the checkout (the probe of 2026-10-15 found it
   unset in a hosted clone, so the committed hooks never ran there).
4. Set the checkout's local author identity from the environment's forge
   token (the intent's decisions 17 and 18): the token's login as
   `user.name` and `<id>+<login>@users.noreply.github.com` as `user.email`,
   through the one token-identity primitive the session announcements use
   (spc-2610150728396006's holder fallback, reached as a `go run ./cmd/abcd`
   verb), never a second lookup in the script. When the lookup fails it sets
   nothing and prints one line saying the identity could not be taken from the
   token; no output, commit, ref or trailer ever repeats the identity the
   container came with.
5. Print one line per prerequisite put in place. Every command after setup runs as `abcd`
(`su abcd -c`), and `begin.sh` and `env-check.sh` refuse to run as uid 0.
Preflight runs in the background as `abcd`, so the host's per-command time
limit does not cut it short.

**Decision 15 of the intent, in the design:** the plugin the settings file
enables supplies the hooks and command pages of the last release; every abcd
verb in a cloud session on this repository runs as `go run ./cmd/abcd` from the
checkout, which AGENTS.md already requires of a source checkout, and the brief
says so in its rules. A managed repository (a later rung) uses the released
binary.

### The environment check (C2)

`scripts/gate-tools.tsv` lists, per line, a tool, the preflight gates that need
it (comma-separated, by their Makefile target names) and the setup step that
supplies it; a tool no preflight gate needs carries `-` for its gates. It lives
beside the gates' own scripts; a test holds it to the Makefile's `preflight:`
prerequisites (every gate named is a real target), so it cannot drift silently
(the cost decision 5 accepted, reduced).

`env-check.sh` reads it, tests each tool with `command -v` (and the toolchain
by `scripts/pinned-toolchain.sh --check` or the equivalent the build session
finds), and prints one line per missing tool naming it, its setup step and the
gates that need it. Exit 1 when a missing tool is needed by any preflight gate
(the session stops and hands back a report saying why, C6); exit 0 with a
warning line otherwise. It writes nothing anywhere (a test compares
`git status --porcelain` and the tree's mtimes before and after).

### The brief (C3, C4, C13)

`brief.sh <item-id>` (run on the person's computer by `send.sh`, usable alone
to preview) refuses, writing nothing, unless:

1. the item is ready: an open issue record under `.abcd/work/issues/open/`, or a
   planned intent for which `go run ./cmd/abcd intent ready <id> --json` exits
   0 (decision 7);
2. the item is public: after `git fetch origin`, its record path exists in the
   remote default branch's tree (`git cat-file -e origin/<default>:<path>`); a
   draft never pushed, a record only on a local branch, and a private record in
   the embargo store all fail this (C13, "not yet public");
3. the item is not under embargo: when the embargo store
   (`~/.abcd.noindex/worktrees/<root-sha>/embargo/`) exists on this machine,
   the composed brief is passed through `go run ./cmd/abcd embargo check
   --file <brief>`, and a store that exists while the verb is unknown or
   cannot run is a refusal, never a pass (C13, "held under embargo"; fail
   closed). With no store, nothing is under embargo on this machine.

The brief, from `brief.md.tmpl`, names: the item and its record path; the
first step, `scripts/cloud/begin.sh`; the rule that every commit is authored
under the identity the setup took from the forge token (no author is read
from the person's computer, decision 17); the rules (no hook is ever
skipped, every abcd verb is `go run ./cmd/abcd`, every commit carries
`Assisted-by:` and no co-author trailer, reserve the item with `abcd reserve`
once it is on the default branch); the branch, the connector's `wip_prefix`
followed by the item id; and the hand-back, `scripts/cloud/hand-back.sh` with
the report fields. Before it is printed or sent, the brief passes `go run
./cmd/abcd lint outbound --label brief`, and a check that it holds no `$HOME`
path and no session link (C4).

`begin.sh` runs the connector's `is_remote` script first: outside the cloud it
prints that it is not running in the environment the brief was written for and
exits 1, before anything else (C3). Inside, it refuses uid 0, stops before any
work when the checkout's author identity is unset or is not a person's (the
attribution gate's own signals, through `scripts/check-attribution.sh`; it
names the cause without repeating the identity), and runs `env-check.sh`.

### The push hook entry (C7, invariant 23)

The reservation build (itd-2609091034175565, branch `feat/reserve-on-remote`)
turns the hook's reservation exception into a declared list of receipt-less
prefixes, each entry naming the prefix, the commit shape a push under it must
have, and the record that rules it. This spec adds one entry and no special
case: its prefixes are those the connector declarations name (read through the
same validation as `connector.sh`; a declaration that fails it makes the hook
refuse the push, naming the file); its shape is (a) the branch is the prefix
followed by exactly one record id (`iss-`, `itd-` or `spc-` followed by sixteen
digits, or an ordinal `itd-`/`iss-` id) and nothing else, and (b)
`scripts/check-attribution.sh commits <base> <local_sha>` passes over the
pushed range (`<remote_sha>..<local_sha>` on an update; from the merge base
with `origin/<default>` on a create); its record is adr-2610120937435856. The
default-branch refusal and the non-fast-forward refusal stay ahead of every
entry. The hook's header comment gains the entry and its reason.

### The required check and the ruleset (C7, invariant 23)

`scripts/check-wip-source.sh <source-ref> <base> <head>` exits 1 when the
source branch holds the path segment `abcd-wip` (anywhere, so it is
host-agnostic) or when the diff `<base>...<head>` adds a path under
`.abcd/work/cloud-reports/`; exit 0 otherwise. Its cases file follows
`scripts/check-attribution-cases.sh`.

`.github/workflows/ci.yml` gains a job `wip-guard` that always runs, on
`pull_request` and `merge_group` (the docs-only classifier never stands it
down): on a pull request it passes `github.head_ref`; on a merge-queue entry it
reads the pull request number from the queue ref
(`gh-readonly-queue/<base>/pr-<N>-<sha>`) and the pull request's head ref from
the REST API with the job's read-only token, and refuses when either cannot be
read. Least-privilege `permissions:`, pinned actions, and `zizmor` clean.

`.abcd/work/rulesets/main-protection.json` adds `wip-guard` to the required
contexts. Applying the ruleset on GitHub is the person's act (step 7), and only
after the job has landed on the default branch, since a required context no
workflow reports blocks every merge.

### The hand-back and the report (C6)

`hand-back.sh --state <handed-back|stopped> --did <text> --failed <text>
--needs <text>` writes `.abcd/work/cloud-reports/<item-id>.md` (a short
frontmatter: `item`, `state`, `channel`, `finished_at`; then the three
sections), runs `go run ./cmd/abcd lint outbound --label report` over it and
refuses to send a report that fails (it prints the finding so the session can
fix its text), commits it alone as the branch's last commit (attribution-clean,
`Assisted-by:`), and pushes `HEAD:refs/heads/<wip_prefix><item-id>` through the
committed hook (never `--no-verify`). When the push fails, it posts the same
report with `channel: issue-comment` and the push's error as a comment on the
tracking issue (`cloud.conf`'s `tracking_issue`) through `gh api
repos/{owner}/{repo}/issues/{n}/comments --input -`, after the same outbound
check. When both fail it exits 1 and prints the report, so the transcript still
holds it as a last resort.

### The list (C8)

`list.sh` fetches `refs/heads/<wip_prefix>*` for every declared prefix and the
reservations (`abcd-reserved/*`, once that build is on the default branch),
reads each branch's report file from its tip with `git show`, and reads the
tracking issue's comments through `gh api` for sessions whose push failed. One
row per item: item, branch, state (`handed-back`, `stopped`, `working` when a
live reservation exists and no report does, `unknown` otherwise), and the
report's first line. It never opens a transcript, and it writes nothing but
the fetched remote-tracking refs.

### The collect (C9)

`collect.sh <item-id>`: fetch the branch; refuse if its last commit is not the
report commit alone; create a worktree in the machine store
(`~/.abcd.noindex/worktrees/<root-sha>/collect-<item-id>/`, per
adr-2610031751065746) on a new branch `collect/<item-id>` from
`origin/<default>`; cherry-pick every commit of the branch except the report
commit; refuse if `scripts/check-attribution.sh commits` fails on them (the
commits must already name the person as author; collect never rewrites
authorship); run `make preflight` there, waiting while another preflight runs
on the machine; on a pass, print the branch, the receipt and the push command
for the person, and push nothing; on a failure, print the failures, keep the
worktree for inspection, and push nothing. Any receipt the cloud session made
is ignored: receipts live in the local tier and never travel, and collect never
reads one from the branch (decision 9).

## Steps

1. The proof rung: the platform's hypotheses on a real hosted session (**needs the person**)
   - criteria: C1 and C10 (the hypotheses); the scope conditions on non-root running and on the proxy accepting the work-in-progress name
   - packages: .abcd/development/research/notes (one dated note holding the receipts)
   - the person, in a hosted session on this repository started from their own terminal with `claude -p "<probe brief>" --environment <env-id> --ref main --output-format json` (proving that the start returns an id and a link), runs as root: `useradd -m abcd`, gives it the checkout, installs gitleaks 8.24.3 checksum-verified, and, as `abcd`, runs `make preflight` on unchanged `main` in the background and records how long it took and what failed; then makes one empty-tree commit under their own author identity with an `Assisted-by:` trailer and pushes it to `claude/abcd-wip/iss-0000000000000000` (a reserved example id), records whether the proxy accepted it, and posts one comment to a scratch issue through `gh api` over REST; also records whether `CLAUDE_CODE_REMOTE` is `true` and whether a commit made with the attribution settings off carries a co-author line or a session link
   - also the person's: create the one tracking issue for cloud sessions (a public act) and give its number to `scripts/cloud/cloud.conf` in step 8's change, or tell the build session
   - outcome: if the proxy refuses the work-in-progress name, steps 2, 6 and 8 stay unlanded and the prefix goes back to the person (the ADR's decision 2 and the intent's decision 3); if the `abcd` user cannot run preflight, step 4 stays unlanded and the runner goes back to the person (decision 8); every other step proceeds
   - tests: none in code; the note's receipts are the evidence, kept free of session links and home paths
2. The host connector's declaration and the gate-tool list (**build-alone**)
   - criteria: C12, C2 (the list)
   - packages: scripts/cloud/connector.sh, scripts/cloud/cloud.conf, scripts/cloud/connectors/claude/connector.conf, scripts/cloud/connectors/claude/is-remote.sh, scripts/gate-tools.tsv, internal/core/lint, internal/surface/cli
   - tests: TestConnectorDeclarationLoads; TestConnectorRefusesBadPrefix (table: `main`, `claude/`, `abcd-wip`, `a/b/abcd-wip/`, empty, a trailing segment after `abcd-wip/`); TestConnectorRefusesUnknownKeyAndEscapingPath; TestConnectorRefusesDuplicatePrefix; TestIsRemoteOnlyWhenMarkerSet; TestGateToolsNamesOnlyRealPreflightGates; TestNoHostMarkerOutsideConnector (searches every file this spec adds or changes outside the connector directory and `.claude/settings.json` for the host's name, its command-line tool, `CLAUDE_CODE_REMOTE` and `claude/abcd-wip`, and finds none)
3. The place check and the environment check (**build-alone**)
   - criteria: C2, C3
   - packages: scripts/cloud/begin.sh, scripts/cloud/env-check.sh, internal/surface/cli
   - tests: TestBeginRefusesOutsideCloud (exit 1, the message, nothing else run); TestBeginRefusesRoot (a fake `id -u` of 0); TestBeginStopsOnVendorOrUnsetIdentity (and the output never contains the identity it refused); TestEnvCheckNamesMissingToolSetupAndGates; TestEnvCheckStopsForAGateTool; TestEnvCheckWarnsForANonGateTool; TestEnvCheckWritesNothing
4. The setup step and the settings file (**build-alone**; proved live in step 12)
   - criteria: C1, C11
   - needs: 2
   - waits on: step 1's outcome on non-root running
   - packages: scripts/cloud/connectors/claude/setup.sh, .claude/settings.json, internal/surface/cli, internal/core/lint
   - waits on: the token-identity verb from spc-2610150728396006's step 2 reaching the default branch, for the identity half only
   - tests: TestSetupDoesNothingOffCloud; TestSetupSetsIdentityFromToken (a fake verb returning login and id; `user.name` and `user.email` set locally); TestSetupLookupFailureSetsNothingAndSaysSoOnce (the container's identity appears in no output, commit, ref or trailer); TestSetupReadsGitleaksPinFromCI (the version it would install equals ci.yml's); TestSetupRefusesWhenUserCannotBeCreated (a fake `useradd` failing); TestSetupIsIdempotent (fakes for `useradd`, `chown`, the download and `su`, run twice); TestSettingsFileShape (plugin enabled from this repository's marketplace, both attribution keys off, one SessionStart hook naming setup.sh, nothing else)
5. The brief (**build-alone**)
   - criteria: C4, C13
   - packages: scripts/cloud/brief.sh, scripts/cloud/brief.md.tmpl, internal/surface/cli
   - tests: TestBriefNamesEveryPart (item, author, first step, the no-skip and go-run rules, branch from the connector's prefix, hand-back and fallback); TestBriefCarriesNoSessionLinkHomePathOrSecret; TestBriefRefusesUnreadyIntent; TestBriefAcceptsOpenIssueAndReadyIntent; TestBriefRefusesItemNotOnRemoteDefault (a draft never pushed; a record only on a local branch); TestBriefRefusesEmbargoedItem (a fixture embargo store and a fake `abcd embargo check` exiting 2); TestBriefFailsClosedWhenEmbargoCheckCannotRun (store present, verb unknown); TestBriefWithoutStoreSkipsEmbargoCheck; TestBriefReadsNoAuthorFromThisComputer
6. The push hook's work-in-progress entry (**build-alone**, after `feat/reserve-on-remote` has landed its declared list on the default branch; until then this step waits)
   - criteria: C7 (the hook half), C12
   - needs: 2
   - waits on: the reservation build's declared list of receipt-less prefixes reaching the default branch (`feat/reserve-on-remote`; landed by pull request #1004, so this step is free); the session-announcement build (spc-2610150728396006) adds a row to the same table, so the two rows are coordinated with that build
   - packages: .githooks/pre-push, internal/surface/cli/githook_prepush_test.go; internal/core/guard/defaults if the guard registry refuses the session's push (add it as a `known_good`, as the reservation step did)
   - tests: TestPrePushAdmitsWIPWithoutReceipt (create and update); TestPrePushRefusesOtherHostBranchWithoutReceipt (`claude/fix-x`); TestPrePushRefusesWIPWithBadRecordID (none, two ids, a trailing path); TestPrePushRefusesWIPFailingAttribution (an AI co-author trailer; no `Assisted-by:`); TestPrePushRefusesOnBadDeclaration; TestPrePushWIPStillRefusesForcePush; TestPrePushDefaultBranchStillRefused; the reservation entry's tests pass unchanged
7. The required check and the ruleset (**build-alone** for the job and the mirror; **needs the person** to apply the ruleset)
   - criteria: C7 (the boundary half)
   - packages: scripts/check-wip-source.sh, scripts/check-wip-source-cases.sh, .github/workflows/ci.yml, .abcd/work/rulesets/main-protection.json, internal/core/lint
   - tests: the cases file (a source holding `abcd-wip` at either depth refused; `abcd-wipe/` and `wip/` passed; a diff adding `.abcd/work/cloud-reports/x.md` refused; a clean diff passed); TestWIPGuardJobRunsOnPullRequestAndMergeGroup; TestWIPGuardNeverStoodDown; TestMainProtectionRequiresWIPGuard; zizmor clean
   - the person's act, after the job is on the default branch: apply the ruleset to GitHub, then open a throwaway pull request from `test/abcd-wip/iss-0000000000000000` and see `wip-guard` refuse it; close it unmerged
8. The hand-back and the report (**build-alone**)
   - criteria: C6
   - needs: 2
   - packages: scripts/cloud/hand-back.sh, scripts/cloud/cloud.conf (the tracking issue number from step 1), internal/surface/cli
   - until the person has created the tracking issue, `cloud.conf` carries `tracking_issue=` empty and the fallback refuses by name ("no tracking issue configured") and prints the report; it never posts anywhere else
   - tests: TestHandBackCommitsReportAloneAsLastCommit; TestHandBackPushesToWIPBranchThroughTheHook (no `--no-verify` in any argv); TestHandBackFallsBackToIssueCommentWhenPushFails (fake `gh` records the REST call and body); TestHandBackRefusesReportFailingOutbound; TestHandBackReportNamesItsChannel; TestHandBackPrintsReportWhenBothFail
9. Starting a session through the connector (**build-alone**; proved live in step 12)
   - criteria: C10, C5
   - needs: 2, 5
   - packages: scripts/cloud/send.sh, scripts/cloud/connectors/claude/launch.sh, internal/surface/cli
   - the environment id is read from the local tier (`.abcd/.work.local/cloud/environment`), never committed; the session's link is printed to the person and never written to a tracked file
   - tests (fake `claude` on PATH): TestSendStartsOneSessionWithTheBrief; TestSendPrintsIdAndLink; TestSendRefusesWhenHostToolMissing; TestSendRefusesWithoutLogin; TestSendExpiredLoginSaysRenewByHand; TestSendRefusesUnknownEnvironment; TestSendHostRefusalStartsNothingLocal (no worktree, no process, no file beyond the brief preview); TestSendRefusesPrivateOrUnreadyItem (through step 5)
10. The list (**build-alone**)
    - criteria: C8
    - needs: 2, 8
    - packages: scripts/cloud/list.sh, internal/surface/cli
    - tests (bare origin fixture, fake `gh`): TestListShowsHandedBackStoppedAndWorking (the working one from a reservation branch); TestListShowsFallbackCommentReport; TestListReadsNoTranscript (no path under any transcript store is opened); TestListWritesOnlyRemoteTrackingRefs
11. The collect (**build-alone**)
    - criteria: C9
    - needs: 8
    - packages: scripts/cloud/collect.sh, internal/surface/cli
    - tests (fake `make`): TestCollectLeavesReportCommitOut; TestCollectRefusesBranchWithoutReportTip; TestCollectRefusesAttributionFailure; TestCollectRunsPreflightInMachineStoreWorktree; TestCollectPassPrintsBranchAndPushesNothing; TestCollectFailurePushesNothingAndKeepsWorktree; TestCollectIgnoresAnyCloudReceipt
12. The live run end to end, and the close (**needs the person**)
    - criteria: C1, C10, C11 proved in a real hosted session; C6, C8, C9 seen working
    - needs: 1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11
    - waits on: the ruleset applied on GitHub (step 7's person act)
    - the person sends one small open issue with `send.sh`, lets the session hand back, runs `list.sh` and `collect.sh`, and lands the collected branch the usual way; a build session can prepare everything and write the note, but the send and the landing are the person's
    - records: a dated research note with the receipts (no session link, no home path); the change that finishes this step closes this spec (`go run ./cmd/abcd spec close spc-2610120941441282`, after the docs review for HEAD), carries `Delivers: itd-2610120738075547`, and resolves iss-2610120729071657, iss-2610120729094755, iss-2610120729087253, iss-2610120729083126 and iss-2610120724251698 with a `Resolves:` trailer each; iss-2610120729076366 keeps its worktree-guidance half and stays open with a note that C5 answered its other half

## Delivery

**Step 1, partly answered on 2026-10-15.** A probe in a real hosted session
found that the forge proxy accepted a push to a branch outside the host's own
namespace as well as inside it, that the forge credited both pushes to the
person's own account through the environment's token, that the clone's git
identity was the host vendor's default (decisions 17 and 18 answer it in
step 4), and that the committed pre-push hook never ran because the clone had
no `core.hooksPath` (step 4 sets it). Still unproved there: a push to
`claude/abcd-wip/<item-id>` itself, running preflight as a non-root user,
`CLAUDE_CODE_REMOTE`, and a start from the person's terminal; step 12 proves
them.

A build session does not wait for step 1 to start: steps 2, 3, 5, 7 (its code
half), 8, 9, 10 and 11 need nothing from the person, and step 1's outcome can
only hold back steps 2, 4, 6 and 8, as step 1 says. Step 6 waits on the
reservation build's declared list reaching the default branch; the reserve
line in the brief and the `working` state in the list wait on `abcd reserve`
reaching it, and work without it until then (the list shows `unknown`). After
step 2, steps 3, 5, 8 and 11 touch disjoint files and can run as parallel
lanes.

Every lane honours the machine's one-preflight-at-a-time rule, and no lane runs
a hosted session or touches the forge's settings.

## Safety properties

Each is held by a named test above.

1. No receipt-less push outside a declared work-in-progress prefix
   (TestPrePushRefusesOtherHostBranchWithoutReceipt,
   TestConnectorRefusesBadPrefix, TestPrePushRefusesOnBadDeclaration).
2. No work-in-progress branch or cloud report reaches the default branch
   (the check-wip-source cases, TestMainProtectionRequiresWIPGuard,
   TestWIPGuardNeverStoodDown).
3. No cloud receipt is trusted (TestCollectIgnoresAnyCloudReceipt,
   TestCollectRunsPreflightInMachineStoreWorktree).
4. Nothing private is handed out (TestBriefRefusesItemNotOnRemoteDefault,
   TestBriefRefusesEmbargoedItem, TestBriefFailsClosedWhenEmbargoCheckCannotRun).
5. No quiet local fallback (TestSendHostRefusalStartsNothingLocal,
   TestBeginRefusesOutsideCloud).
6. No hook is skipped (TestHandBackPushesToWIPBranchThroughTheHook).
7. No check runs as root (TestBeginRefusesRoot; the env check refuses uid 0).
8. Public text carries no session link, tool footer, home path or secret
   (TestBriefCarriesNoSessionLinkHomePathOrSecret,
   TestHandBackRefusesReportFailingOutbound).

## Out of scope

The intent's "Left out of the first rung": the Go verbs, managed repositories,
other hosts and runners, a session cap, a stored login, automatic merging,
messages between running sessions, the wider session register, recurring
routines, deleting spent branches, and the two issue-local halves.
