---
id: itd-2610140858048004
slug: abcd-s-home-keeps-itself-out-of-a
spec_id: spc-2610140858477542
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# abcd's home keeps itself out of a dotfiles repository

## Press Release

> abcd's home keeps itself out of a dotfiles repository. Frank, a DevOps engineer, keeps their whole home folder in a git repository they push to a public remote, and abcd keeps session transcripts, private idea verdicts and run state under ~/.abcd.noindex in that same home. The first time abcd runs, it lays a .gitignore holding * in ~/.abcd.noindex, so Frank's next git add -A skips everything abcd keeps there and nothing private leaves the machine by accident. abcd lays the file once: if Frank deletes it because they want their dotfiles repository to track abcd's home, abcd leaves it deleted. Credentials stay out of every git working tree whatever the file says. "I found out about the .gitignore when I went looking for why my dotfiles diff was clean — which is exactly how I wanted to find out."

## Why This Matters

abcd keeps a person's machine-scoped records in `~/.abcd.noindex`: redacted session transcripts, private ideate verdicts (iss-2610090740139804), run and loop state, labs, lifeboats, interview records and settings backups. For anyone whose home folder is itself a git repository, every one of those files is one `git add -A && git push` from a public remote, and today nothing but the credential store and a new sources corpus refuses to write there. The person did nothing wrong; abcd put the files in the path of their ordinary habit. abcd added the risk, so abcd removes it (the abcd-fixes-what-it-adds principle), and does so inside its own folder, without touching the person's repository or its configuration.

## Mechanism

We expect a `.gitignore` holding `*` inside `~/.abcd.noindex` to keep every file abcd writes out of a dotfiles repository, because git applies a `.gitignore` to the paths beneath its own directory, so the file excludes abcd's untracked files without abcd reading, running or changing anything of the person's repository. It would be shown wrong by a `git add -A` in such a home that stages a file abcd wrote after the file was laid.

## Scope Conditions

- The files under `~/.abcd.noindex` are not already tracked by the home repository: an ignore file never untracks a committed file. <!-- cond: cond-2610140858479946 -->
- The person stages with `git add` and not `git add -f`, which overrides every ignore file. <!-- cond: cond-2610140858479857 -->
- The person's git honours a `.gitignore` in a subdirectory, as every git release abcd supports does. <!-- cond: cond-2610140858471836 -->

## What's In Scope

- Laying `~/.abcd.noindex/.gitignore` holding `*`, owner-only, once per home, from the step every abcd command passes through, the status line excepted: it writes nothing at all (iss-2610050556383525).
- A marker beside it that records the file was laid, so a deleted file is never laid again.
- Homes that predate the change: the first run after it lays the file.
- A private ideate verdict written into the account's own git-tracked home, which the laid file keeps out of that repository.
- The help and docs saying what the file is for, and that deleting it is how a person lets their home repository track abcd's home.

## What's Out of Scope

- Reading, running or changing the person's own repository: its ignore files, its configuration, its hooks. abcd judges no repository to lay the file; it lays it in every home.
- Untracking a file the home repository already tracks.
- Loosening the credential rule: the abcd home store stays refused inside any git working tree, the account's own included, whatever the file says.

## Acceptance Criteria

- Given a home with no `~/.abcd.noindex`, When any abcd command creates it, Then `~/.abcd.noindex/.gitignore` holds `*`, is readable by the account alone, and the marker records that abcd laid it.
- Given a `~/.abcd.noindex` from before this change with no `.gitignore` and no marker, When the next abcd command runs, Then the file is laid once and marked.
- Given abcd laid the file and the person deleted it, When any abcd command runs, Then abcd does not lay it again, and says nothing about it.
- Given a home folder that is a git working tree with the file laid, When the person runs `git add -A` in the home, Then nothing under `~/.abcd.noindex` is staged.
- Given a home folder that is the account's own git working tree with the file laid, When `abcd ideate record --private` runs, Then the verdict is written; and Given a HOME planted inside another checkout, Then the private verdict is refused with nothing written.
- Given any home inside a git working tree, with or without the file, When a credential is stored in the abcd home, Then the store refuses it, as it does today.
- Given the old `~/.abcd` still stands, or `~/.abcd.noindex` is a symlink, When any abcd command runs, Then nothing is laid and nothing is said: the stop writes nothing, and the lay never writes through a link, reading a symlinked home as absent.
- Given the file cannot be laid (a read-only home), When an abcd command runs, Then the command still runs, and one line on stderr says the file was not laid and why, except under `--json` and in a hook call, where a machine reads stderr and nothing is added to it.
- Given a home with no file laid, When the harness runs the status line, Then nothing is written: the status line never lays the file.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-69d8a2a7ab69 -->
Fidelity review — receipt rcp-69d8a2a7ab69 (verifier claude-opus-5-5 2026-10-14).

