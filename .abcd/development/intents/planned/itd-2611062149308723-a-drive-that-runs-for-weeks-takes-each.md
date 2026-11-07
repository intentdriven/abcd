---
id: itd-2611062149308723
slug: a-drive-that-runs-for-weeks-takes-each
spec_id: spc-2611062151392235
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611041549461012]
refines: [itd-2610222215485265]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
target_release: next
---

# A drive takes new releases itself at a lane boundary

## Press Release

> A drive that runs for weeks takes each new release of abcd itself, and nobody stops or restarts it. Alice starts abcd drain --drive on a repository and leaves it. When a newer stable release of abcd is published, the drive notices at the next lane boundary, fetches it into a private staging file through the same verification abcd update uses, runs the new binary read-only against the drain, proves it against the full live check that a drive proves itself with before its first move, and only then swaps the installed binary and carries on from the drain's own state, with the same cap, bound, skips and profile, saying in one line which version it moved from and to. A release that fails the verification, the read-only run or the proof changes nothing: the drive says so once and goes on with the version it has. One that is swapped in and cannot start the drive goes back to the kept copy, and the drive carries on. The consent screen says in one line that the drive takes releases this way, and --no-self-update turns it off. Before, a new release could not reach a running drive without someone ending it and starting it again, and a binary left behind fell behind the records newer binaries wrote.

## Why This Matters

A drive compares abcd's binary with the one it started on before every stage, and a difference ends the run (`internal/core/runner/wall.go`, `checkStageFiles`; `.abcd/development/brief/04-surfaces/35-drain.md`). That rule is right: a binary that changes under a run is a binary something else changed. But it means a release cut while a drive runs cannot reach it. The drive keeps running the version it started on, and the one way to move it forward is to end the run and start it again, which the technical facilitator has ruled a long run never has (dcn-2611062149362998). A binary left behind also falls behind the records newer binaries write: a record gate, a schema field or a refusal added since its release is unknown to it, and a long run on it stalls on records it cannot read rather than on work.

`abcd update` already holds a release binary to the two checks that do not trust the release page: the build provenance the release workflow signed, verified in process against the Sigstore trusted root and bound to the commit the tag names, and the release the binary's own build metadata claims (`internal/core/update/update.go`, `verifyRelease`; `commands/update.md`). A drive already proves the binary it runs in one short live launch before its first move (`ProveWall`). This intent joins the two: the verification says the bytes are abcd's release, the proof says the release holds every wall on this machine, and only a binary that passes both replaces the one running. Nothing new is trusted: the drive swaps a file it is already running and proves the replacement the way it proved itself.

This is a drive's capability for any repository abcd manages. The repository's own work carries on across the swap because the drain's state is in abcd's files, not in the process (itd-2611041549461012).

## Mechanism

We expect a drive to take a release unattended without weakening what a drive proves, because the release reaches the installed binary only after the same verification `abcd update` makes and after the new binary has served as the approver and the hooks of a full live launch from its staging file, and because the re-executed drive pins the new binary as its baseline at its own start and resumes from state the drain wrote to disk rather than from anything the old process held. The claim is shown wrong if a binary that fails the verification or the proof ever reaches the installed path, if a swapped drive loses its cap, bound, skips or profile, or if anything but a release that passed them reaches the installed path. A failed read-only run keeps that version out and leaves the drive on the version it has.

## Scope Conditions

- Holds for a binary installed from a published release: one whose own build metadata claims a release tag, that is a regular file abcd may swap (not a plugin root's binary, not a package manager's, not the track-latest shim). A source build, whether a person's or a session's, never self-updates, and the consent screen says so when it applies. <!-- cond: cond-2611062151394419 -->
- Holds for `abcd drain --drive` alone, the one process that runs unattended for weeks; `abcd update` stays the person's verb and abcd otherwise never checks for or applies a release on its own (adr-38). <!-- cond: cond-2611062151393460 -->
- Holds for the release origin and the transport `abcd update` uses, over a network that answers; a fetch, a lookup or a verification that fails leaves the drive on the version it has, and no wait is added for it. <!-- cond: cond-2611062151397006 -->
- Holds on macOS and Linux, where the installed binary sits in a directory the account can write; there is no Windows target. <!-- cond: cond-2611062151391082 -->
- Holds for a new binary whose failure reaches the drive's own start code before its first move, a panic there included: that start restores the kept copy and carries on with it. A binary that fails to load, panics before that code or on another goroutine, or hangs at its start, and a fault that first shows after the first move, are the person's to recover, from the previous binary kept beside the installed one. <!-- cond: cond-2611062151394636 -->

## Acceptance Criteria

