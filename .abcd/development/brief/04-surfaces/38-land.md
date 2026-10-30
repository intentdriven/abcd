# `/abcd:land` — Land a Branch Whose Preflight Passed

`/abcd:land` is one landing path for any branch whose work is committed, not
only a build lane's: it gates the branch's head on a preflight receipt, pushes
the branch to `origin`, and queues its open pull request where the merge gate
allows (iss-2610150524235929, iss-2610150524246898, and the queue confirmation
of iss-2610090642376032 for this verb). Before it, a branch landed outside the
implement loop was pushed only if someone remembered to, and ready pull
requests reached the merge queue one at a time. It is listed in the
person's records group beside `build`, and its page is on a person's `/abcd:` menu.

## Sub-verbs

> _Machine-checked (`surface_coverage`, spc-27): each row records the verb's
> adr-40 bucket (`lint` / `review` / `audit` / `gate`, or `—` for a
> non-assessment verb) and its existence (`shipped` / `staged`). The existence
> fact is verified against the committed command-tree snapshot in both
> directions. The bucket cell is checked for membership of the closed adr-40
> vocabulary only: the snapshot carries no bucket field, so a bucket that is
> wrong but legal passes, and that cell stays a review-grain claim._

| Verb | Bucket | Status |
|---|---|---|

## The steps

The verb takes at most one operand, the branch, and lands the checkout's
current branch without one. A named branch is landed from the worktree that
has it checked out, because the preflight runs in the branch's own checkout.
Every step is idempotent, so running the verb again resumes where it stopped.
The preflight is the branch's own `make preflight`, run with the person's
environment and credentials, so the verb is for branches whose content the
person trusts. With `land.push` and `land.enqueue` both off nothing runs. With
`land.push` off and `land.enqueue` on the gate still runs, by design: arming a
pull request says its head is ready to merge, and readiness is the gate.