Provenance: claude-opus-5-5@2026-10-14 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:e8e67adacb06799c46f5e1a9ec02336df68bce41de03c9dada257fb07481fb8d
Input attestations: intent:.abcd/development/intents/shipped/itd-2610140858048004-abcd-s-home-keeps-itself-out-of-a.md@sha256:0fbe58e90fbeb83f66dd58761a356107e0eccd582013d71b5f6d75f7580673b5; spec:.abcd/development/specs/closed/spc-2610140858477542-abcd-s-home-keeps-itself-out-of-a.md@sha256:fc9a1fddb679a09c0b65e0b4e0dacf019e54105bc72bdc3542734d1acb03c67e; diff:b86cf6bbd..15389cf78 (branch fix/ideate-visibility; feature commit f5060d4b8)@-;

Acceptance rollup: MET 8 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: The front door runs a second, deferred lay after the command, so a home the command creates is laid before run returns; Lay writes .gitignore holding `*` and then the marker, both by exclusive create at abcdhome.FileMode (0600). The front-door test shows a fresh home created by `ideate record --private` carries .gitignore; the unit test pins the `*` line and 0600 on both files.
  evidence: internal/surface/cli/cli.go:6609 — "if !isStatusline(args) {"
  evidence: internal/surface/cli/cli.go:6612 — "defer layHomeIgnore(stderr, quiet)"
  evidence: internal/core/homeignore/homeignore.go:95 — "fsutil.CreateExclusiveIn(root, IgnoreName, []byte(ignoreBody), abcdhome.FileMode)"
  evidence: internal/core/homeignore/homeignore.go:105 — "fsutil.CreateExclusiveIn(root, MarkerName, []byte(markerBody), abcdhome.FileMode)"
  evidence: internal/core/homeignore/homeignore_test.go:50 — "if fi.Mode().Perm() != 0o600 {"
  evidence: internal/surface/cli/homeignore_test.go:40 — "func TestFrontDoorLaysAfterTheCommandThatCreatesTheHome(t *testing.T) {"
- ac-2 — MET: A pre-existing ~/.abcd.noindex with neither file is laid by the next command (the before-lay), and the after-lay then finds the marker and writes nothing, so the file is laid once and marked; the front-door test runs `--version` against such a home and finds both files.
  evidence: internal/surface/cli/homeignore_test.go:20 — "func TestFrontDoorLaysTheIgnoreFileInAnExistingHome(t *testing.T) {"
  evidence: internal/surface/cli/homeignore_test.go:30 — "for _, name := range []string{homeignore.IgnoreName, homeignore.MarkerName} {"
  evidence: internal/core/homeignore/homeignore.go:89 — "if _, err := root.Lstat(MarkerName); err == nil {"
- ac-3 — MET: While the marker stands Lay returns AlreadyLaid with a nil error and writes nothing; layHomeIgnore prints only on an error, so a deleted .gitignore is neither relaid nor mentioned. The unit test deletes the file and proves the second Lay is AlreadyLaid with the file still absent.
  evidence: internal/core/homeignore/homeignore.go:90 — "return AlreadyLaid, nil"
  evidence: internal/surface/cli/homestop.go:138 — "if _, err := homeignore.Lay(os.Getenv("HOME")); err != nil {"
  evidence: internal/core/homeignore/homeignore_test.go:71 — "func TestLayNeverLaysAgain(t *testing.T) {"
- ac-4 — MET: The mechanism is run with real git: a home that is a git working tree holds a transcript under ~/.abcd.noindex, Lay runs, `git add -A` stages .zshrc and nothing under abcd's home. The test passed on this machine (go test -count=1).
  evidence: internal/core/homeignore/homeignore_test.go:159 — "func TestGitAddSkipsTheLaidHome(t *testing.T) {"
  evidence: internal/core/homeignore/homeignore_test.go:184 — "git("add", "-A")"
  evidence: internal/core/homeignore/homeignore_test.go:186 — "if strings.Contains(staged, abcdhome.Rel()) {"
