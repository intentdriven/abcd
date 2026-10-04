---
id: itd-2609091034175565
slug: nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has
spec_id: spc-2610031753359849
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
refines: [itd-2609221656373558, itd-2609211116005482]
severity: minor
related_issues: [iss-2609020716570699]
origin: extracted-from-record
production_mode: hand-written
related_intents: [itd-2609091416295622, itd-2609091416304128, itd-2609091014076309, itd-2609150819440345, itd-2610031259176838, itd-118]
related_adrs: [adr-2609091248200336, adr-2610031352212697]
supersedes: [itd-33]
impact: additive
---

# Before an agent starts on an item, it reserves it on the shared remote, and everyone who works on the project sees the reservation

Typed links: `refines` [itd-2609221656373558](../shipped/itd-2609221656373558-two-orchestrator-sessions-in-one-user-account-share-an.md) (the shipped in-run claim, whose lease shape, two-hour default, back-off refusal and liveness judgement this record carries to the shared remote; the judgement moves into one function both use) and [itd-2609211116005482](itd-2609211116005482-abcd-build-next-picks-the-readiest-planned-intent-itself-wri.md) (the pick's exclusions gain "held by another person's reservation" and the view's age); `supersedes` [itd-33](../superseded/itd-33-agent-communication-infrastructure.md) (agent coordination through a local `active-work.json`, which decision 1 rules out); `related_issues` [iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md) (the duplicated re-fixes and pull requests this record exists to prevent); `related_intents` [itd-2609091416295622](../shipped/itd-2609091416295622-a-session-sees-the-records-its-sibling-worktrees-hold-before.md) (the read-only sibling-worktree listing; prose cross-reference to the 2026-09-09 ruling, which gave it its own record), [itd-2609091416304128](../drafts/itd-2609091416304128-capture-resolve-refuses-a-record-the-default-branch-has-alre.md) (the resolve-time refusal; prose cross-reference to the same ruling), [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md) (the machine-scoped worktree store), [itd-2609150819440345](../drafts/itd-2609150819440345-which-session-holds-which-worktree-branch-or-record-is-coord.md) (the wider register of running sessions, kept for later; its decision 4 is this record's transport), [itd-2610031259176838](../drafts/itd-2610031259176838-someone-outside-the-project-can-reserve-a-piece-of-work-too.md) (outside contributors, who cannot push a reservation branch), [itd-118](../drafts/itd-118-merged-work-leaves-no-residue-abcd-managed-repos-delete-a-pr.md) (the home of tombstone-branch deletion); `related_adrs` [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md) (a tool never writes in space it was not handed) and [adr-2610031352212697](../../decisions/adrs/2610031352212697-claims-are-branches-on-the-shared-remote-nothing-kept-on-one.md) (reservations are branches on the shared remote, nothing kept on one computer is the register, and reservation text is data; proposed). Prose cross-reference, not a typed link, because no schema field carries the relation ([iss-2609091256264547](../../../work/issues/open/iss-2609091256264547-three-of-the-four-mandated-typed-relations-cannot-be-written.md)): [iss-2608220750029993](../../../work/issues/open/iss-2608220750029993-session-presence-detection-for-shared-checkouts-each-live-se.md), session presence, which this record answers for records and not for sessions.

**On words.** Every surface a person reads says **reservation**. Inside the record and the code a reservation is a **claim**, the name the shipped in-run `implement claim` already uses; the two words name one thing.

## Press Release

> **Two people's agents no longer start the same piece of work.** When an agent starts on an item, abcd reserves it for the person the agent works for. The reservation is a small branch on the project's shared copy that nobody ever merges: Everyone who works on the project sees it, whichever computer they work on, and so does an agent working from a cloud session. If someone else already holds the item, or the files the work will touch, abcd says who and until when, and the agent takes something else. A reservation lasts two hours and is renewed every hour while the work goes on, so one left behind by a closed laptop frees itself. It stops counting the moment the work is merged. Starting a build reserves the item, and a person working by hand reserves with one short command.
>
> "There are two of us now, on two computers, each running agents, and sometimes a cloud session as well," said Maya, an autonomous-development practitioner. "My agents could only see what my own machine knew, so they would happily start an item my colleague's agents were halfway through. Now they see the reservation before they begin, and the one that loses finds out in seconds, not after a night's work. My laptop and my cloud session count as me, so they never lock each other out."

## Why This Matters

A second person now works on the project, from their own computer, and the repository is public; a cloud session can join the work with its own copy. Everything abcd used to keep agents apart lived on one computer: The shared run's lease file, the sibling-worktree listing, the notes sessions leave each other. None of it reaches the second person or the cloud session. Their agents and ours can each start the same item, and the first anyone hears of it is when the second set of changes collides at merge, after the work is written, tested and reviewed. That is the collision this record has carried since 2026-09-01; it is now certain to recur, because the participants share nothing but the remote.

The autonomous runs taught the same thing from inside one account (the multi-agent coordination synthesis of 2026-10-03, [`2026-10-03-multi-agent-coordination-sota-and-run-lessons.md`](../../research/notes/2026-10-03-multi-agent-coordination-sota-and-run-lessons.md), which ranks the run lessons and the state of the art):

- **A reservation taken by one atomic check works.** The shipped `implement claim` (an exclusive file create with a two-hour lease) saw no collisions inside one account. But it is per account and per run: A peer outside the run still re-fixed two issues the run held.
- **Only shared, durable records carry work across sessions.** Habits held in one session's prose died at every handover; only the decision log carried authority, and the product thinker could not see which session ran the run.
- **Seeing records is not seeing who works on them.** The listing shows what a peer's tree holds, not who is working on what, so agents mistook each other for foreign peers.
- **The merge is the one serial point.** Coordination that only bites at merge is coordination that bites after the cost is spent.
- **Every channel a peer can write to is untrusted input.** The state of the art's incidents of 2026 are agents following instructions planted in issue text, comments and branch names.

The state of the art names the same primitive this record adopts: A reservation taken by a create-only compare-and-swap on a git ref, a lease with an expiry, a re-check at merge time (fencing), and the files the work will touch declared up front, because concurrent agent pull requests that touch one file conflict often.

### How the record got here

The record began as the remedy of [iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md): On one night of the 2026-09-01 run a peer session re-fixed two issues a paused branch had already fixed, and two of its pull requests duplicated merged work; the resolution gate refused the push, the first moment anything said no. The recorded remedy was a `claimed_by` stamp on the record, pushed alone, plus a duplicate guard and a resolve-time refusal. On 2026-09-09 two local near-collisions (a record nearly minted twice, a worktree nearly created over a peer's) widened the draft to cover sessions on one machine.

Two adversarial reviews of the widened draft then found five things, and the ruling of 2026-09-09 in the decision log split the record in three: The listing ([itd-2609091416295622](../shipped/itd-2609091416295622-a-session-sees-the-records-its-sibling-worktrees-hold-before.md), shipped) and the resolve-time refusal ([itd-2609091416304128](../drafts/itd-2609091416304128-capture-resolve-refuses-a-record-the-default-branch-has-alre.md)) went their own way, and this record kept the claim, marked not ready. The findings, and what each becomes under this design:

- **A stamp on an issue record is invisible to an older reader.** The issue schema is a closed allow-list (`internal/core/issueschema/issueschema.go`, mirrored by `record_schema` in `internal/core/lint/schema.go`), so a stamp needed a two-release migration. This design writes no field into any record, so the migration is gone with the stamp.
- **The refusals fired after the work.** Here the reservation is taken when a build starts, `build next` passes over a reserved item, and the land stage re-checks.
- **No staleness threshold was safe in both directions.** Here nothing detects liveness: A lease expires unless renewed, bounded by the lease in one direction and by the hourly renewal in the other, and merged work releases its reservation without any step.
- **The pushed half cost a merge-queue pass per claim**, fifteen to sixteen minutes measured on this repository. A reservation branch opens no pull request and enters no queue.
- **Every collision on record was covered without a claim.** That held while every session shared one machine. A second person or a cloud session has no sibling worktree to list, and the resolve-time refusal fires only after their work has merged.

Three homes were weighed for a lease on one machine: The shared `.git`, the machine-scoped `~/.abcd/` lane keyed on the root commit, and the host's session registry (a source through hooks, never a surface, per [itd-22](../drafts/itd-22-harness-portability.md)). All three are one computer's knowledge, which decision 1 rules out as the source of truth.

### The reviews and the planning interview (2026-10-03)

Two adversarial reviews of this rewrite followed, design and feasibility, then record discipline; both returned NEEDS-WORK, and their evidence is in the synthesis's Review evidence section. Review 1 found the primitive right and the transport wrong: A hosted cloud session's git proxy pushes branches only (HTTP 403 on a ref outside `refs/heads/`); abcd's hazard registry and the cloud proxy rule out lease flags and deletions, while a plain push is already the compare-and-swap; the committed pre-push hook refused every claim push; the merge queue merges after the fence; the claimer sets every time; and the lease arithmetic must have one home. Review 2 found the grounds unreadable to the second person, the criteria compound, and logged rulings reversed only inside the draft; the synthesis was promoted, the criteria split, and the rulings logged. The planning interview then answered the pre-pass's questions: Decisions 12 to 32 below.

## What's In Scope

**How the two specs are realised** (decision 38). `abcd intent plan` mints spec 1, whose `## Steps` list the spec-1 work below first and the spec-2 work after it. When spec 1's steps have landed, `abcd spec close <spc-N> --remainder <slug>` mints spec 2, carrying the unlanded steps. The intent ships when spec 2 closes.

**Spec 1: Reserve, renew, release, the listing and the hook.**

- **A reservation is a branch nobody merges**, `refs/heads/abcd-reserved/<record-id>` on `origin`, holding a chain of commits whose every tree is the empty tree (`4b825dc642cb6eb9a060e54bf8d69288fbee4904`). abcd writes each commit with `git commit-tree`, so the caller's HEAD, index and working tree are never touched and a detached HEAD makes no difference. Each commit's message carries trailers: The record id; the holder, the person's GitHub username; the copy, an opaque working-copy key naming the checkout the reservation was made from (never a path or a host); the footprint; `claimed_at` (when this commit was written); `expires_at`; `released_at` on a tombstone; and `Assisted-by: abcd:<version>`, which abcd writes itself because `commit-tree` runs no `prepare-commit-msg` hook. Author and committer are the person's git identity. No record file gains a field.
- **The holder is a person, named by their GitHub username.** abcd reads it from a no-reply address in `user.email` (`<id>+<username>@users.noreply.github.com` or `<username>@users.noreply.github.com`), or else from a value set once per computer in the user-level `~/.abcd/config.json`, never from the repository. With neither, every write is refused, naming both sources. A laptop and a cloud session of one person resolve to one username and count as one person.
- **A reservation also records the copy it was made from** (decision 33). The working-copy key is a random value minted once per checkout and kept in that checkout's local tier, so it names neither a path nor a host. The same copy continues and renews freely, so a fresh agent in the same folder carries on. Another copy of the same person, on a second computer or in a cloud session, is refused while the reservation is live, with the message "you are already building <id> on another computer, until <expiry>". To continue elsewhere on purpose, the person passes `--take-over` (on `abcd reserve` and `abcd build`), which pushes a child naming the new copy; the old copy's renewal and land fence then refuse. The person stays the unit for the cap, the per-person count and the identity.
- **Every act is a plain push, never a lease flag or a deletion.** The verb first fetches the reservation branch, judges its tip, then pushes:
  - **Reserve**: A parentless commit; the remote refuses it as non-fast-forward when the branch exists, and the verb then re-reads and refuses naming the holder.
  - **Renew**: A fast-forward child, when the tip names this person and this copy.
  - **Take over from one's own other copy**: With `--take-over`, a fast-forward child naming this copy, when the tip names this person and another copy; the output names the copy it replaced.
  - **Take over**: A fast-forward child of a tip whose effective expiry, judged by the taker's clock, passed at least five minutes ago; the output names the holder and expiry it replaced.
  - **Release**: A fast-forward tombstone child carrying `released_at`, with `expires_at` set to it, when the tip names this person and this copy.
  - A reservation another person holds is refused with the back-off exit, naming the holder and the effective expiry.
- **A footprint is required.** Every reservation names the repository-relative paths the work expects to change; one without a footprint is refused.
- **One liveness judgement.** The effective expiry is `min(expires_at, claimed_at + 24h)`, 24 hours being the shipped `MaxLease`; a longer lease is refused. A tip that cannot be parsed, or exceeds the message size cap, is reported as unreadable, held until its commit date plus the two-hour default lease, and then open to a takeover that names what it replaced. A reservation stops counting the moment its record is terminal on the default branch as last fetched (`refs/remotes/origin/<default>`), whatever its expiry. One function holds this judgement, and the shipped in-run `implement claim` calls it too.
- **The verbs.** `abcd reserve <record-id> --footprint <path>…` and `abcd reserve <record-id> --release` cover hand work, each a push by its documented meaning. `abcd peers` fetches `+refs/heads/abcd-reserved/*:refs/remotes/origin/abcd-reserved/*` with pruning and lists every reservation with its holder, footprint, effective expiry and state (live, expired, released, unreadable), and each person's count of live reservations.
- **Only `origin`.** A checkout whose remotes do not include one named `origin` is refused, naming the remotes found.
- **Every failure refuses loudly and writes no local reservation**: No remote; no write permission; offline; a remote or proxy that rejects the push. Each names its cause and says that no reservation was made.
- **The pre-push hook exempts reservation chains, narrowly.** `.githooks/pre-push` passes an update to `refs/heads/abcd-reserved/*` without a preflight receipt only when every commit in the pushed range (every commit, on a create) has the empty tree; anything else under that prefix is gated as today. The non-fast-forward refusal stays and covers reservation branches too. No reservation push skips the hooks.
- **Public safety.** A reservation names no host and no home path; the message passes the outbound policy (no session URL, no tool footer). A reservation is parsed by its trailer keys and rendered as quoted data, never handed to an agent as instructions; an unknown key is ignored. Footprint paths are trailer values only and never reach a shell or a refspec. The branch name is built only from a record id that passes the record-id grammar. The message is size-capped, and the verb refuses to write a larger one.

**Spec 2: The land fence, the cap, the footprint refusal, the renders, and build and drain.**

- **`abcd build <itd-N>` reserves the intent when it starts**, with the footprint its spec's `## Footprint` names, and `abcd build <itd-N> --release` gives it back. **`abcd drain` reserves each issue it takes**, with the issue's `found_at` path as its footprint. A run with no footprint to give is refused at the reservation, naming what is missing. These pushes join each verb's documented meaning.
- **Renewal during a run.** `implement step` renews the run's reservation whenever half its lease has passed (every hour for the two-hour default); its documented meaning gains the reservation read and write.
- **The land fence.** Inside `implement step`: Immediately before arming the merge (or leaving the pull request for a person), the land stage fetches the reservation branch and refuses unless the tip still names this person and this copy, naming the person or the other copy that holds it now; when it is, the stage renews it with a lease of at least the merge queue's `check_response_timeout_minutes` from `.abcd/work/rulesets/main-protection.json` (45 here) plus a margin, or the default lease where no mirror exists, and keeps renewing at half-lease while the merge waits. Once the record is terminal on the default branch the reservation has stopped counting; a later step tidies up with a tombstone, and leaves and reports a tip that is no longer this person's.
- **A per-person cap.** The verb refuses a reservation that would take one person past ten live reservations, naming the cap and their live reservations.
- **Footprint overlap is refused.** A new reservation whose footprint shares a path with another live reservation is refused, naming that reservation, and the agent takes other work.
- **Renders read only the disk.** `abcd <record-id>`, the board, `build <itd-N>` and `build next` read `refs/remotes/origin/abcd-reserved/*` as last fetched, through the shared liveness function, and say how old that view is, or that nothing has been fetched. `build next` passes over an item another person holds, naming the reservation and the view's age in its exclusion; `build <itd-N>` refuses it at the `peers` check. The "as of last fetch" reader and its age line are the same ones the resolve-time refusal (itd-2609091416304128) uses.
- **Names never collide.** The prefix `abcd-reserved/` is disjoint from the implement loop's lane prefix `build/`, and the land stage's branch removal never touches a reservation branch.

### Records to touch at ship

- Brief invariant 10 gains one clause: A git branch push is not remote configuration (decision 12).
- Brief invariant 7's documented-meaning entries: `reserve`, `build`, `drain`, `peers` and `implement step`.
- A new surface page for the reservation verb in `.abcd/development/brief/04-surfaces/`, with its README row; `34-build.md` corrected (the in-run claim within one account, the reservation across people); `08-abcd.md` and `27-implement.md`.
- `commands/reserve.md` (new), `commands/build.md`, `commands/drain.md`, `commands/peers.md`, `commands/implement.md` and `commands/abcd.md`.
- `AGENTS.md`'s concurrent-sessions and attribution sections; the `.githooks/pre-push` header; the verification matrix; a reservation-push `known_good` in `internal/core/guard/defaults/guard.json`.
- `docs/reference/terminology.md` gains "reservation"; `ACKNOWLEDGEMENTS.md` gains any source the build adopts.
- adr-2610031352212697 moves to accepted.

## What's Out of Scope

- **The wider session register**, saying which sessions run and what each holds: [itd-2609150819440345](../drafts/itd-2609150819440345-which-session-holds-which-worktree-branch-or-record-is-coord.md), kept as a draft for later.
- **Outside contributors**, who cannot push a branch to the remote: [itd-2610031259176838](../drafts/itd-2610031259176838-someone-outside-the-project-can-reserve-a-piece-of-work-too.md).
- **The resolve-time refusal** of a record already terminal on the default branch: [itd-2609091416304128](../drafts/itd-2609091416304128-capture-resolve-refuses-a-record-the-default-branch-has-alre.md).
- **Deleting reservation branches.** Tombstones are deleted by [itd-118](../drafts/itd-118-merged-work-leaves-no-residue-abcd-managed-repos-delete-a-pr.md)'s sweep from an ordinary machine, since the cloud proxy rejects deletions.
- **Keeping one person's sessions in one copy apart.** Any session in the copy that made a reservation may renew or release it; sessions sharing one run are kept apart by the in-run `implement claim`, unchanged.
- **Judging that two records describe one observation**: [itd-87](../drafts/itd-87-recurrence-escalation-in-capture.md).
- **Moving, listing or reclaiming worktrees**: [itd-2609091014076309](itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md).
- **Any field on a record file.** No `claimed_by` stamp, no schema change, no migration.
- **A second backend.** One transport, the reservation branch.
- **A lock.** Anyone with write access can push to or delete a reservation branch from an ordinary machine, and the rulesets here target only the default branch; a reservation is advisory among trusted writers.
- **Detecting whether a session is alive.** Expiry, renewal and the derived release replace it.
- **Implicit network traffic.** No render, gate, hook or session start fetches or pushes a reservation.
- **A shared remote with another name than `origin`**, and any change to the user's git configuration.

## Mechanism

We expect two people's agents, and a cloud session's, to stop starting the same item because the reservation is taken by one atomic compare-and-swap on the one place they all share, a branch on the remote every participant can push, at the moment work starts (a build or a drain reserves as it begins), and because the verbs that choose work (`build next`, `build <itd-N>`) read it from disk; we expect the footprint refusal to keep apart work on different items that would touch the same files, and the land fence to catch a reservation lost mid-work before the merge is armed. We expect a person's own second computer or cloud session to be stopped the same way, because the reservation names the copy it was made from, while `--take-over` keeps a deliberate move one flag away. We expect it to cost little enough that nobody routes round it, because a reservation is one plain push of one empty commit, seconds, with no pull request and no queue pass, and because merged work releases itself. It is falsified if, after it ships, two people's runs work one item, or overlapping files, in parallel although both reserved; if live work regularly loses its reservation to a takeover because renewal does not keep pace; if a cloud session cannot reserve; or if the footprint refusal turns away so much work that people widen footprints or skip the verb.

## Scope Conditions

- Holds for participants with write access to the shared remote, all pushing to one remote named `origin`. Someone who cannot push a branch cannot reserve, and is out of scope here. <!-- cond: cond-2610031753354810 -->
- Holds while the remote, and any proxy in front of it, accepts fast-forward branch pushes under `abcd-reserved/`. The hosted cloud session's proxy does, by the 2026-10-03 probe; the branch shows in the forge's branch list and may raise a compare-and-pull-request banner, costs the product thinker accepted. <!-- cond: cond-2610031753352651 -->
- Hypothesis until a real push proves it: A reservation branch starts no workflow, because every `on: push` trigger in `.github/workflows/` filters on the default branch or release tags. It is argued from the workflow definitions, not observed. <!-- cond: cond-2610031753357820 -->
- Holds while the remote stays under the forge's limit of 5,000 branches ([repository limits](https://docs.github.com/en/repositories/creating-and-managing-repositories/repository-limits)). Every reserved record leaves one tombstoned branch until itd-118's sweep deletes it. <!-- cond: cond-2610031753357092 -->
- Holds among trusted writers. Anyone with write access can push onto or delete a reservation branch from an ordinary machine, and nothing here prevents it. <!-- cond: cond-2610031753351641 -->
- Holds while each person's computers resolve to one GitHub username. A person whose laptop and cloud session resolve differently counts as two people, and their own sessions refuse each other. <!-- cond: cond-2610031753358669 -->
- Holds while the participants' clocks agree to within minutes. Expiry is judged by the reader's clock against times the claimer wrote; the five-minute takeover margin absorbs small skew, and the 24-hour clamp bounds a claimer whose clock or `expires_at` is far ahead. <!-- cond: cond-2610031753352100 -->
- Holds while renewal runs at least once per lease. `implement step` renews only when it is called; a run whose steps are more than an hour apart can lose its reservation to a taker, and the land fence then refuses its merge. <!-- cond: cond-2610031753358608 -->
- The cap and the footprint refusal are judged on the view fetched just before the push. Two reservations on different records pushed within the same seconds can both pass, though they overlap or together exceed the cap. <!-- cond: cond-2610031753354769 -->
- The fence covers the merge queue's wait only while `implement step` keeps renewing; the derived release then covers the merge itself, so a merged record never needs a step to stop counting. <!-- cond: cond-2610031753356382 -->
- Renders and `build`'s peer check are only as fresh as the last fetch of the reservation branches, by `abcd peers` or by any `git fetch` of `origin` under the default refspec. A reservation taken since then is invisible to them, and the reserving push itself refuses a second reservation on the same item. <!-- cond: cond-2610031753358785 -->
- A reservation's contents are public. Anyone who can read the repository can fetch the branches, so a reservation carries only what a commit would already publish. <!-- cond: cond-2610031753359470 -->
- A cloud session, running on a hosted machine with its own clone, holds the same position as a second person's machine. It sees only what is on the remote, and nothing in this machine's local tier, run store or desktop notes. <!-- cond: cond-2610031753355764 -->

## Acceptance Criteria

Confirmed by the product thinker at the planning interview, 2026-10-03: person-visible criteria walked one by one (C2, C4, C5, C8, C12–C13, C15, C16, C27, C34–C39, C42, C47, C48, C50), technical criteria accepted as lists; C4, C20 and the new C52–C53 walked one by one after the take-over ruling and confirmed; C4 and C20 revised by decision 33 (take-over). One Given-When-Then and one verdict per bullet; each is testable with a local bare repository standing in for the remote and further clones standing in for a second person, a second computer and a cloud session. P and Q are two people with different GitHub usernames; C52 and C53 were added with decision 33 and are numbered after the rest so the confirmed numbering stands.

**Spec 1: Reserve, renew, release, the listing and the hook.**

- **Given** a bare repository as `origin` with no `abcd-reserved/iss-N` branch and P's checkout on a detached HEAD, **when** P runs `abcd reserve iss-N --footprint a.go`, **then** `refs/heads/abcd-reserved/iss-N` points at a parentless empty-tree commit, authored and committed by P's git identity, whose trailers carry the record id, P's username, the footprint, `claimed_at`, an `expires_at` two hours later and `Assisted-by: abcd:<version>`, with P's HEAD, index and `git status --porcelain` unchanged.
- **Given** P holds a live reservation on `iss-N`, **when** Q reserves `iss-N`, **then** Q is refused with the back-off exit, naming P's username and the effective expiry, and the branch is unchanged.
- **Given** Q fetched before P reserved `iss-N`, **when** Q reserves `iss-N`, **then** the bare repository rejects Q's parentless push as non-fast-forward, Q's verb re-reads and refuses naming P, and Q writes no reservation anywhere.
- **Given** P holds a reservation made from one working copy, **when** a fresh agent session in the same copy reserves `iss-N`, **then** it pushes a fast-forward child with a later `expires_at`.
- **Given** the tip of `abcd-reserved/iss-N` is Q's, **when** P runs `abcd reserve iss-N --release`, **then** P is refused naming Q and nothing is pushed.
- **Given** P holds a reservation, **when** P runs `abcd reserve iss-N --release`, **then** a fast-forward tombstone child carrying `released_at`, with `expires_at` equal to it, is pushed and no branch is deleted.
- **Given** a reservation branch whose tip is a tombstone, **when** Q reserves `iss-N`, **then** Q's reservation succeeds as a fast-forward child of the tombstone.
- **Given** P's reservation reached its effective expiry more than five minutes ago by Q's clock, **when** Q reserves `iss-N`, **then** Q's verb judges the expiry itself, pushes a fast-forward child, and names P and the expiry it replaced.
- **Given** P's reservation reached its effective expiry less than five minutes ago by Q's clock, **when** Q reserves `iss-N`, **then** Q is refused naming P.
- **Given** a tip whose `expires_at` is thirty days after its `claimed_at`, **when** any verb judges it, **then** its effective expiry is `claimed_at` plus 24 hours.
- **Given** a request for a lease longer than 24 hours, **when** the verb runs, **then** it refuses before pushing.
- **Given** a tip whose trailers carry instruction-shaped text, an unknown key and a malformed `expires_at`, **when** `abcd peers` lists it, **then** every value is shown as quoted data, the unknown key is ignored, and the reservation is reported as unreadable and held until its commit date plus two hours.
- **Given** an unreadable tip whose commit date plus two hours and five minutes has passed, **when** Q reserves the record, **then** Q takes it over as a fast-forward child and the output names the unreadable commit it replaced.
- **Given** a tip whose message exceeds the size cap, **when** any verb reads it, **then** it is reported as unreadable.
- **Given** P's live reservation on `iss-N` and `iss-N` in `resolved/` on `refs/remotes/origin/main` as last fetched, **when** any verb judges the reservation, **then** it counts as released.
- **Given** P holds reservations on `iss-N` and `iss-M`, **when** Q runs `abcd peers`, **then** Q's `refs/remotes/origin/abcd-reserved/*` holds both and the listing shows each one's holder, footprint, effective expiry and state, with P's live count as two.
- **Given** `user.email` is `1234+octo@users.noreply.github.com`, **when** P reserves, **then** the holder trailer is `octo`.
- **Given** `user.email` is not a no-reply address and `~/.abcd/config.json` names the username `octo`, **when** P reserves, **then** the holder trailer is `octo`.
- **Given** neither `user.email` nor `~/.abcd/config.json` gives a username and the repository's own `.abcd/` configuration names one, **when** P reserves, **then** the verb refuses naming the two user-level sources and pushes nothing.
- **Given** a laptop copy resolving to `octo` through `~/.abcd/config.json` holds a live reservation on `iss-N`, and a cloud copy whose `user.email` is `octo`'s no-reply address, **when** the cloud session reserves `iss-N` without `--take-over`, **then** it is refused with "you are already building iss-N on another computer, until <expiry>" and nothing is pushed.
- **Given** a checkout whose only remote is named `upstream`, **when** the verb runs, **then** it refuses naming `upstream` as the remote found.
- **Given** an `origin` whose path does not exist, **when** the verb runs, **then** it fails naming the cause and says no reservation was made.
- **Given** a bare repository whose `pre-receive` hook rejects `refs/heads/abcd-reserved/*`, standing in for a missing write permission and for a proxy that rejects the push, **when** the verb runs, **then** it fails quoting the rejection and says no reservation was made.
- **Given** this repository's committed pre-push hook, **when** the verb pushes a reserve, a renewal, a takeover and a release, **then** the hook passes each without a preflight receipt and without `--no-verify`.
- **Given** this repository's committed pre-push hook, **when** a push to `refs/heads/abcd-reserved/*` includes a commit whose tree is not the empty tree, **then** the hook refuses it for want of a receipt.
- **Given** this repository's committed pre-push hook, **when** a non-fast-forward push to any branch, a reservation branch included, is attempted, **then** the hook refuses it.
- **Given** no footprint, **when** P runs `abcd reserve iss-N`, **then** the verb refuses and pushes nothing.
- **Given** the reservation commits, every render and the `--json` payloads, **when** they are scanned, **then** none names a host or carries a home path and the outbound policy finds no session URL or tool footer.
- **Given** a footprint path containing shell metacharacters, a colon and an asterisk, **when** P reserves with it, **then** it is stored verbatim as a trailer value and appears in no git or shell argument.
- **Given** a reservation whose message would exceed the size cap, **when** the verb would write it, **then** the verb refuses and pushes nothing.
- **Given** an argument that is not a valid record id, **when** the verb runs, **then** it refuses before forming any ref name.
- **Given** the shipped `implement claim` and the reservation, **when** each judges whether a claim is live, expired or unreadable, **then** both call one shared function and the `implement claim` tests pass unchanged.
- **Given** a cloud clone whose git identity is the person's no-reply address, **when** `scripts/check-attribution.sh` reads its reservation commits, **then** it passes.

**Spec 2: The land fence, the cap, the footprint refusal, the renders, and build and drain.**

- **Given** no reservation on planned intent `itd-X` whose spec names a footprint, **when** P runs `abcd build itd-X`, **then** `abcd-reserved/itd-X` holds P's reservation with the spec's footprint.
- **Given** planned intent `itd-X` whose spec carries no `## Footprint`, **when** P runs `abcd build itd-X`, **then** it is refused at the reservation, naming the missing footprint.
- **Given** P's run holds the reservation on `itd-X`, **when** P runs `abcd build itd-X --release`, **then** a tombstone child is pushed.
- **Given** a drain whose rule takes `iss-N`, **when** `abcd drain` starts its lane, **then** `abcd-reserved/iss-N` holds P's reservation with the issue's `found_at` path as its footprint.
- **Given** Q's last-fetched view shows P's live reservation on planned intent `itd-X`, **when** Q runs `build next`, **then** `itd-X` appears under `excluded`, naming P's reservation, its expiry and the view's age.
- **Given** Q's last-fetched view shows P's live reservation on planned intent `itd-X`, **when** Q runs `build itd-X`, **then** it is refused at the `peers` check naming P.
- **Given** a run holding its reservation, half of whose lease has passed since the tip's `claimed_at`, **when** `implement step` runs, **then** it renews the reservation with a fast-forward child and logs the renewal.
- **Given** a run holding its reservation, less than half of whose lease has passed, **when** `implement step` runs, **then** it pushes nothing for the reservation.
- **Given** a lane at land whose reservation tip is now Q's, **when** `implement step` reaches the arm step, **then** it refuses naming Q and arms nothing.
- **Given** a lane at land whose reservation tip is still P's and a ruleset mirror whose `check_response_timeout_minutes` is 45, **when** `implement step` reaches the arm step, **then** it renews with a lease of at least 45 minutes plus the margin before arming.
- **Given** a lane at land in a repository with no ruleset mirror, **when** `implement step` reaches the arm step, **then** it renews with the default two-hour lease.
- **Given** a lane whose record is terminal on the default branch and whose reservation tip is P's, **when** `implement step` completes the landing, **then** it pushes a tombstone child.
- **Given** a lane whose record is terminal on the default branch and whose reservation tip is Q's, **when** `implement step` completes the landing, **then** it reports Q and leaves the branch as it is.
- **Given** P holds ten live reservations, **when** P reserves an eleventh record, **then** the verb refuses naming the cap and P's live reservations, and pushes nothing.
- **Given** Q's live reservation names `internal/x/a.go` in its footprint, **when** P reserves `iss-M` with a footprint that includes `internal/x/a.go`, **then** P is refused, naming Q's reservation and the shared path.
- **Given** the zero-network harness and a checkout that has fetched the reservations, **when** `abcd <record-id>`, the board, `build <itd-N>` and `build next` run, **then** none makes a network request and each shows the view's age.
- **Given** a checkout that has never fetched the reservations, **when** `abcd <record-id>` renders, **then** it says nothing has been fetched rather than that the item is free.
- **Given** a run whose lane branch is `build/<run>` and a reservation branch `abcd-reserved/<record-id>`, **when** the land stage removes the lane's branch, **then** the reservation branch is untouched.
- **Given** P's laptop copy holds a live reservation on `iss-N`, **when** P's cloud session runs `abcd reserve iss-N --take-over`, **then** it pushes a fast-forward child naming the cloud copy, and the output names the laptop copy it replaced.
- **Given** P's laptop lane at land whose reservation P's cloud copy has taken over, **when** `implement step` reaches the arm step on the laptop, **then** it refuses naming the other copy and arms nothing.

## Decisions

Ruled by the product thinker on 2026-10-03:

1. **Coordination is visible to every participant.** A second person now works on the project and the repository is public, so nothing kept only on one computer (the local tier, a note on the desktop, same-account messaging) is the source of truth for who holds what. Such channels may only speed up notice.
2. **The marker lives on the shared remote and abcd lists it.** As first ruled, it was a ref the forge's website does not show; decision 10 narrows that to a branch, which the website does show. It covers people with write access to the remote; outside contributors are the follow-up [itd-2610031259176838](../drafts/itd-2610031259176838-someone-outside-the-project-can-reserve-a-piece-of-work-too.md).
3. **"A today, B later."** This record is settled and built today; the wider session register, [itd-2609150819440345](../drafts/itd-2609150819440345-which-session-holds-which-worktree-branch-or-record-is-coord.md), stays a draft for later.
4. **A cloud session working in parallel is covered.** In the product thinker's words: "A cloud session ([a hosted session] running on a hosted machine with its own clone) counts as one more participant that shares only the remote; it reserves, sees and honours claims exactly as a second person's machine does."

Agent-proposed on 2026-10-03, following from decision 2; the transport questions (7, 8 and how each act is pushed) are settled by evidence in review 1, and the rest were confirmed at the planning interview:

5. **The refusal surface is before the work.** The reservation is taken when work starts and refuses an item another person holds; `build next` passes over a reserved item and `build <itd-N>` refuses it; the land stage re-checks before arming the merge.
6. **Liveness is expiry plus renewal.** A reservation lapses two hours after its last renewal unless renewed hourly, its expiry clamped to 24 hours; nothing tries to detect that a session has died.
7. **The price is one plain push.** Seconds, with no pull request and no merge-queue pass. Settled by evidence in review 1; the no-workflow part is a hypothesis (decision 21).
8. **The lease lives on the shared remote**, as a reservation branch, not on any one machine. Settled by evidence in review 1 and ruled in decision 10.
9. **There is no stamp.** No record file carries a reservation field, so the two-release schema migration the 2026-09-09 review required does not arise.

Ruled by the product thinker on 2026-10-03, after review 1:

10. **The marker is a branch**, `refs/heads/abcd-claim/<record-id>` as ruled, not a hidden ref, because a hosted cloud session's git proxy accepts pushes of branches only (the 2026-10-03 probe got HTTP 403 on `refs/abcd/*` while branch pushes succeeded, and the host's description of the proxy says it rejects branch deletions and anything but a branch). This narrows decision 2's "hidden" wording rather than deleting it. Accepted costs: The branch shows in the forge's branch list, may show a compare-and-pull-request banner, and needs occasional clean-up. One backend only; no hybrid. The prefix is refined by decision 20.
11. **Build everything today.** The land fence, the footprint handling, the board and `abcd <record-id>` renders and the malformed-reservation handling are all in; no criterion is cut, and the criteria the reviews found defective are fixed instead.

Ruled by the technical facilitator (the person) on 2026-10-03, at the planning interview:

12. **A reservation push is not remote configuration** (pre-pass Q1, invariant 10). `implement step` may renew a reservation by itself. The change that builds this adds one clause to invariant 10 saying a git branch push is not remote configuration; it is listed under Records to touch at ship, and the invariant is not edited now.
13. **Release is derived** (Q3). A reservation stops counting as soon as its record is terminal on the default branch: Readers derive the release, and no post-merge step is needed. The tombstone after the merge is a tidy-up, not the release; expiry stays for abandoned work.
14. **An unreadable tip keeps its grace** (Q4). It is held until its commit date plus two hours, then open to a takeover that names what it replaced, on the precedent of the shipped `implement claim`. The principle "unrecognised input never writes" gives way here, because refusing every write on an unreadable tip would let one malformed push hold a record for ever: No verb deletes a reservation branch today and a cloud session cannot delete one, while a takeover is a fast-forward child that keeps the unreadable commit in the history, so nothing is lost or reinterpreted.
15. **A per-person cap of ten live reservations**, enforced by the verb (Q5).
16. **The footprint is required, and overlap refuses** (Q6). A reservation without a footprint is refused; one whose footprint shares a path with another live reservation is refused, naming the other reservation, and the agent takes other work. This replaces "overlap is reported, not refused".
17. **The holder is the person** (Q20). Any agent or session of the same person may renew or release that person's reservations; only other people are refused. Two of one person's own sessions are kept apart by the in-run `implement claim`, not by this. The opaque session key is therefore dropped from the reservation. Refined by decision 33, which records the copy and refuses the same person's other copy.
18. **A person is their GitHub username** (Q22). It is read from the no-reply address, or from a value set once per computer in the user-level `~/.abcd/config.json`, never the repository; a laptop and a cloud session then count as one person. A session with neither source is refused.
19. **Two specs under this one intent, both built today** (Q7). Spec 1 is reserve, renew, release, the listing and the pre-push hook exemption; spec 2 is the land fence, the cap, the footprint refusal, the renders and the build and drain integration. The intent ships when both close.
20. **The verbs and the word** (reworked from the person's own idea). `abcd build <itd-N>` reserves the intent when it starts, and `abcd build <itd-N> --release` gives it back; `abcd drain` reserves each issue it takes; a small top-level `abcd reserve <record-id>` and `abcd reserve <record-id> --release` cover hand work; `abcd peers` lists every reservation. The word a person reads is "reservation" everywhere; the record keeps "claim" as the internal name. The branch prefix is `abcd-reserved/`, the facilitator's preference, so the forge's branch list reads in the same plain word; the record holds no reason for `abcd-claim/` beyond its first use in decision 10 and the decision log's line of the same day, which this refines.

Decided without a question at the planning interview, each the one defensible answer on the record:

21. **The no-workflow claim is a hypothesis** (Q2) until a real push proves it, and sits in the scope conditions; the 5,000-branch limit cites the forge's repository-limits documentation.
22. **This record refines the shipped in-run claim** (Q8, Q11), written as `refines: [itd-2609221656373558]`: The liveness judgement moves into one function both use, and the in-run claim's store and lease shape stay.
23. **The state-of-the-art path is build-native** (Q9): The primitive is git itself, a compare-and-swap on a branch, so nothing is adopted or wrapped.
24. **The register and the outside-contributor follow-up stay separate** (Q12, Q13), as already ruled; they will read reservations through the same liveness function.
25. **The mailbox draft is untouched** (Q14). itd-2609151838312703 stays as it is; this record's decision 1 governs any local channel, which may speed up notice and never decide who holds what.
26. **The resolve-time refusal stays separate** (Q15); the two share the "as of last fetch" reader and its age line at build time.
27. **This record refines `build next`** (Q16), [itd-2609211116005482](itd-2609211116005482-abcd-build-next-picks-the-readiest-planned-intent-itself-wri.md): The pick's exclusions gain "held by another person's reservation", with the view's age.
28. **Tombstone deletion lives in itd-118** (Q17); both records stay.
29. **The fence's lease comes from the ruleset mirror** (Q18): `check_response_timeout_minutes` (45) in `.abcd/work/rulesets/main-protection.json`; with no mirror, the default lease.
30. **The `.git` rule covers files abcd invents** (Q19). "Read it, never write it" forbids abcd's own files under `.git/`; git's own refs and objects, written by `git fetch` and `git commit-tree`, are outside it.
31. **Only `origin`** (Q21). Any other remote layout is refused, naming the remotes found.
32. **No username, no reservation** (Q22, with decision 18). A session whose username resolves from neither source is refused.

Ruled by the product thinker on 2026-10-03, during the criteria walk (Product Q15):

33. **A reservation records the person and the copy; another copy needs `--take-over`.** This refines decision 17. A reservation records the person and the working copy it was made from, as an opaque working-copy key, never a path or a host. The same copy continues and renews freely, so a fresh agent in the same folder carries on. Another copy of the same person (a second computer or a cloud session) is refused while the reservation is live, with the message "you are already building <id> on another computer, until <expiry>". To continue elsewhere on purpose, the person passes `--take-over` (on `abcd reserve` and `abcd build`), which pushes a child naming the new copy, and the old copy's land fence then refuses its merge. The person remains the unit for the cap, the per-person count and the username identity. Criteria C4 and C20 are revised to match, and C52 and C53 added.

Further decisions of 2026-10-03, at the planning interview:

34. **The press release is confirmed as written** (the product thinker). The take-over detail stays in What's In Scope and the Mechanism.
35. **The Mechanism is confirmed as written** (the product thinker); its sentence on the person's own other copy was added with decision 33.
36. **The Scope Conditions are confirmed as written** (the product thinker).
37. **Impact is additive** (the product thinker). `abcd intent plan --impact additive` stamps it; it is not hand-written.
38. **Two specs, through the verbs that exist** (decided without a question). `intent plan` mints spec 1, whose `## Steps` list the spec-1 work first and the spec-2 work after; when spec 1's steps land, `spec close <spc-N> --remainder <slug>` mints spec 2, carrying the unlanded steps; the intent ships when spec 2 closes. This realises decision 19.

## Open Questions

None open.

## Prior Art

- [iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md): The source record, with the 2026-09-01 collisions and the original stamp-and-push remedy.
- [itd-2609221656373558](../shipped/itd-2609221656373558-two-orchestrator-sessions-in-one-user-account-share-an.md) and `internal/core/implement/claim.go`: The shipped in-run claim; exclusive create, a two-hour lease bounded by a 24-hour `MaxLease`, a back-off exit and a grace for an unparseable claim. This record refines it.
- [itd-2609150819440345](../drafts/itd-2609150819440345-which-session-holds-which-worktree-branch-or-record-is-coord.md), decision 4: The product thinker's choice of transport, recorded on the register before this record was rewritten around it.
- The multi-agent coordination synthesis of 2026-10-03 ([`2026-10-03-multi-agent-coordination-sota-and-run-lessons.md`](../../research/notes/2026-10-03-multi-agent-coordination-sota-and-run-lessons.md)): The run lessons and the state of the art. **Path: build-native.** The primitive is git itself (a create-only compare-and-swap on a ref, as in a sixteen-agent compiler run coordinated through git push conflicts), with leases and fencing re-checked at merge, per-reservation path footprints, and every peer channel treated as untrusted data.
- Reviews 1 and 2 of 2026-10-03 (design and feasibility; record discipline), summarised in the synthesis's Review evidence section: The cloud probe, the scratch bare-repository experiments behind the plain-push compare-and-swap, and the hook, fence, clock, duplication and record-placement findings.
- The 2026-09-09 split, in the decision log: The five review findings recorded above.
- `.githooks/pre-push`, `internal/core/guard/defaults/guard.json` and `internal/core/implement/loop/land.go`: The hook a reservation push meets, the registry that rules out lease flags, and the land stage the fence sits in.
- [Brief invariant 7](../../brief/02-constraints/03-invariants.md): Implicit operations never touch the network; every reservation push and fetch belongs to a verb that says so.
- [Loud staging](../../principles/loud-staging.md): Its letter is about unwired code announcing itself; this record applies its spirit, an explicit refusal and never a half-working stub, to a reservation that cannot reach the remote.
- [itd-22](../drafts/itd-22-harness-portability.md): The host-profile seam; the reason the host's session registry is never the reservation's home.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: A second person joined today, and agents on two computers plus a cloud session now work on the project at once. We expect no item to be worked twice in the first month after it ships; one duplicate pull request or one issue fixed twice by two people would show it wrong.