Before the steps, the origin and the forge's default branch are checked (see
the trust boundaries), and then `origin`'s default branch is fetched. A head
already on it has landed (the implement loop's own ancestor check,
`landing.Landed`). A squash or rebase merge leaves no such ancestry, so when
the ancestor check says no the forge is asked for a merged pull request from
this repository's branch whose head was the head and whose base was the
default branch (the landing package's merged-pull-request check); one there has landed too. Either way the verb reports one step, `land`, with
status `landed` (naming the pull request the forge listed), gating, pushing
and queueing nothing, so a re-run after the merge never recreates the branch
the merge deleted, whichever method the queue merged by. A pull request merged
at another head, from another repository, or into another base (a stacked
pull request, whose work is not on the default branch) proves nothing, and a
forge that cannot be asked is a refusal: no gate and no push. So is a listing
its bound (20) filled with none at the head, since the proof may lie past the
bound: the refusal says so and gives the push to run by hand.

1. **Gate.** When a preflight receipt names the head in any worktree of the
   repository, the lookup the pre-push hook makes, the gate is skipped.
   Otherwise `make preflight` runs in the branch's checkout. It is run
   directly, never wrapped in a second load wait, because the target already
   waits on the machine's load waiter. After it the head must be
   unchanged, a receipt must name it, and `origin`'s fetch and push URLs must
   read exactly as they were checked: the preflight is the branch's own and
   could add a push URL or a `pushInsteadOf`. A repository whose make file declares
   no `preflight` target has no gate that mints a receipt, and gets no push.
   The receipt is a convenience check, not a security boundary: CI re-runs
   every gate on the pushed commit.
2. **Announce** (with `land.push` on, after the release hold). Before the
   push, the verb posts `merging <item>` on the person's status branch for
   each issue, intent or pull request the branch's own commits (those not on
   `origin`'s default branch as just fetched) name in a `Refs:`, `Resolves:`
   or `Delivers:` trailer (a pull request written `#N`: a bare number is a
   word of the trailer's text, as in `Refs: spc-9 step 3`), each once,
   through `claim.AnnounceBeforePush`, the
   code `abcd announce` runs, and the push waits until each announcement has
   reached `origin` (iss-2610161645224876, dcn-2610170655202547,
   dcn-2610170704191129). So a colleague's next `abcd status` shows the
   announcement before the branch is on `origin`. An item the person already
   announced `merging`, standing a minute more, is not announced again, so a
   re-run posts nothing new, unless the person's newest announcement of it is
   now another (`merged`, say), which a reader shows instead, and then
   `merging` is sent anew. A record not yet on the default branch is not
   announced, since an announcement names only a public record, and the step
   says so; a branch whose commits name no item announces nothing, and the
   step says that too (status `skipped`). An announcement that cannot be
   published (an embargo store that cannot be checked or holds the record,
   no username to send as, `origin` refusing the push or not answering)
   refuses the landing with the reason, and nothing is pushed
   (dcn-2610170704189712): run the verb again once it is settled. When the
   branch names several items and a later one fails, the announcements already
   sent for the earlier ones stay standing, and the re-run resumes without
   sending them again while they stand. A push a
   person types by hand with `git push` posts no announcement; no hook
   announces.
3. **Push** (`land.push`). A plain `git push --porcelain origin
   <head>:refs/heads/<branch>` from the branch's checkout, through the
   networked git runner, with the repository's hooks running: never forced,
   never `--no-verify`, never the default branch. The source is the commit the
   gate checked, never the branch's ref and never its configured upstream, so a
   branch that moves after the gate cannot change what is pushed. The push of a branch other
   than the default is not a merge, so it does not wait on the facilitator.
4. **Enqueue** (`land.enqueue`). The branch's one open pull request from this
   repository is armed where the merge gate (below) allows, pinned to the head
   (`--match-head-commit`), and, where a queue gates the default branch,
   confirmed in the merge queue at that head, through the forge's GraphQL API (the forge client's pull-request
   view has no merge-queue field), asked of the repository the forge client
   named when `origin` was checked. The same query reads the forge's review
   decision, which the note reports (REVIEW_REQUIRED: held for a person's
   approval; APPROVED: approved; none: no review required for this pull
   request), never an approval hold asserted from the ruleset mirror alone,
   whose approval rule covers only the paths code owners own. A pull request
   outside the queue is reported with what holds it: the required checks that
   have not passed (`isRequired` for this pull request; at most eight,
   sanitised) before the forge's merge state, and BEHIND alone named as no
   holder, since the queue tests the pull request on top of the default
   branch. The checks are read one page at the forge's maximum of 100. BEHIND
   is never called no holder when land cannot see every holder: when the
   forge's rollup is PENDING or EXPECTED and no required check land read is
   pending (a required check with no run yet, just after a push), the note
   says checks are still pending and names the rollup's state, which is the
   authority; and when the forge has more checks than the page held, it says
   the forge lists more checks than land read. Auto-merge is toggled once a
   run, and only when this account's arming is CLEAN and still unqueued more
   than 30 minutes after the later of the arming's `enabledAt` and
   the newest time a required check passed (a check run's `completedAt`, a
   commit status's `createdAt`), all the forge's own times, with no local
   clock state: on the first real run (pull request #1018) the last required
   check passed at 12:59:42Z, the pull request was seen CLEAN at 12:59:47Z and
   the forge queued it by itself at 13:15:01Z, a 15m14s wait which a toggle
   would only have churned, and the grace is about twice that. Those times
   are compared with the local clock, and the verb makes no forge call to
   read the forge's: a local clock running fast shortens the grace by its
   lead, and the verb never toggles when `enabledAt` or a check's time is
   later than the local clock. Inside the grace the note says how long ago
   the arming was made (and how long it has been CLEAN, when its checks
   passed after the arming) and when to run the verb again if the pull
   request is still not queued. An arming of this account's on a head other
   than the one gated is refused, naming both, as after a fresh arming. The
   forge cannot tell land's arming
   from the person's own under the same account, so both count as land's; an
   arming another account made is left as it is and reported, with the
   person's way on: wait for the forge, or withdraw and re-arm it with the
   exact `gh pr merge` commands the note gives. A toggle whose re-arming fails
   says the pull request is now disarmed and gives the command that arms it.
   With no
   pull request the step stops with the `gh pr create` command; the verb never opens one (ruling R2), and the next run
   queues it. The command names a body land composed from the branch's commits (iss-220): each subject, then their
   `Refs:`, `Resolves:` and `Delivers:` lines once each, then abcd's own `Assisted-by: abcd:<version>` label and the
   models the commits' `Assisted-by:` lines name (ruling R4), written to the checkout's local tier (land makes none) and
   judged with the shell guard's checks of a posted body, the failing check named in the stop's note, so the body
   the hosted checks read was checked before it was posted.

## The merge gate

Whether the verb arms a merge is decided in one function, `mergeGate`
(`internal/core/landing/facilitator.go`), which the build loop's arm asks
too, so abcd and every managed repository take one code path (ruling R3). It
replaces ruling AM1 ([`34-build.md`](34-build.md)), which armed only where a
ruleset required a person's approval and so let a change touching no owned
path merge with no person reading it (itd-2610150602547271).

Its inputs:

- **The facilitator mode**, `facilitator.mode` in `.abcd/config.json`: `human`
  (a person is the technical facilitator; the bundled default, so installing
  abcd changes nothing about who merges) or `auto` (abcd is). It is read from
  the default branch's committed file at the tip just fetched, beside
  `~/.abcd.noindex/config.json`, never from the branch being landed, so a
  branch that commits `auto`, or deletes the default branch's `human`, does
  not decide its own gate (dcn-2610230058333177). `human` from either file
  wins. A value that is neither is refused, naming the file, its text masked
  for the terminal, and nothing is armed.
- **Who the author is.** One seam reads the pull request author's
  permission on the repository from the forge (`role_name`): Write, Maintain
  and Admin are write access; Read and Triage, a 404 or no role make a
  contributor; a custom role counts as its base, read from the same
  answer's legacy level. A permission the forge cannot be asked for, or a
  custom role whose base cannot be read, is unverified.
- **The approvals of the head.** An approval counts when it is the latest
  review by someone with write access, approves the head commit itself and is
  not the author's own; or when the technical facilitator recorded it through
  the verb's approval act, which writes `.abcd/.work.local/approvals/<sha>`.
  The second exists because the forge refuses an author's approval of their
  own pull request, and the facilitator authors most of the pull requests
  abcd opens. It is the person's act, and the shell guard refuses it to an
  agent ([`17-guard.md`](17-guard.md)).
- **A review receipt** naming the head, recorded through the verb, the proof
  that the review pair passed it.

The decision, in order:

1. An unverified author: nothing is armed, whatever the approvals, and the
   note says the role is unverified.
2. A contributor's pull request, in either mode: nothing is armed until an
   approval of its head counts, and the note says an approval by someone with
   write access is needed. Once one counts, the mode decides as for an author
   with write access.
3. `human`: the pull request of an author with write access is armed only
   once an approval of its head counts; until then the note says it waits for
   the technical facilitator's approval. The product thinker never approves a merge: their
   part is to try the outcome and judge it.
4. `auto`: the pull request of an author with write access is armed with no
   one asked, but only when a review receipt names its head; otherwise the
   note names the commit with no recorded review pair. Automated merging is never unreviewed
   merging: the reviews are automated, not skipped.

`human` holds against an agent merging by mistake, not against one set on
getting round it, and the verb says so wherever it describes the mode. The
gate assumes agents keep the rules: the recorded approval is a file in the
checkout's local tier, which anything running as the person's account can
write, and an agent holds the person's forge login and can reach routes the
shell guard cannot read ([`17-guard.md`](17-guard.md)). The approval is a rule
agents keep, not a proof of who approved; nothing proves an approval was the
person's yet.

abcd's own repository commits `"facilitator": {"mode": "auto"}`: its pull
requests are opened by people with write access, so each is armed once a
review receipt names its head, and a contributor's still waits for an approval
by someone with write access. A machine that sets `human` in `~/.abcd.noindex/config.json` holds every one
of them for the technical facilitator's approval.

An armed pull request goes by the ruleset mirror's method: into the merge
queue where one gates the default branch, and otherwise as the forge's
auto-merge with a merge commit; where the forge refuses auto-merge, the step
says so and arms nothing. abcd adds no approval rule or code-owner file to the
forge, and changes none of its settings: the forge lets a pull request merge
with no approval, so abcd enforces this itself. Every forge read the gate
makes is a GET, its fields pinned in `internal/ghcontract` after a live
read-only call, and a forge answer that cannot be read counts for nothing.

## Trust boundaries

- The remote is `origin` alone, and its every fetch and push URL must be the
  repository the forge client resolves for the checkout. An SSH host alias (a
  host of one label, no dot) is refused by name, with the `insteadOf` rule
  that maps it in the URL's own form (scp-like or `ssh://`): the verb does not
  read ssh config. A URL on another real host is refused as a different host,
  and the only advice is `git remote set-url`, never a rule that reroutes that
  host's traffic. The URLs are read again after the preflight, and a change
  is refused.
- The default branch the verb never pushes is the forge's own, asked in the
  same call, and it must match the one `origin` itself names (`git ls-remote
  --symref origin HEAD`, no fetch and no write); a forge and an `origin` that
  disagree are refused, naming both, and a forge or an `origin` that cannot be
  asked gets no gate and no push. The local `origin/HEAD` is no authority: a
  stale or missing one is reported in an `origin` step and passed over, the
  agreed default is the one the landed check fetches, and `origin/HEAD` is
  never written (iss-2610161831078335).
- A pull request from another repository is never armed (it is set aside
  when this repository's own is listed beside it), and one whose head on the
  forge is not the landed head is not armed. After a push that moved the
  branch, the pull-request listing alone is read again every 2 s, with at
  most 30 s of pauses in all (two real runs saw the forge lag the push by 12
  and 14 s; each read keeps the forge client's own timeout), and a head
  that never shows stops the step, naming the wait; nothing else repeats.
- The merge rule is read from the ruleset mirror at `origin`'s default branch
  as just fetched, never at the branch's tip: equally out of the branch's
  reach as the merge base, and fresher, so the branch cannot change the rule
  it lands by. (The implement loop reads it at the lane's base.)
- The forge client's and git's environments are scrubbed of `GIT_DIR` and
  `GIT_CONFIG_*`, as the networked git runner's are. A local git command (the
  read of `origin`'s URLs) runs through `gitutil.LocalGit`, which adds none of the
  networked runner's settings. Under a test binary, networked git is
  restricted to local transports with no credential helper, so a test can
  neither reach a real forge nor ask anyone for a credential; outside one the
  person's own configuration applies unchanged.
- A credential in a URL that git or the forge client quotes
  (`scheme://user:token@`, `https://token@`) is dropped from every note and
  refusal.
- Nothing is written to the repository's or the queue's settings.

## Configuration

`land.push` and `land.enqueue` are booleans, both true by default, read
through the layered configuration reader from `.abcd/config.json` and
`~/.abcd.noindex/config.json`
([`../05-internals/03-configuration.md`](../05-internals/03-configuration.md)).
The repository layer is read in the working directory's checkout and in the
landed branch's own, and both count. The precedence is deliberately
asymmetric: off wins from any layer, so the machine's file can switch an
outward act off for every repository and no repository's file switches it
back on. A home the reader refuses hides the machine's file, and with it any
off there, so both acts read as off and say why. A flag turns either act off for one
run (the appendix lists them); no flag turns one on. A step that is off names the layer and
file that turned it off.

## Exits

Exit 0 when every step did its act or said why it stopped or left the merge
to a person. Exit 2 on a refusal, which names the steps done before it,
reservations that cannot be fetched before the push and an announcement that
cannot be published before it included. The
JSON document carries `branch`, `head`, `checkout`, `default_branch`, `push`
and `enqueue` (each `key`, `on`, `layer`, `origin`), `pull_request` and
`pull_request_url` when one was found (or the forge proved the head merged,
the merged pull request's),
`pushed` when the push sent a commit, `landed` when the head had already
landed, and `steps`, each
with `name`, `status` (`done`, `skipped`, `off`, `stopped`, `left` or
`landed`; the steps are `origin` when it reports a stale default, `gate`,
`announce`, `push` and `enqueue`, or `land` alone), `note` and an optional
`command`; and `notices` when there are
lines to show beside the steps (a release dated ahead that the hold did not
count).

A landing a release cut holds exits 3, contention, as every verb that backs
off does (itd-2609151838312703 C7, spc-2610150728396006 step 4). After the
gate and before the push, the verb fetches the reservations, which reads
`origin`'s release tags with them, and makes the implement loop's land-fence
judgement through the one primitive both call (`claim.FetchMergeHold`): while
a release is being cut by any session but this copy's own, an unreadable
release reservation within its hold included, nothing is pushed, armed or
queued, and the refusal names the release, its holder and until when. The
push is held as well as the arming, because a push onto a pull request
already armed can merge it. A fetch that fails refuses the landing too, since a
hold it cannot see may be there, but as exit 2, not 3: nobody is known to hold
the release, and an automation backing off on 3 would wait on no one. A
checkout with no `origin` is not held. The copy cutting the release lands
through its own hold: under the concurrent-sessions rule the cutting session
has the final say on what merges before its tag. The hold is judged for the
checkout the verb runs in, whose copy key the reservation verbs there use,
not for the worktree that has the branch checked out.

A release reservation dated more than `lease.Skew` (five minutes) ahead of
this clock holds a day (an unreadable one two hours) from this checkout's first
sighting of it, which the judgement keeps in the local tier whichever fetch
brought it. A checkout that cannot keep that sighting, with no local tier or
one it cannot write, would count such a release from every reading, so it
does not count it at all: the landing goes on, and one notice line on stderr
(and `notices` in the JSON document) names the release. The tier is never made
for it. An honest cut is reserved on a clock within five minutes of the
reader's, so this lets no landing past one, with one limit: a cutting copy
whose clock runs more than five minutes fast is not counted by a checkout that
cannot keep a sighting.

## Where this sits

- The records: iss-2610150524235929, iss-2610150524246898,
  iss-2610090642376032.
- The shared primitives: `internal/core/landing`, which the implement loop's
  land stage also reads its receipt lookup, forge client and ruleset mirror
  through.
- The release hold: `internal/core/claim` (`FetchMergeHold`), shared with the
  implement loop's land fence.
- The announcement before the push: `internal/core/claim`
  (`AnnounceBeforePush`), shared with the implement loop's landing and the
  code [`abcd announce`](38-announce.md) runs.
- The plugin surface: `commands/land.md`.

<!-- surface-appendix:begin — generated from the command tree by `go generate ./internal/surface/cli`; never edit by hand -->

## Appendix: the shipped surface

_Generated from the command tree; a drift test fails `go test` when this appendix and the tree disagree. It lists flags and sub-verbs only. What each flag means is in the [CLI reference](../../../../docs/reference/cli/commands.md), and exit codes, output fields and behaviour are the prose's to state._

### `abcd land`

Sub-verbs: none.

| Flag | Type |
|---|---|
| `--approve` | bool |
| `--no-enqueue` | bool |
| `--no-push` | bool |
| `--record-review` | stringArray |

<!-- surface-appendix:end -->