- ac-5 — MET_WITH_CONCERNS: validateStore refuses a private verdict only when gitutil.HomeInsideWorkingTree names a tree, which exempts the account-database home; tests prove the account's own tracked home is written and a planted HOME inside a tree is refused with no abcd home created. Concern: the refusal is proven at the core only; through the front door the before-lay is not gated on the working-tree rule, so a planted HOME whose ~/.abcd.noindex already exists gets .gitignore and gitignore-laid written into the other checkout on the same run whose verdict is refused (the intent's 'lays it in every home' scope signs this off, but 'nothing written' then holds for the verdict, not the command).
  evidence: internal/core/ideate/record.go:323 — "if gitutil.HomeInsideWorkingTree(store.Home) != "" {"
  evidence: internal/core/ideate/visibility_test.go:272 — "func TestRecordPrivateWritesIntoTheAccountsOwnTrackedHome(t *testing.T) {"
  evidence: internal/core/ideate/visibility_test.go:159 — "func TestRecordPrivateRefusesAHomeInsideAWorkingTree(t *testing.T) {"
  evidence: internal/surface/cli/cli.go:6611 — "if layHomeIgnore(stderr, quiet) {"
- ac-6 — MET: Set refuses the abcd home whenever any .git entry sits above ~/.abcd.noindex (workingTreeAbove, no account-home exemption); a new test proves the refusal with the ignore file laid and nothing written, and the pre-existing test covers the same refusal without the file.
  evidence: internal/core/credential/store.go:238 — "if c.Home == HomeABCD && workingTreeAbove(home, abcdhome.Rel()) != "" {"
  evidence: internal/core/credential/homeignore_test.go:18 — "func TestTheLaidIgnoreFileDoesNotAdmitACredential(t *testing.T) {"
  evidence: internal/core/credential/store_test.go:202 — "func TestAValueIntoAWorkingTreeIsRefused(t *testing.T) {"
- ac-7 — MET: The old-home stop returns before the lay is reached, so nothing is laid while ~/.abcd stands; a symlinked ~/.abcd.noindex fails the home-scope walk with HomeScopeLinkError, which Lay reads as NoHome with no error, so nothing is written through the link and no stderr line is printed. Unit and front-door tests cover both.
  evidence: internal/surface/cli/cli.go:6599 — "if stop := homeStop(); stop != nil {"
  evidence: internal/core/homeignore/homeignore.go:82 — "if os.IsNotExist(err) || errors.As(err, &link) {"
  evidence: internal/surface/cli/homeignore_test.go:84 — "func TestFrontDoorLaysNothingWhileTheOldHomeStands(t *testing.T) {"
  evidence: internal/surface/cli/homeignore_test.go:94 — "func TestFrontDoorIsSilentForASymlinkedHome(t *testing.T) {"
  evidence: internal/core/homeignore/homeignore_test.go:117 — "func TestLayLeavesASymlinkedHomeAlone(t *testing.T) {"
- ac-8 — MET: A failed lay returns false, so the deferred second lay is never scheduled and exactly one line reaches stderr; the command's exit is unaffected. quiet is set under --json and for `hook <event>` / `guard hook`, and tests prove exit 0 with one line for a read-only abcd home and no line for --json, guard hook and hook prompt-router.
  evidence: internal/surface/cli/homestop.go:140 — "fmt.Fprintf(stderr, "abcd: %s/%s was not laid (%s); the next command tries again\n","
  evidence: internal/surface/cli/cli.go:6610 — "quiet := jsonRequestedIn(args) || isHookCall(args)"
  evidence: internal/surface/cli/homeignore_test.go:74 — "if n := strings.Count(stderr, "was not laid"); n != 1 {"
  evidence: internal/surface/cli/homeignore_test.go:110 — "func TestFrontDoorKeepsAFailedLayOffMachineStderr(t *testing.T) {"
- ac-9 — MET: The front door skips both lays when the first non-flag word is `statusline`; the pre-existing read-only proof runs the status line through run() against a home with ~/.abcd.noindex present and no file laid (history-registration case) and fails on any created path, and it passes with the lay wired in.
  evidence: internal/surface/cli/homestop.go:168 — "func isStatusline(args []string) bool {"
  evidence: internal/surface/cli/statusline_readonly_test.go:103 — "code = run([]string{"statusline"}, strings.NewReader(stdin), &so, &se)"
  evidence: internal/surface/cli/statusline_readonly_test.go:191 — "name: "managed by history registration only","
  evidence: internal/surface/cli/statusline_readonly_test.go:244 — "if d := diffTrees(before, snapshotTrees(t, home, repo)); len(d) > 0 {"