- **Given** a drive on a release-installed binary, **when** a lane has ended and the next is not yet opened, or the drive wakes from a wait before opening one, or a stuck lane or judgement has held the check for an hour, and at least an hour has passed since its last check, **then** it asks the release origin for the latest release once, and at no other point of the run: never inside a lane, never from a stage, and never more often than about once an hour.
- **Given** a latest release that is not newer than the running version, or that carries a pre-release suffix, or that the origin's latest-release answer did not name, **when** the drive reads it, **then** it fetches nothing and says nothing new until a later release.
- **Given** a newer stable release, **when** the drive takes it, **then** it fetches the platform binary into a private staging file beside the installed one through the verification `abcd update` uses (the release's checksums, the build provenance the release workflow signed and bound to the commit the tag names, and the release the bytes' own build metadata claims), and a verdict on the release at any of these (a digest or provenance that does not hold, a claim that does not match) changes nothing, is said once in the drive's log, and is not retried for that release.
- **Given** a fault that says nothing about the release (a network fault, a server error, a rate limit or a timeout while fetching it), **when** the drive meets it, **then** it changes nothing, writes one line, does not mark the release as tried, and tries again at the next check; a dead network costs one short check per interval, one bounded request and no fetch, so it does not hold a lane boundary.
- **Given** a verified staging file, **when** the drive proves it, **then** the staged binary serves as the approver and the hooks of the same live launch the drive proves itself with before its first move, and a wall that does not hold changes nothing: the installed binary and the profile's settings are as they were, the drive says which wall once, carries on with the version it has, and tries again only on a later release.
- **Given** a verified staging file, **when** the drive rehearses it before the proof, **then** the staged binary accepts the arguments its re-execution would be given and answers `drain --dry-run --json` with a plan, within a bound, writing nothing; anything else changes nothing, is said once per reason, and the drive carries on with the version it has and tries that version again at later checks.
- **Given** a staging file whose proof passed, **when** the drive swaps it in, **then** the swap is atomic as `abcd update`'s is, the previous binary is kept beside the installed one, the drive re-executes itself with no new question and with none of the arguments the drain's state already holds, and the re-executed drive resumes from the drain's own state with the same cap, the same bound, the same skips and the same profile, pins the new binary as its baseline, and prints one line naming the version it moved from and the one it moved to.
- **Given** a re-execution that the system refuses, **when** the drive sees it, **then** it restores the previous binary from the kept copy, says so once, and carries on with the version it has.
- **Given** a resuming start that fails, **when** it sees the failure, **then** it tries again on a schedule of a few minutes that honours a stop, so a passing failure counts against no release.
- **Given** a new binary that starts and keeps failing before its first move, a panic included, **when** the start sees the failure, **then** it restores the kept copy, says so in one line naming both versions, and executes the kept copy, verified as the release it replaced, which proves itself in one live launch, goes on from the drain's state with no question and does not try that version again; a restore or an execution that fails ends with exit 2 naming the kept copy and the way to restore it, which is the person's act, as is recovery from a fault that first shows after the first move.
- **Given** `--no-self-update`, **when** the drive runs, **then** it checks for no release and behaves as a drive did before this intent; no configuration key, environment variable or repository file turns the check on or off.
- **Given** the consent screen, **when** it shows, **then** one line says the drive takes newer releases this way and that `--no-self-update` turns it off, or, on a binary that never self-updates, one line says why; a drive that may take a release says in one more line that any other drive on this account ends when this one moves, whatever flags that other has, and a second drive is not refused (dcn-2611070019463013).
- **Given** any file a stage can write, including the installed binary, the staging name, the release mark, the install record and the profile's settings, **when** a stage writes it, **then** the writing brings no binary in that has not passed the verification and the proof, and decides nothing of what runs after it; a failed read-only run keeps that version out: the installed binary and the settings are compared with the start before every stage as today, an attempt at a start after a failed one is held to what the process began on before it pins anything and proves itself live, the staging file's digest is read again after the proof and must equal the verified digest before the swap, and the decision reads only the flag, the clock, the running binary's own build metadata, the installed binary's kind and the release origin.

## SOTA

The existing alternatives, each with its rough maturity, and the path taken:

- **Self-updating daemons that swap their own binary on a signed release and re-execute** (package-managed services, the Go self-update libraries abcd's update verb already uses through `selfupdate.CommitBinary`): mature, and they settle the swap itself, the atomic rename with a kept previous copy and the re-execution with the same arguments. They verify a signature and swap; none runs the replacement in a live launch before it depends on it, because none has a live proof to run.
- **Update notifiers with a background version-discovery request** (the notifier pattern adr-38 scored down): mature, and rejected by adr-38 for every implicit operation; this intent keeps the request on the person's yes and in one verb.
- **Blue-green process supervision** (a supervisor keeps the old process to restart the new one): mature in service hosts, rejected here because no process stays alive beyond the run (dcn-2610180720366282).

Path taken: 2, the native floor with a seam. The swap, the kept copy and the re-execution are the update verb's and the swap library's as they stand; what is bespoke is the order, the live proof of the staged binary before the swap, and the resume from the drain's own state, which no library can supply because they are abcd's own walls and records.

## Open Questions

_None open. The design was settled before filing; the decisions below record what settled it and what the code contradicts._

## Decisions

- 2026-11-06, the technical facilitator (dcn-2611062149362998): a long unattended run has no stops and no restarts, so a drive takes new releases itself; the capability goes into the release being prepared.
- 2026-11-06, filed from the ruling: the check runs at a lane boundary only, at most about once an hour, and reads the origin's latest-release answer alone; a tag a file or a stage names is never a candidate, and no downgrade and no pre-release is ever taken.
- 2026-11-06, filed from the ruling: the proof runs the new binary from its staging file before the installed binary is touched; a release that fails the verification or the proof changes nothing, is said once, and is tried again only on a later release. The proof writes the profile's settings naming the staged binary for the launch and restores the settings naming the running binary after, so a failed proof leaves the drive exactly as it was.
- 2026-11-06, filed from the ruling: the switch is a flag and nothing else, `--no-self-update`, so a stage cannot turn it and a repository cannot choose it (adr-2610222222326104, condition 1).
- 2026-11-06, filed from the ruling: a source build never self-updates. The test is the binary's own build metadata, the same reading `abcd update` makes of a release binary's claim: a build that claims no release tag (a source build's version, a pseudo-version, a modified tree) is never updated, and so is a binary a plugin root, a package manager or the track-latest shim owns.
- 2026-11-06, filed from the ruling: recovery is bounded. The previous binary is kept beside the installed one. A re-execution the system refuses is restored automatically and said; a new binary that started and cannot go on is restored by the person from the kept copy, and the exit names it.
- 2026-11-06, the technical facilitator, on the settled design: a drive that ends on its own is what the ruling forbids, and the wall proof exercises no drain verb, so the candidate also runs read-only against the checkout's drive before the swap, and a re-executed start that fails its own preconditions goes back to the kept copy and carries on. This takes the place of the bounded recovery above for a failure the start itself sees; what the start cannot see stays the person's.
- 2026-11-06, the technical facilitator, on the first build: a run lasts weeks and a passing fault must not cost it a weekly release, so only a verdict about the release refuses a tag for the run (a provenance or digest mismatch, a failed proof, a failed rehearsal, a failed start), and a fault that says nothing about it is tried again at the next check. Filed before as not retried; this replaces that.
- 2026-11-06, the technical facilitator, after the security review: a failed read-only run does not latch its tag, since it reads the checkout as it stands; later checks try it again, and each reason is said once. This replaces not trying that version again.
- 2026-11-06, on building the read-only run: it reads the checkout, so what the design holds is that no file on the account brings a binary in. A failed run keeps a version out and leaves the drive where it was, the failure this design already accepts.
- 2026-11-06, filed on reading the code: the check is a network request a drive makes on its own, which adr-38 ruled out of every implicit operation and named as a thing requiring a record in the open. The yes at the consent screen is the explicit ask, and its line names the fetch; the proposed ADR adr-2611062149324491 refines adr-38's tier 2 for this one verb and is flagged for a person to confirm as such.
- 2026-11-06, filed on reading the code: the drive keeps `~/.abcd.noindex/release-mark` and the install record as `abcd update` does, after a swap and never as an input: the mark is raised and the install record's digest refreshed so the swapped copy still reads as abcd's own, and neither is read to decide a swap, since a stage on the same account can write both.

## Decomposition

Hand-run four-piece table, 2026-11-06, from the settled design the technical facilitator ruled; the design arrived with its routing named.

| Part | Kind | Home |
|---|---|---|
| a drive takes each newer stable release at a lane boundary: the hourly check, the staged fetch, the proof from the staging file, the atomic swap, the re-execution that resumes the drain's state, the one line, the consent line, the flag that turns it off, the kept previous binary | capability | this intent |
| an unattended process replaces its own binary only after the verification the update verb makes and a live proof the new binary passes, and nothing a stage writes decides it; a source build never self-updates | trust rule | adr-2611062149324491, proposed; a brief invariant when a person accepts it (its wording is in the ADR's consequences) |
| a long unattended run has no stops and no restarts | ruling | dcn-2611062149362998, recorded; no principle, since it binds one run's shape and not every verb |
| a drive's own network request for a release, where adr-38 names auto update discovery as needing a record in the open | trust rule, existing | adr-38, refined by the ADR above for this one verb; flagged for a person to confirm as a refinement and not a reversal |
| the seams: the target classification of the running binary, the staged fetch and the proof naming a candidate, the swap keeping the previous copy, the re-execution's hidden resume flag, the drain state read back | plumbing | this intent's spec, then the brief's drain and update chapters |
| the drain's own state is what a drive resumes from | capability, existing | itd-2611041549461012; cited, not refiled |

Verdict: FILE-AS-IS, with the trust rule routed to its own proposed ADR as the facilitator's instruction named. Typed links: `refines` itd-2610222215485265 (its drive, whose baseline comparison this intent moves rather than removes); `builds_on` itd-2611041549461012 (the restart path's reading of the drain's state); itd-130 (the update verb's verification and swap) is shipped and is cited in the text rather than edged, since a shipped record is not backfilled. Advisory reversal flag: adr-38's consequence that checking for updates is forever a verb the person runs; the yes names the check, so the record proposes a refinement of tier 2, for a person to confirm.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: a drive can take a release unattended without weakening what it proves, because the update verb's verification and the drive's own live proof both exist and can run on a staging file before the installed binary is touched; shown wrong if a binary that fails either ever reaches the installed path, or if a re-executed drive loses its cap, bound, skips or profile