Gap audit:
- honoured:
  - The first time abcd runs it lays a .gitignore holding * in ~/.abcd.noindex, so git add -A skips everything abcd keeps there
    evidence: internal/core/homeignore/homeignore.go:43 — ""*\n""
    evidence: internal/core/homeignore/homeignore_test.go:159 — "func TestGitAddSkipsTheLaidHome(t *testing.T) {"
  - abcd lays the file once: a deleted file stays deleted
    evidence: internal/core/homeignore/homeignore.go:89 — "if _, err := root.Lstat(MarkerName); err == nil {"
    evidence: internal/core/homeignore/homeignore_test.go:71 — "func TestLayNeverLaysAgain(t *testing.T) {"
  - Credentials stay out of every git working tree whatever the file says
    evidence: internal/core/credential/store.go:238 — "if c.Home == HomeABCD && workingTreeAbove(home, abcdhome.Rel()) != "" {"
    evidence: internal/core/credential/homeignore_test.go:18 — "func TestTheLaidIgnoreFileDoesNotAdmitACredential(t *testing.T) {"
  - abcd reads, runs and changes nothing of the person's repository to lay the file, and never overwrites a .gitignore the person keeps
    evidence: internal/core/homeignore/homeignore.go:75 — "root, err := fsutil.OpenHomeScope(home, abcdhome.Rel())"
    evidence: internal/core/homeignore/homeignore_test.go:90 — "func TestLayKeepsAnExistingIgnoreFile(t *testing.T) {"
  - Docs say what the file is for and that deleting it lets the home repository track abcd's home
    evidence: docs/how-to/install.md:205 — "stands abcd never writes `.gitignore` again. To let your home repository track"
- diverged:
  - 'The help and docs saying what the file is for': the docs and the brief say it, and the laid file and marker carry their own explanation, but no CLI --help text or command page names the file
    evidence: internal/core/homeignore/homeignore.go:41 — "const ignoreBody = "# Laid once by abcd so that a git repository in your home folder never stages abcd's records.\n" +"
    evidence: docs/how-to/install.md:202 — "other than the status line runs with `~/.abcd.noindex` in place, it writes `~/.abcd.noindex/.gitignore` holding `*`,"
  - The lay judges no repository, so on a HOME planted inside another checkout whose ~/.abcd.noindex exists, every command writes .gitignore and the marker into that checkout; the home-scope walk also does not judge a level's owner or mode (filed as iss-2610140913332484)
    evidence: internal/surface/cli/cli.go:6611 — "if layHomeIgnore(stderr, quiet) {"
    evidence: .abcd/work/issues/open/iss-2610140913332484-fsutil-ensurehomescope-and-openhomescope.md:13 — "fsutil.EnsureHomeScope and OpenHomeScope judge each level of ~/.abcd.noindex for symlinks and swaps but not for owner or mode"
- missing: (none)

Scope-condition dispositions:
- cond-2610140858479946 — untested: The delivery's git test stages only files that were untracked before the lay; nothing exercises a home repository that already tracks a file under ~/.abcd.noindex, so the assumption was neither exercised nor contradicted.
- cond-2610140858479857 — survived: The mechanism was exercised with a plain `git add -A`, the staging habit the condition assumes, and held; nothing in the delivery depends on or attempts `git add -f`.
  evidence: internal/core/homeignore/homeignore_test.go:184 — "git("add", "-A")"
- cond-2610140858471836 — survived: Real git honoured the subdirectory .gitignore in ~/.abcd.noindex: the staged list carried .zshrc and nothing under abcd's home (exercised against this machine's git only).
  evidence: internal/core/homeignore/homeignore_test.go:186 — "if strings.Contains(staged, abcdhome.Rel()) {"
  evidence: internal/core/homeignore/homeignore_test.go:189 — "if !strings.Contains(staged, ".zshrc") {"
<!-- abcd-review-end receipt=rcp-69d8a2a7ab69 -->

## Grounds

- pursued: a .gitignore holding * in ~/.abcd.noindex keeps every file abcd writes out of a dotfiles repository's git add -A; a staged abcd file after the lay would show it wrong
