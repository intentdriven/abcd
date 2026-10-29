---
id: itd-2609091034175565
slug: nothing-tells-an-agent-that-a-record-it
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
related_intents: [itd-2609091416295622, itd-2609091416304128, itd-2609091014076309, itd-2609150819440345, itd-2610031259176838, itd-118, itd-2610170634394993]
related_adrs: [adr-2609091248200336, adr-2610031352212697]
supersedes: [itd-33]
impact: additive
---

# Before an agent starts on an item, it reserves it on the shared remote, and everyone who works on the project sees the reservation

Typed links: `refines` [itd-2609221656373558](itd-2609221656373558-two-orchestrator-sessions-in-one-user.md) (the shipped in-run claim, whose lease shape, two-hour default, back-off refusal and liveness judgement this record carries to the shared remote; the judgement moves into one function both use) and [itd-2609211116005482](itd-2609211116005482-abcd-build-next-picks-the-readiest.md) (the pick's exclusions gain "held by another person's reservation" and the view's age); `supersedes` [itd-33](../superseded/itd-33-agent-communication-infrastructure.md) (agent coordination through a local `active-work.json`, which decision 1 rules out); `related_issues` [iss-2609020716570699](../../../work/issues/resolved/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it.md) (the duplicated re-fixes and pull requests this record exists to prevent); `related_intents` [itd-2609091416295622](itd-2609091416295622-a-session-sees-the-records-its-sibling.md) (the read-only sibling-worktree listing; prose cross-reference to the 2026-09-09 ruling, which gave it its own record), [itd-2609091416304128](../drafts/itd-2609091416304128-capture-resolve-refuses-a-record-the.md) (the resolve-time refusal; prose cross-reference to the same ruling), [itd-2609091014076309](../planned/itd-2609091014076309-session-and-agent-worktrees-live-in-a.md) (the machine-scoped worktree store), [itd-2609150819440345](../superseded/itd-2609150819440345-which-session-holds-which-worktree.md) (the wider register of running sessions, kept for later; its decision 4 is this record's transport), [itd-2610031259176838](../drafts/itd-2610031259176838-someone-outside-the-project-can-reserve.md) (outside contributors, who cannot push a reservation branch), [itd-118](../superseded/itd-118-merged-work-leaves-no-residue-abcd.md) (the home of tombstone-branch deletion); `related_adrs` [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user.md) (a tool never writes in space it was not handed) and [adr-2610031352212697](../../decisions/adrs/2610031352212697-claims-are-branches-on-the-shared-remote.md) (reservations are branches on the shared remote, nothing kept on one computer is the register, and reservation text is data; accepted). Prose cross-reference, not a typed link, because no schema field carries the relation ([iss-2609091256264547](../../../work/issues/open/iss-2609091256264547-three-of-the-four-mandated-typed.md)): [iss-2608220750029993](../../../work/issues/open/iss-2608220750029993-session-presence-detection-for-shared.md), session presence, which this record answers for records and not for sessions.

**On words.** Every surface a person reads says **reservation**. Inside the record and the code a reservation is a **claim**, the name the shipped in-run `implement claim` already uses; the two words name one thing.

## Press Release

> **Two people's agents never start the same piece of work.** When an agent starts on an item, abcd reserves it for the person the agent works for. The reservation is a small branch on the project's shared copy that nobody ever merges: Everyone who works on the project sees it, whichever computer they work on, and so does an agent working from a cloud session. If someone else already holds the item, or the files the work will touch, abcd says who and until when, and the agent takes something else. A reservation lasts two hours and is renewed every hour while the work goes on, so one left behind by a closed laptop frees itself. It stops counting the moment the work is merged. Starting a build reserves the item, and a person working by hand reserves with one short command.
>
> "There are two of us now, on two computers, each running agents, and sometimes a cloud session as well," said Maya, an autonomous-development practitioner. "My agents could only see what my own machine knew, so they would happily start an item my colleague's agents were halfway through. Now they see the reservation before they begin, and the one that loses finds out in seconds, not after a night's work. Moving a piece of work from one of my copies to another is one flag."

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

The record began as the remedy of [iss-2609020716570699](../../../work/issues/resolved/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it.md): On one night of the 2026-09-01 run a peer session re-fixed two issues a paused branch had already fixed, and two of its pull requests duplicated merged work; the resolution gate refused the push, the first moment anything said no. The recorded remedy was a `claimed_by` stamp on the record, pushed alone, plus a duplicate guard and a resolve-time refusal. On 2026-09-09 two local near-collisions (a record nearly minted twice, a worktree nearly created over a peer's) widened the draft to cover sessions on one machine.

Two adversarial reviews of the widened draft then found five things, and the ruling of 2026-09-09 in the decision log split the record in three: The listing ([itd-2609091416295622](itd-2609091416295622-a-session-sees-the-records-its-sibling.md), shipped) and the resolve-time refusal ([itd-2609091416304128](../drafts/itd-2609091416304128-capture-resolve-refuses-a-record-the.md)) went their own way, and this record kept the claim, marked not ready. The findings, and what each becomes under this design:

- **A stamp on an issue record is invisible to an older reader.** The issue schema is a closed allow-list (`internal/core/issueschema/issueschema.go`, mirrored by `record_schema` in `internal/core/lint/schema.go`), so a stamp needed a two-release migration. This design writes no field into any record, so the migration is gone with the stamp.
- **The refusals fired after the work.** Here the reservation is taken when a build starts, `build next` passes over a reserved item, and the land stage re-checks.
- **No staleness threshold was safe in both directions.** Here nothing detects liveness: A lease expires unless renewed, bounded by the lease in one direction and by the hourly renewal in the other, and merged work releases its reservation without any step.
- **The pushed half cost a merge-queue pass per claim**, fifteen to sixteen minutes measured on this repository. A reservation branch opens no pull request and enters no queue.
- **Every collision on record was covered without a claim.** That held while every session shared one machine. A second person or a cloud session has no sibling worktree to list, and the resolve-time refusal fires only after their work has merged.

Three homes were weighed for a lease on one machine: The shared `.git`, the machine-scoped `~/.abcd.noindex/` lane keyed on the root commit, and the host's session registry (a source through hooks, never a surface, per [itd-22](../drafts/itd-22-harness-portability.md)). All three are one computer's knowledge, which decision 1 rules out as the source of truth.

### The reviews and the planning interview (2026-10-03)

Two adversarial reviews of this rewrite followed, design and feasibility, then record discipline; both returned NEEDS-WORK, and their evidence is in the synthesis's Review evidence section. Review 1 found the primitive right and the transport wrong: A hosted cloud session's git proxy pushes branches only (HTTP 403 on a ref outside `refs/heads/`); abcd's hazard registry and the cloud proxy rule out lease flags and deletions, while a plain push is already the compare-and-swap; the committed pre-push hook refused every claim push; the merge queue merges after the fence; the claimer sets every time; and the lease arithmetic must have one home. Review 2 found the grounds unreadable to the second person, the criteria compound, and logged rulings reversed only inside the draft; the synthesis was promoted, the criteria split, and the rulings logged. The planning interview then answered the pre-pass's questions: Decisions 12 to 32 below.

## What's In Scope

**How the two specs are realised** (decision 38). `abcd intent plan` mints spec 1, whose `## Steps` list the spec-1 work below first and the spec-2 work after it. When spec 1's steps have landed, `abcd spec close <spc-N> --remainder <slug>` mints spec 2, carrying the unlanded steps. The intent ships when spec 2 closes.

**Spec 1: Reserve, renew, release, the listing and the hook.**

- **A reservation is a branch nobody merges**, `refs/heads/abcd-reserved/<record-id>` on `origin`, holding a chain of commits whose every tree is the empty tree (`4b825dc642cb6eb9a060e54bf8d69288fbee4904`). abcd writes each commit with `git commit-tree`, so the caller's HEAD, index and working tree are never touched and a detached HEAD makes no difference. Each commit's message carries trailers: The record id; the holder, the person's GitHub username; the copy, an opaque working-copy key naming the checkout the reservation was made from (never a path or a host); the footprint; `claimed_at` (when this commit was written); `expires_at`; `released_at` on a tombstone; and `Assisted-by: abcd:<version>`, which abcd writes itself because `commit-tree` runs no `prepare-commit-msg` hook. Author and committer are the person's git identity. No record file gains a field.
- **The holder is a person, named by their GitHub username.** abcd reads it from a no-reply address in `user.email` (`<id>+<username>@users.noreply.github.com` or `<username>@users.noreply.github.com`), or else from a value set once per computer in the user-level `~/.abcd.noindex/config.json`, never from the repository. With neither, every write is refused, naming both sources. A laptop and a cloud session of one person resolve to one username and count as one person.
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
- **The verbs.** `abcd reserve <record-id> --footprint <path>…` and `abcd reserve <record-id> --release` cover the work no build or drain starts, each a push by its documented meaning; `abcd reserve` is an agent's verb (decision 39), listed with the verbs agents and hosts call. `abcd peers` fetches `+refs/heads/abcd-reserved/*:refs/remotes/origin/abcd-reserved/*` with pruning and lists every reservation with its holder, footprint, effective expiry and state (live, expired, released, unreadable), and each person's count of live reservations.
- **Only `origin`.** A checkout whose remotes do not include one named `origin` is refused, naming the remotes found.
- **Every failure refuses loudly and writes no local reservation**: No remote; no write permission; offline; a remote or proxy that rejects the push. Each names its cause and says that no reservation was made.
- **The pre-push hook exempts reservation chains, narrowly.** `.githooks/pre-push` passes an update to `refs/heads/abcd-reserved/*` without a preflight receipt only when every commit in the pushed range (every commit, on a create) has the empty tree; anything else under that prefix is gated as today. The non-fast-forward refusal stays and covers reservation branches too. No reservation push skips the hooks.
- **Public safety.** A reservation names no host and no home path; the message passes the outbound policy (no session URL, no tool footer). A reservation is parsed by its trailer keys and rendered as quoted data, never handed to an agent as instructions; an unknown key is ignored. Footprint paths are trailer values only and never reach a shell or a refspec. The branch name is built only from a record id that passes the record-id grammar. The message is size-capped, and the verb refuses to write a larger one.

**Spec 2: The land fence, the cap, the footprint refusal, the renders, and build and drain.**

- **`abcd build <itd-N>` reserves the intent when it starts**, with the footprint its spec's `## Footprint` names, and `abcd build <itd-N> --release` gives it back. **`abcd drain` reserves each issue it takes**, with the issue's `found_at` path as its footprint. A run with no footprint to give is refused at the reservation, naming what is missing. These pushes join each verb's documented meaning.
- **Renewal during a run.** `implement step` renews the run's reservation whenever half its lease has passed (every hour for the two-hour default); its documented meaning gains the reservation read and write.
- **The land fence.** Inside `implement step`: Immediately before arming the merge (or leaving the pull request for a person), the land stage fetches the reservation branch and refuses unless the tip still names this person and this copy, naming the person or the other copy that holds it now; when it is, the stage renews it with a lease of at least the merge queue's `check_response_timeout_minutes` from `.abcd/work/rulesets/main-protection.json` (45 here) plus a margin, or a day where no mirror exists (decision 40), and keeps renewing at half-lease while the merge waits. Once the record is terminal on the default branch the reservation has stopped counting; a later step tidies up with a tombstone, and leaves and reports a tip that is no longer this person's.
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

- **The wider session register**, saying which sessions run and what each holds: [itd-2609150819440345](../superseded/itd-2609150819440345-which-session-holds-which-worktree.md), kept as a draft for later.
- **Outside contributors**, who cannot push a branch to the remote: [itd-2610031259176838](../drafts/itd-2610031259176838-someone-outside-the-project-can-reserve.md).
- **The resolve-time refusal** of a record already terminal on the default branch: [itd-2609091416304128](../drafts/itd-2609091416304128-capture-resolve-refuses-a-record-the.md).
- **Deleting reservation branches.** Tombstones are deleted by [itd-118](../superseded/itd-118-merged-work-leaves-no-residue-abcd.md)'s sweep from an ordinary machine, since the cloud proxy rejects deletions.
- **Keeping one person's sessions in one copy apart.** Any session in the copy that made a reservation may renew or release it; sessions sharing one run are kept apart by the in-run `implement claim`, unchanged.
- **Judging that two records describe one observation**: [itd-87](../drafts/itd-87-recurrence-escalation-in-capture.md).
- **Moving, listing or reclaiming worktrees**: [itd-2609091014076309](../planned/itd-2609091014076309-session-and-agent-worktrees-live-in-a.md).
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
- **Given** `user.email` is not a no-reply address and `~/.abcd.noindex/config.json` names the username `octo`, **when** P reserves, **then** the holder trailer is `octo`.
- **Given** neither `user.email` nor `~/.abcd.noindex/config.json` gives a username and the repository's own `.abcd/` configuration names one, **when** P reserves, **then** the verb refuses naming the two user-level sources and pushes nothing.
- **Given** a laptop copy resolving to `octo` through `~/.abcd.noindex/config.json` holds a live reservation on `iss-N`, and a cloud copy whose `user.email` is `octo`'s no-reply address, **when** the cloud session reserves `iss-N` without `--take-over`, **then** it is refused with "you are already building iss-N on another computer, until <expiry>" and nothing is pushed.
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
- **Given** a lane at land in a repository with no ruleset mirror, **when** `implement step` reaches the arm step, **then** it renews with a day's lease, `lease.Max` (revised by decision 40 on 2026-10-15; it first read "with the default two-hour lease").
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
2. **The marker lives on the shared remote and abcd lists it.** As first ruled, it was a ref the forge's website does not show; decision 10 narrows that to a branch, which the website does show. It covers people with write access to the remote; outside contributors are the follow-up [itd-2610031259176838](../drafts/itd-2610031259176838-someone-outside-the-project-can-reserve.md).
3. **"A today, B later."** This record is settled and built today; the wider session register, [itd-2609150819440345](../superseded/itd-2609150819440345-which-session-holds-which-worktree.md), stays a draft for later.
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
20. **The verbs and the word** (reworked from the person's own idea). `abcd build <itd-N>` reserves the intent when it starts, and `abcd build <itd-N> --release` gives it back; `abcd drain` reserves each issue it takes; a small top-level `abcd reserve <record-id>` and `abcd reserve <record-id> --release` cover hand work (refined by decision 39: an agent's verb); `abcd peers` lists every reservation. The word a person reads is "reservation" everywhere; the record keeps "claim" as the internal name. The branch prefix is `abcd-reserved/`, the facilitator's preference, so the forge's branch list reads in the same plain word; the record holds no reason for `abcd-claim/` beyond its first use in decision 10 and the decision log's line of the same day, which this refines.

Decided without a question at the planning interview, each the one defensible answer on the record:

21. **The no-workflow claim is a hypothesis** (Q2) until a real push proves it, and sits in the scope conditions; the 5,000-branch limit cites the forge's repository-limits documentation.
22. **This record refines the shipped in-run claim** (Q8, Q11), written as `refines: [itd-2609221656373558]`: The liveness judgement moves into one function both use, and the in-run claim's store and lease shape stay.
23. **The state-of-the-art path is build-native** (Q9): The primitive is git itself, a compare-and-swap on a branch, so nothing is adopted or wrapped.
24. **The register and the outside-contributor follow-up stay separate** (Q12, Q13), as already ruled; they will read reservations through the same liveness function.
25. **The mailbox draft is untouched** (Q14). itd-2609151838312703 stays as it is; this record's decision 1 governs any local channel, which may speed up notice and never decide who holds what.
26. **The resolve-time refusal stays separate** (Q15); the two share the "as of last fetch" reader and its age line at build time.
27. **This record refines `build next`** (Q16), [itd-2609211116005482](itd-2609211116005482-abcd-build-next-picks-the-readiest.md): The pick's exclusions gain "held by another person's reservation", with the view's age.
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
39. **`abcd reserve` is an agent's verb** (the product thinker, 2026-10-12: "I suggest /abcd:reserve becomes an agent-only verb"). This refines decision 20, which had it cover hand work: the verb is listed in the agents-and-hosts block and hidden from a person's `/abcd:` menu, it stays runnable from the CLI, and a person sees reservations through `abcd peers` and the record and board renders.

Ruled by the product thinker on 2026-10-15, at about 20:15 BST (iss-2610152020234609):

40. **A pull request waiting on a person holds the reservation for a day, mirror or not.** This amends decision 29, whose "with no mirror, the default lease" no longer holds: the land fence renews for `lease.Max` whether or not a ruleset mirror exists, because a two-hour lapse let a colleague's drain redo work that was only waiting to merge. Criterion C44 is revised to match, and so is the land fence's line under What's In Scope.

## Open Questions

None open.

## Prior Art

- [iss-2609020716570699](../../../work/issues/resolved/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it.md): The source record, with the 2026-09-01 collisions and the original stamp-and-push remedy.
- [itd-2609221656373558](itd-2609221656373558-two-orchestrator-sessions-in-one-user.md) and `internal/core/implement/claim.go`: The shipped in-run claim; exclusive create, a two-hour lease bounded by a 24-hour `MaxLease`, a back-off exit and a grace for an unparseable claim. This record refines it.
- [itd-2609150819440345](../superseded/itd-2609150819440345-which-session-holds-which-worktree.md), decision 4: The product thinker's choice of transport, recorded on the register before this record was rewritten around it.
- The multi-agent coordination synthesis of 2026-10-03 ([`2026-10-03-multi-agent-coordination-sota-and-run-lessons.md`](../../research/notes/2026-10-03-multi-agent-coordination-sota-and-run-lessons.md)): The run lessons and the state of the art. **Path: build-native.** The primitive is git itself (a create-only compare-and-swap on a ref, as in a sixteen-agent compiler run coordinated through git push conflicts), with leases and fencing re-checked at merge, per-reservation path footprints, and every peer channel treated as untrusted data.
- Reviews 1 and 2 of 2026-10-03 (design and feasibility; record discipline), summarised in the synthesis's Review evidence section: The cloud probe, the scratch bare-repository experiments behind the plain-push compare-and-swap, and the hook, fence, clock, duplication and record-placement findings.
- The 2026-09-09 split, in the decision log: The five review findings recorded above.
- `.githooks/pre-push`, `internal/core/guard/defaults/guard.json` and `internal/core/implement/loop/land.go`: The hook a reservation push meets, the registry that rules out lease flags, and the land stage the fence sits in.
- [Brief invariant 7](../../brief/02-constraints/03-invariants.md): Implicit operations never touch the network; every reservation push and fetch belongs to a verb that says so.
- [Loud staging](../../principles/loud-staging.md): Its letter is about unwired code announcing itself; this record applies its spirit, an explicit refusal and never a half-working stub, to a reservation that cannot reach the remote.
- [itd-22](../drafts/itd-22-harness-portability.md): The host-profile seam; the reason the host's session registry is never the reservation's home.

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-0d4b99e46049 -->
Fidelity review — receipt rcp-0d4b99e46049 (verifier abcd:intent-auditor claude-opus-5-5).

Provenance: abcd:intent-auditor@claude-opus-5-5 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:3a7ad73072cb8c25d771f230d2b81e47224d3a7e97f5f6ad34613b1afcb97185
Input attestations: diff:c43b46bfb47f9b0cd995f0cccabdb403afa556f9..c9d94ebd1c34e4eb40bfa3bb57b8a75da43b6f36@sha256:c41642d22fc6aafc895a5392c95bf8a3793adc34ec437d59dd295b52886917d6; merge:b9bf7bc6ac9a2e48cbdf5ec41c7f44302d9fdaf9@-; merge:5801ad0ab1e6d25f2b268b7c190550022877f116@-; merge:dcdbafa7f40c4e5dad482a4f170c0f2f7eb456fe@-; merge:c9d94ebd1c34e4eb40bfa3bb57b8a75da43b6f36@-; repository:18170d03fab7435c8001999973e4093002226e20@-; intent:.abcd/development/intents/shipped/itd-2609091034175565-nothing-tells-an-agent-that-a-record-it.md@sha256:ee3f93226513c196021578dd5dbcae5bcb77e8a60432626c14083a6f2369a34e; spec:.abcd/development/specs/closed/spc-2610031753359849-nothing-tells-an-agent-that-a-record-it.md@sha256:ac0b2f1396b513a60c335651156965c256298ff30b73c8f13fe6d829f5b287d0; request:.abcd/.work.local/reviews/rcp-0d4b99e46049.request.md@sha256:e6e63c32458c8bc96f53ba8dc2d2bf23ef8a996c14c25db1ebc7b7ce7f3fa757;

Acceptance rollup: MET 52 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: TestAReservationIsAParentlessEmptyTreeCommitAndLeavesTheCheckoutAlone reserves from a detached HEAD with a dirty and staged tree against a bare origin and asserts a parentless commit on the empty tree at refs/heads/abcd-reserved/iss-5, authored and committed as P's git identity, whose trailers carry iss-5, octo, a.go, claimed 14:00Z, expiring 16:00Z and Assisted-by: abcd:< version>, with HEAD, index and porcelain status unchanged and no local reservation branch.
  evidence: internal/core/claim/remote_test.go:142 — "p.git("checkout", "-q", "--detach", "HEAD")"
  evidence: internal/core/claim/remote_test.go:163 — "tree != "4b825dc642cb6eb9a060e54bf8d69288fbee4904""
  evidence: internal/core/claim/remote_test.go:171 — ""Assisted-by: abcd:dev""
  evidence: internal/core/claim/remote_test.go:176 — "h != head || i != index || s != status"
- ac-2 — MET: Q's reserve over P's live reservation is refused as ErrBackOff naming octo and the effective expiry, origin's tip is unmoved, and even --take-over cannot move another person's live reservation; the CLI maps the back-off to exit 3.
  evidence: internal/core/claim/remote_test.go:195 — "want := "iss-5 is reserved by octo until 2026-10-03T16:00:00Z; back off and take other work""
  evidence: internal/core/claim/remote_test.go:199 — "if got := w.remoteTip("iss-5"); got != tip {"
  evidence: internal/surface/cli/reserve.go:292 — "return &exitError{Code: 3, Msg: msg}"
- ac-3 — MET: Both shapes of the race are driven (git's client rejecting the advertised tip, and origin's own compare-and-swap refusing a push that landed while Q's was received): lostRace re-fetches, re-judges and refuses with the back-off naming octo, origin's tip stays P's and Q holds no local reservation ref.
  evidence: internal/core/claim/remote_test.go:254 — "want := "iss-5 was reserved by octo while this push was on its way, until 2026-10-03T16:00:00Z; back off and take other work""
  evidence: internal/core/claim/remote_test.go:266 — "if local := q.git("for-each-ref", "refs/heads/abcd-reserved/"); local != "" {"
  evidence: internal/core/claim/remote.go:298 — "return backOff("%s was reserved by %s while this push was on its way, until %s; back off and take other work","
- ac-4 — MET: A second reserve from the same checkout, which shares its copy key, 30 minutes later is ActRenewed: a child of the first commit whose Abcd-Expires-At moves from 16:00Z to 16:30Z.
  evidence: internal/core/claim/remote_test.go:283 — "if err != nil || res.Act != ActRenewed {"
  evidence: internal/core/claim/remote_test.go:287 — "if w.parentOf(tip) != first.Commit || !res.ExpiresAt.After(first.ExpiresAt) ||"
  evidence: internal/core/claim/remote_test.go:288 — "Abcd-Expires-At: 2026-10-03T16:30:00Z"
- ac-5 — MET: P's release over Q's tip is refused naming quinn and saying nothing was pushed, and origin's tip is unmoved.
  evidence: internal/core/claim/remote_test.go:325 — "err.Error() != "iss-5 is reserved by quinn, not by you; nothing was pushed""
  evidence: internal/core/claim/remote_test.go:328 — "if w.remoteTip("iss-5") != tip {"
  evidence: internal/core/claim/remote.go:436 — "fmt.Sprintf("%s is reserved by %s, not by you; %s", p.id, j.Holder, nothingPushed))"
- ac-6 — MET: P's release pushes a child of the reservation with the subject 'abcd reservation released' and Abcd-Expires-At equal to Abcd-Released-At (14:20Z); the branch remains (the committed hook refuses any deletion under the prefix), and a second release reports already_released and pushes nothing.
  evidence: internal/core/claim/remote_test.go:349 — "if tip == "" || w.parentOf(tip) != first.Commit || !strings.HasPrefix(msg, "abcd reservation released: iss-5") ||"
  evidence: internal/core/claim/remote_test.go:350 — "Abcd-Released-At: 2026-10-03T14:20:00Z"
  evidence: .githooks/pre-push:432 — "a branch under $row_prefix ends by a push, never a deletion"
- ac-7 — MET: After P's tombstone, Q's reserve is ActReserved as a child of the tombstone commit (the released row of the push table).
  evidence: internal/core/claim/remote_test.go:372 — "if err != nil || res.Act != ActReserved || w.parentOf(w.remoteTip("iss-5")) != rel.Commit {"
  evidence: internal/core/claim/remote.go:372 — "return res, m, j.Commit, nil"
- ac-8 — MET: At P's effective expiry plus the five-minute margin by Q's injected clock, Q's reserve is ActTakenOver as a child of P's tip, Replaced naming octo and the 16:00Z expiry; the text render prints the replaced holder and the lapse time.
  evidence: internal/core/claim/remote_test.go:386 — "w.set(t0.Add(2*time.Hour + TakeoverMargin))"
  evidence: internal/core/claim/remote_test.go:388 — "res.Replaced.Holder != "octo" ||"
  evidence: internal/surface/cli/reserve.go:365 — "replaced: %s's reservation, which lapsed at %s"
- ac-9 — MET: One second inside the margin Q's reserve is refused with the back-off naming octo, the lapse and the retry time, and the branch is unmoved.
  evidence: internal/core/claim/remote_test.go:406 — "w.set(t0.Add(2*time.Hour + TakeoverMargin - time.Second))"
  evidence: internal/core/claim/remote_test.go:408 — "want := "iss-5 is reserved by octo, lapsed at 2026-10-03T16:00:00Z, under five minutes ago by this computer's clock; retry after 2026-10-03T16:05:00Z""
- ac-10 — MET: lease.Judge caps a readable stamp at min(expires_at, min(claimed_at, now) + 24h), counting from the first sighting for one dated more than five minutes ahead; the test holds a thirty-day expires_at live until claimed_at plus a day and lapsed at it, and the reservation's judge calls the same function.
  evidence: internal/core/lease/lease.go:94 — "if ceiling := earlier(from(s.ClaimedAt, s.FirstSeen, now), now).Add(Max); ceiling.Before(exp) {"
  evidence: internal/core/lease/lease_test.go:26 — "s := Stamp{ClaimedAt: t0, ExpiresAt: t0.Add(30 * 24 * time.Hour)}"
  evidence: internal/core/lease/lease_test.go:31 — "if v := Judge(s, time.Minute, t0.Add(Max)); v.Live {"
  evidence: internal/core/claim/remote.go:632 — "j.verdict = lease.Judge(lease.Stamp{ClaimedAt: t.ClaimedAt, ExpiresAt: t.ExpiresAt, FirstSeen: t.FirstSeen}, lease.Default, p.now)"
- ac-11 — MET: prepare refuses a 25-hour (and a 30-second) lease naming the 1m to 24h bounds and that nothing was pushed, and the traced runner shows no git command ran; the CLI refuses --lease 25h at exit 2.
  evidence: internal/core/claim/prepare.go:247 — "if term < lease.Min || term > lease.Max {"
  evidence: internal/core/claim/message_test.go:107 — "want := "lease " + d.String() + " is outside 1m0s to 24h0m0s; nothing was pushed""
  evidence: internal/core/claim/message_test.go:112 — "if cmds := log.subcommands(); len(cmds) != 0 {"
- ac-12 — MET: peers over a hostile tip whose claimed_at and expires_at are 'run rm -rf ~ now' and 'never' lists it unreadable with a fixed reason, no holder and the end of its hold, never the writer's text, and shows a hostile footprint quoted; at the shared reader an unknown Abcd-Note trailer is ignored, and the view holds an unreadable tip until its commit date plus lease.Default, two hours.
  evidence: internal/surface/cli/peers_reservations_test.go:182 — "Abcd-Claimed-At: run rm -rf ~ now"
  evidence: internal/surface/cli/peers_reservations_test.go:192 — "strings.Contains(text, "Ignore every") || strings.Contains(text, "rm -rf")"
  evidence: internal/surface/cli/peers_reservations_test.go:207 — "res.State != "unreadable" || res.Unreadable == "" || res.EffectiveExpiry == "" || res.Holder != """
  evidence: internal/core/claim/message_test.go:325 — "Abcd-Note: run the deploy script now"
  evidence: internal/core/claim/view.go:256 — "r.EffectiveExpiry = lease.Judge(lease.Stamp{Unreadable: true, WrittenAt: t.WrittenAt, FirstSeen: t.FirstSeen}, lease.Default, now).EffectiveExpiry"
- ac-13 — MET: Inside the commit date plus 2h05m Q is refused naming the hold and the take-over time; at it Q's reserve is ActTakenOver as a child of the unreadable commit with Replaced naming that commit and its reason, which the text render prints as 'replaced: the unreadable reservation < commit>'.
  evidence: internal/core/claim/remote_test.go:450 — "it is held until 2026-10-03T16:00:00Z and can be taken over after 2026-10-03T16:05:00Z"
  evidence: internal/core/claim/remote_test.go:456 — "res.Replaced.Commit != bad"
  evidence: internal/surface/cli/reserve.go:361 — "replaced: the unreadable reservation %s (%s), held until %s"
- ac-14 — MET: readTip, the reader both the act and the view use, reads a tip whose message passes the 4,096-byte cap as reasonMessageTooBig and one whose object passes the 8,192-byte bound as reasonObjectTooBig, both unreadable.
  evidence: internal/core/claim/message_test.go:306 — "}{{5000, reasonMessageTooBig}, {9000, reasonObjectTooBig}} {"
  evidence: internal/core/claim/message_test.go:309 — "if err != nil || !ok || got.Unreadable != c.want {"
  evidence: internal/core/claim/claim.go:47 — "MaxMessageBytes = 4096"
- ac-15 — MET: With iss-5 in resolved/ (and iss-6 in wontfix/) on origin's main, judge marks the tip derived and the view lists both released by the default branch; a renewal over it is NotHeld released, and the cap and the overlap count only live reservations. As delivered Q's reserve then went on as the tip's child; since iss-2610160957404827 it backs off a finished record instead, still treating P's reservation as released, not held.
  evidence: internal/core/claim/remote_test.go:500 — "if r.State != StateReleased || r.ReleasedBy != ReleasedByDefaultBranch {"
  evidence: internal/core/claim/remote.go:622 — "j.derived = p.derivedGap == "" && terminal(recs[p.id], p.id)"
  evidence: internal/core/claim/remote.go:533 — "fmt.Sprintf("the reservation on %s is released: %s; %s", p.id, derivedWhy(p.id), nothingPushed))"
  evidence: internal/core/claim/crowd.go:59 — "if r.State == StateLive && r.Record != p.id {"
- ac-16 — MET: A third clone's peers --json, after P reserved iss-5 and itd-7 (and released iss-8) and Q reserved iss-6, lists all four from refs/remotes/origin/abcd-reserved/* with holder, footprint, effective expiry and state, counts octo's live reservations as 2, stamps the fetch, and prunes a branch origin dropped.
  evidence: internal/surface/cli/peers_reservations_test.go:110 — ""iss-5": {"octo", "live", []string{"a.go"}},"
  evidence: internal/surface/cli/peers_reservations_test.go:127 — "if got.LiveByHolder["octo"] != 2 || got.LiveByHolder["quinn"] != 1 || len(got.LiveByHolder) != 2 {"
  evidence: internal/surface/cli/peers_reservations_test.go:161 — "t.Fatalf("iss-6 is still listed after origin dropped it: %+v", res)"
  evidence: internal/core/claim/claim.go:43 — "TrackingPrefix = "refs/remotes/" + Remote + "/abcd-reserved/""
- ac-17 — MET: prepare resolves octo from both no-reply forms and writes Abcd-Holder: octo; the resolver reads the no-reply login first.
  evidence: internal/core/claim/message_test.go:144 — "for _, email := range []string{"1234+octo@users.noreply.github.com", "octo@users.noreply.github.com"} {"
  evidence: internal/core/claim/message_test.go:150 — "if p.holder != "octo" ||"
  evidence: internal/core/claim/holder.go:152 — "if u := NoReplyUsername(who.Email); u != "" && validUsername(u) {"
- ac-18 — MET: With user.email pat@example.com and ~/.abcd.noindex/config.json naming octo, prepare's holder is octo; the machine key is the resolver's second source, read from the machine layer only.
  evidence: internal/core/claim/message_test.go:160 — "r := newCheckout(t, "pat@example.com")"
  evidence: internal/core/claim/message_test.go:163 — "if err != nil || p.holder != "octo" {"
  evidence: internal/core/claim/holder.go:155 — "u, err := machineHolder(o.home())"
- ac-19 — MET_WITH_CONCERNS: Realised as delivered: at c9d94ebd1, the end of the delivered range, a reserve with neither source was refused naming user.email and the machine config.json and saying the repository's own configuration is never read, and the test asserted that over a repository naming mallory. Concern: PR #1037 (iss-2610152046379890, merged after this delivery under the 'no manual steps' ruling) added the session's forge token as a third source, so at HEAD such a reserve takes the holder from gh when it answers, and otherwise refuses naming only the token's failure class, never the two user-level sources; the repository's value is still never used and nothing is pushed, but the intent's In Scope and decision 32 still state the two-source refusal.
  evidence: c9d94ebd1:internal/core/claim/message_test.go:129 — "want := "cannot tell which GitHub user you are: user.email is not a GitHub no-reply address, and " + abcdhome.Display("config.json") + " sets no " +"
  evidence: c9d94ebd1:internal/core/claim/holder.go:116 — "return "", refuse("cannot tell which GitHub user you are: user.email is not a GitHub no-reply address, and %s sets no %s; "+"
  evidence: internal/core/claim/holder.go:170 — "return tokenHolder(root, o)"
  evidence: internal/core/claim/message_test.go:178 — "want := "could not work out whose GitHub account this session uses: " + errForgeNoToken.Error() + "; nothing was sent or pushed""
  evidence: internal/core/claim/holder_test.go:219 — "func TestRepoConfigCannotBlockTokenFallback(t *testing.T) {"
- ac-20 — MET: A laptop clone resolving octo through the machine config holds iss-5; a cloud clone with octo's no-reply address is refused with the back-off and 'you are already building iss-5 on another computer, until 2026-10-03T16:00:00Z', followed by the --take-over hint, and the branch is unmoved.
  evidence: internal/core/claim/remote_test.go:514 — "laptop := w.clone("pat@example.com", machineConfig(t, `{"reservation": {"github_username": "octo"}}`))"
  evidence: internal/core/claim/remote_test.go:521 — "want := "you are already building iss-5 on another computer, until 2026-10-03T16:00:00Z; pass --take-over to continue here""
  evidence: internal/core/claim/remote.go:403 — "return Result{}, message{}, "", backOff("you are already building %s on another computer, until %s; pass --take-over to continue here","
- ac-21 — MET: With its only remote renamed upstream the act refuses naming upstream as the remote found, with no remote it says so, and nothing reaches origin.
  evidence: internal/core/claim/remote_test.go:564 — "want := "reservations live on the remote named origin, and this checkout's remotes are: upstream; nothing was pushed""
  evidence: internal/core/claim/remote_test.go:572 — "if w.remoteTip("iss-5") != "" {"
- ac-22 — MET: An origin pointing at a path that does not exist fails at the fetch quoting git's reason (the gone.git path) and ending 'no reservation was made'; no ref and no fetch stamp is left.
  evidence: internal/core/claim/remote_test.go:583 — "if !errors.Is(err, ErrRefused) || !strings.HasPrefix(err.Error(), "could not read the reservations from origin: ") ||"
  evidence: internal/core/claim/remote_test.go:584 — "!strings.Contains(err.Error(), "gone.git") || !strings.HasSuffix(err.Error(), "; no reservation was made") {"
  evidence: internal/core/claim/remote_test.go:587 — "if refs := p.git("for-each-ref", "refs/heads/abcd-reserved/", TrackingPrefix); refs != "" {"
- ac-23 — MET: A bare origin whose pre-receive refuses refs/heads/abcd-reserved/* makes the act fail as 'origin refused the reservation push:' quoting the hook's own message and ending 'no reservation was made', leaving no ref.
  evidence: internal/core/claim/remote_test.go:607 — "if !errors.Is(err, ErrRefused) || !strings.HasPrefix(err.Error(), "origin refused the reservation push: ") ||"
  evidence: internal/core/claim/remote_test.go:608 — "!strings.Contains(err.Error(), "the proxy refuses abcd-reserved pushes") || !strings.HasSuffix(err.Error(), "; no reservation was made") {"
  evidence: internal/core/claim/remote_test.go:611 — "if w.remoteTip("iss-5") != "" || p.git("for-each-ref", "refs/heads/abcd-reserved/") != "" {"
- ac-24 — MET: The fixture installs the committed .githooks/pre-push and pushes a parentless empty-tree reserve, a renewal, a take-over and a tombstone with git push --porcelain, the verb's own push shape (pushTo), with hooks on and no receipt; each lands and the preflight never runs. The verb makes that push through gitutil.Net, which runs the repository's hooks, with no --no-verify.
  evidence: internal/surface/cli/githook_prepush_reserve_test.go:60 — "if out, err := c.push("--porcelain", "origin", next+":"+reservedRef); err != nil {"
  evidence: internal/surface/cli/githook_prepush_test.go:69 — "for _, rel := range []string{".githooks/pre-push", "scripts/preflight-receipt.sh"} {"
  evidence: internal/core/claim/remote.go:256 — "return []string{"push", "--porcelain", Remote, sha + ":" + ref}"
  evidence: internal/gitutil/net.go:39 — "The repository's hooks run (the pre-push gate is the point)"
- ac-25 — MET: Ordinary history pushed under the prefix, an empty-tree tip over a non-empty commit, and a non-empty-tree child of a reservation each meet 'no passing preflight'; reservation_shape requires every commit in the pushed range to have the empty tree.
  evidence: internal/surface/cli/githook_prepush_reserve_test.go:82 — "if err == nil || !strings.Contains(out, "no passing preflight") {"
  evidence: .githooks/pre-push:195 — "reservation_shape() {"
  evidence: .githooks/pre-push:198 — "only_empty_trees "$local_sha""
- ac-26 — MET: A forced push replacing a reservation tip with an unrelated empty-tree chain is refused as non-fast-forward and origin keeps the first reservation; the guard runs before the receipt-less exemption for every branch.
  evidence: internal/surface/cli/githook_prepush_reserve_test.go:128 — "if err == nil || !strings.Contains(out, "non-fast-forward") {"
  evidence: .githooks/pre-push:448 — "echo "pre-push: refusing non-fast-forward (force) push to '$branch'." >&2"
  evidence: .githooks/pre-push:456 — "if receiptless_row "$branch" && shape_check "$row_shape" "$remote_sha" "$local_sha"; then"
- ac-27 — MET: abcd reserve iss-5 with no --footprint exits 2 with the footprint refusal and origin holds nothing for iss-5.
  evidence: internal/surface/cli/reserve_surface_test.go:119 — "want := "abcd reserve: a reservation names the files the work will change: pass --footprint < path> once per path; nothing was pushed""
  evidence: internal/surface/cli/reserve_surface_test.go:123 — "if tip := w.remoteTip(t, "iss-5"); tip != "" {"
  evidence: internal/core/claim/prepare.go:230 — "if !release && !req.Release && !req.Renew && len(req.Footprint) == 0 {"
- ac-28 — MET: The written commit object carries no home, checkout path or host and is authored by the person's identity; the message passes the outbound policy, which refuses one carrying a session URL; reserve's text, --json and refusal renders under a temporary HOME carry no home path or host, and peers' failure render is checked for home paths.
  evidence: internal/core/claim/message_test.go:412 — "for _, leak := range []string{home, r.Root(), host} {"
  evidence: internal/core/claim/message_test.go:420 — "if err := checkMessage(r.Root(), p.reservation().text()); err != nil {"
  evidence: internal/surface/cli/reserve_surface_test.go:263 — "if len(host) >= 4 && strings.Contains(out, host) {"
  evidence: internal/surface/cli/peers_reservations_test.go:254 — "noHomePath(t, w.home, raw+stderr)"
- ac-29 — MET: A footprint carrying ; | & $( ) backticks, a colon, an asterisk and quotes is written and read back verbatim, and the traced git runner shows it in no argument; it travels only in the message on standard input.
  evidence: internal/core/claim/message_test.go:381 — "if log.anyArgContains("$(e)") || log.anyArgContains("refs/heads/main*") {"
  evidence: internal/core/claim/message_test.go:386 — "if err != nil || !ok || got.Unreadable != "" || !slices.Equal(got.Footprint, []string{hostile, "ok.go"}) {"
- ac-30 — MET: Two hundred long footprint paths are refused with the cap message after only config reads, and writeCommit measures again and refuses without writing an object.
  evidence: internal/core/claim/message_test.go:246 — "bytes, over the 4096-byte cap: name fewer or shorter footprint paths; nothing was pushed"
  evidence: internal/core/claim/message_test.go:265 — "if after := r.Git("count-objects"); after != before {"
- ac-31 — MET: Eleven malformed arguments (wrong case, other families, padding, path and refspec shapes, an escape) are refused as not a reservable id before any git command; reservableID admits only CitedIDRe ids canonicalised to iss- or itd- (and, since a later spec, a rel-vX.Y.Z subject) before refName is formed.
  evidence: internal/core/claim/message_test.go:122 — "for _, arg := range []string{"", "ISS-5", "adr-5", "spc-5", "iss-", "iss-0", "itd-5 ", "iss-5/../main","
  evidence: internal/core/claim/message_test.go:132 — "if cmds := log.subcommands(); len(cmds) != 0 {"
  evidence: internal/core/claim/claim.go:124 — "if !recordid.CitedIDRe.MatchString(arg) {"
- ac-32 — MET: The in-run claim's judge calls lease.Judge, as the reservation's judge does, and its DefaultLease, MinLease and MaxLease are the lease package's; TestClaimJudgesThroughTheSharedLease adds the shared clamp, and the in-run claim tests were last edited on 2026-10-04, before the 2026-10-12 refactor, so they run unchanged.
  evidence: internal/core/implement/claim.go:427 — "return lease.Judge(lease.Stamp{ClaimedAt: c.ClaimedAt, ExpiresAt: c.ExpiresAt}, UnreadableClaimGrace, now)"
  evidence: internal/core/claim/remote.go:632 — "j.verdict = lease.Judge(lease.Stamp{ClaimedAt: t.ClaimedAt, ExpiresAt: t.ExpiresAt, FirstSeen: t.FirstSeen}, lease.Default, p.now)"
  evidence: internal/core/implement/lease_test.go:25 — "func TestClaimJudgesThroughTheSharedLease(t *testing.T) {"
  evidence: git log -- internal/core/implement/claim_test.go — "a1ed114bd Sun Oct 4 08:59:13 2026 +0100 feat!: rename abcd's home to ~/.abcd.noindex and stop while ~/.abcd stands"
- ac-33 — MET: A reserve, a renewal and a release authored by a no-reply identity and carrying Assisted-by: abcd:< version> pass scripts/check-attribution.sh commits over their range with 'check-attribution: clean'.
  evidence: internal/core/claim/gates_test.go:65 — "cmd := exec.Command("bash", filepath.Join(root, "scripts", "check-attribution.sh"), "commits", "origin/main", tip)"
  evidence: internal/core/claim/gates_test.go:68 — "if out, err := cmd.CombinedOutput(); err != nil || !strings.Contains(string(out), "check-attribution: clean") {"
- ac-34 — MET: Start on a planned intent whose spec names packages internal/widget and commands/ pushes P's reservation with footprint [internal/widget, commands], records reserved_at and a reservation note, and a resume renews it as the tip's child; abcd build --json names the reservation it took.
  evidence: internal/core/implement/loop/build_reserve_test.go:220 — "if fp := footprintOf(t, bare, "itd-10"); !slices.Equal(fp, []string{"internal/widget", "commands"}) {"
  evidence: internal/core/implement/loop/build_reserve_test.go:224 — "if tip.holder != "octo" || tip.released {"
  evidence: internal/surface/cli/build_reserve_surface_test.go:39 — "if !strings.HasPrefix(started.Reservation, "reserved itd-8 on origin for this copy until ") ||"
- ac-35 — MET: A spec with no ## Footprint refuses the start at the reservation stage, not as contention, naming spc-1 and the missing section and that nothing was started; no run tier is created and origin holds no reservation.
  evidence: internal/core/implement/loop/build_reserve_test.go:292 — "want := "spc-1 has no footprint under ## Footprint to reserve itd-10 with; fill its packages line, then run abcd build again; nothing was started""
  evidence: internal/core/implement/loop/build_reserve_test.go:297 — "if tip := originReservation(t, bare, reservedRef); tip.commit != "" {"
- ac-36 — MET: abcd build itd-8 --release after a build pushes a tombstone child of the run's reservation, says the run's state is unchanged, leaves the state file byte-identical, and a second release reports already_released.
  evidence: internal/surface/cli/build_reserve_surface_test.go:54 — "if code != 0 || !strings.Contains(out, "released the reservation on itd-8") || !strings.Contains(out, "the run's state is unchanged") {"
  evidence: internal/surface/cli/build_reserve_surface_test.go:61 — "if parent := gitCmd(t, w.origin, "rev-parse", tomb+"^"); parent != reserved {"
  evidence: internal/surface/cli/build_reserve_surface_test.go:64 — "if after, err := os.ReadFile(statePath); err != nil || string(after) != string(before) {"
- ac-37 — MET: The drain reserves the issue whose lane it opens, as P from this copy, with its found_at path (commands/drain.md, and AGENTS.md from AGENTS.md:3 with the locator dropped), records it in the run, and reserves no issue whose lane has not opened.
  evidence: internal/core/implement/loop/drain_reserve_test.go:69 — "if fp := footprintOf(t, bare, secondIssue); !slices.Equal(fp, []string{"commands/drain.md"}) {"
  evidence: internal/core/implement/loop/drain_reserve_test.go:96 — "if fp := footprintOf(t, bare, eligibleIssue); !slices.Equal(fp, []string{"AGENTS.md"}) {"
- ac-38 — MET: With Q's live reservation on itd-10 fetched into P's view, the candidates exclude itd-10 at the peers check with a reason naming quinn, the expiry and 'as of the last fetch under a minute ago'; the CLI's build next --json excludes the reserved intent from disk alone.
  evidence: internal/core/implement/loop/build_reserve_test.go:71 — "if e.Check != CheckPeers || !strings.Contains(e.Reason, "reserved by quinn until ") ||"
  evidence: internal/core/implement/loop/build_reserve_test.go:72 — "!strings.Contains(e.Reason, "as of the last fetch under a minute ago") {"
  evidence: internal/surface/cli/reservation_render_test.go:183 — "if code != 2 || !strings.Contains(raw, `"id": "itd-8"`) || !strings.Contains(raw, "reserved by octo until ") {"
- ac-39 — MET: Start over Q's fetched reservation is refused at the check stage, peers check, as contention, naming quinn and the view's age, never suggesting --take-over, creating no run and leaving origin's tip unmoved.
  evidence: internal/core/implement/loop/build_reserve_test.go:123 — "if r.Stage != "check" || r.Check != CheckPeers || !r.Contention {"
  evidence: internal/core/implement/loop/build_reserve_test.go:126 — "if !strings.Contains(r.Reason, "reserved by quinn until ") || !strings.Contains(r.Reason, "as of the last fetch") {"
- ac-40 — MET: A step 61 minutes into the run's two-hour reservation pushes a child of the tip with a fresh two-hour lease, logs reservation_renewed with the new expiry in the run record, and still performs its stage.
  evidence: internal/core/implement/loop/reservation_test.go:176 — "reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-61*time.Minute), "internal/widget")"
  evidence: internal/core/implement/loop/reservation_test.go:181 — "if after.commit == before.commit || after.parent != before.commit {"
  evidence: internal/core/implement/loop/reservation_test.go:187 — "if note := recordNote(t, f.repo, f.runID, StageReservationRenewed); !strings.Contains(note, "itd-10") || !strings.Contains(note, after.expires.Format(time.RFC3339)) {"
- ac-41 — MET: A step 30 minutes into the lease leaves origin's tip unchanged and logs no reservation stage.
  evidence: internal/core/implement/loop/reservation_test.go:199 — "reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-30*time.Minute), "internal/widget")"
  evidence: internal/core/implement/loop/reservation_test.go:202 — "if after := originReservation(t, f.bare, reservedRef); after.commit != before.commit {"
- ac-42 — MET: After P's lapsed reservation is taken over by Q, the step at the arming is refused as contention naming quinn's reservation and expiry, no pr merge runs, the lane stays unarmed, and Q's tip is untouched; landFence runs at the top of landArm.
  evidence: internal/core/implement/loop/reservation_test.go:272 — "want := "the reservation on itd-10 is now quinn's, until " + qTip.expires.Format(time.RFC3339) + ", so this lane does not arm its merge""
  evidence: internal/core/implement/loop/reservation_test.go:249 — "if strings.Contains(f.ghLog(t), "pr merge") {"
  evidence: internal/core/implement/loop/land.go:722 — "holds, notice, err := landFence(c, *lane, fenceLease)"
- ac-43 — MET: Under a mirror whose queue timeout is 45 minutes (and 150, 3000), the fence renews P's reservation as the tip's child with a 24-hour lease, well past 45 minutes plus the margin, before the rule decides whether to arm; since the 2026-10-14 review the fence takes lease.Max under any mirror rather than the timeout plus a margin.
  evidence: internal/core/implement/loop/reservation_test.go:349 — "{"45", 24 * time.Hour},"
  evidence: internal/core/implement/loop/reservation_test.go:361 — "if after.parent != before.commit || after.expires.Sub(after.claimed) != tc.lease || after.claimed.Before(now) {"
  evidence: internal/core/implement/loop/land.go:722 — "holds, notice, err := landFence(c, *lane, fenceLease)"
- ac-44 — MET: With no ruleset mirror the fence renews for 24 hours, lease.Max, and leaves the pull request open for a person (decision 40, delivered in PR #1034).
  evidence: internal/core/implement/loop/parked_reserve_test.go:192 — ""no mirror":"
  evidence: internal/core/implement/loop/parked_reserve_test.go:201 — "if tip.expires.Sub(tip.claimed) != 24*time.Hour {"
  evidence: internal/core/implement/loop/reservation.go:216 — "const fenceLease = lease.Max"
- ac-45 — MET: The fixture's lane commits the spec close (Delivers: itd-10) and its head is pushed to origin's main, so the intent is shipped there; the next step marks the lane done and pushes a tombstone child of the fenced reservation, its note saying it released it.
  evidence: internal/core/implement/loop/reservation_test.go:459 — "if !tomb.released || tomb.parent != held.commit || tomb.holder != "octo" {"
  evidence: internal/core/implement/loop/reservation_test.go:462 — "if note := landNote(t, f); !strings.Contains(note, "released the reservation on itd-10") {"
  evidence: internal/core/implement/loop/land_test.go:230 — "f.repo.Git("push", "-q", "origin", head+":refs/heads/main")"
  evidence: internal/core/implement/loop/land_test.go:260 — ""Delivers: itd-10""
- ac-46 — MET: Q takes the fenced reservation over past its day plus the margin; after the merge the step completes the landing, leaves Q's tip as it stands, and its note names the reservation left with quinn.
  evidence: internal/core/implement/loop/reservation_test.go:509 — "if got := originReservation(t, f.bare, reservedRef); got.commit != qTip.commit {"
  evidence: internal/core/implement/loop/reservation_test.go:512 — "if note := landNote(t, f); !strings.Contains(note, "left the reservation on itd-10") || !strings.Contains(note, "quinn") {"
- ac-47 — MET: With ten live reservations an eleventh is refused (ErrRefused, a CapError holding the ten) naming the cap and iss-1..iss-10, and origin gets nothing, while Q is not refused by P's ten; the cap is MaxLive, ten, unless the person's own machine setting reservation.max_live (added after this delivery) changes it.
  evidence: internal/core/claim/capoverlap_test.go:34 — "want := "you hold 10 live reservations, the most one person may hold: iss-1, iss-10, iss-2, iss-3, iss-4, iss-5, " +"
  evidence: internal/core/claim/capoverlap_test.go:46 — "t.Fatalf("a reservation over the cap reached origin: %s", tip)"
  evidence: internal/core/claim/crowd.go:17 — "const MaxLive = 10"
- ac-48 — MET: With iss-1 reserved by octo on commands/reserve.md and internal/core/claim, quinn's reserves sharing an equal path, a file beneath a reserved directory or a directory over a reserved file are each refused with the back-off naming the shared path quoted, iss-1, octo and its expiry, and nothing reaches origin.
  evidence: internal/core/claim/capoverlap_test.go:130 — "want := `"` + c.path + `" is in the footprint of iss-1` + held"
  evidence: internal/core/claim/capoverlap_test.go:134 — "if tip := w.remoteTip(c.id); tip != "" {"
  evidence: internal/core/claim/crowd.go:112 — "return backOff("%s is in the footprint of %s, reserved by %s until %s%s; take other work; %s","
- ac-49 — MET: After Q's fetch, origin is moved away, HTTP requests are counted and gh is stubbed: abcd iss-5 and the board show octo's reservation 'as of the last fetch', build itd-8 is refused at the peers check (exit 3) and build next excludes itd-8, a reservation made after the fetch is not seen, the fetch stamp is unmoved, and no request or gh call is made; build next's age wording is asserted at the loop level (ac-38).
  evidence: internal/surface/cli/reservation_render_test.go:101 — "if err := os.Rename(w.origin, w.origin+".away"); err != nil {"
  evidence: internal/surface/cli/reservation_render_test.go:131 — "!strings.HasPrefix(r.Text, "live, reserved by octo until ") || !strings.Contains(r.Text, ", as of the last fetch ") {"
  evidence: internal/surface/cli/reservation_render_test.go:166 — "if !strings.HasPrefix(reserved, "reserved by octo until ") || !strings.Contains(reserved, ", as of the last fetch ") {"
  evidence: internal/surface/cli/reservation_render_test.go:178 — "if code != 3 || !strings.Contains(raw, `"check": "peers"`) || !strings.Contains(raw, "reserved by octo until ") ||"
  evidence: internal/surface/cli/reservation_render_test.go:190 — "if n := atomic.LoadInt32(&hits); n != 0 {"
- ac-50 — MET: With no fetch stamped in a checkout that has an origin, Describe renders the reservation as 'unknown, nothing has been fetched here; `abcd peers` fetches', never none, and a tip a plain git fetch left is shown with its age unknown.
  evidence: internal/core/record/reservation_test.go:54 — "d.Reservation.Text != "unknown, nothing has been fetched here; `abcd peers` fetches" {"
  evidence: internal/core/claim/render.go:137 — "return "as of a fetch of unknown age""
- ac-51 — MET: After the landing removes the lane's build/ branch, origin still holds abcd-reserved/itd-10, the checkout's remote-tracking ref for it remains, and no local branch exists under the reservation prefix.
  evidence: internal/core/implement/loop/reservation_test.go:476 — "if got := f.repo.Git("for-each-ref", "refs/heads/"+l.Branch); got != "" {"
  evidence: internal/core/implement/loop/reservation_test.go:479 — "if got := originReservation(t, f.bare, reservedRef); got.commit == "" {"
  evidence: internal/core/claim/claim.go:41 — "// The prefix is disjoint from the implement loop's lane prefix build/."
- ac-52 — MET: The cloud copy's reserve with TakeOver is ActTakenOver as a child of the laptop's commit whose Abcd-Copy is the cloud copy's key, Replaced naming octo's other copy; the CLI prints 'took over iss-5' and 'your copy on another computer' and never a copy key.
  evidence: internal/core/claim/remote_test.go:545 — "if err != nil || res.Act != ActTakenOver || res.Replaced == nil || res.Replaced.Copy != CopyAnother || res.Replaced.Holder != "octo" {"
  evidence: internal/core/claim/remote_test.go:553 — "if w.parentOf(tip) != first.Commit || !strings.Contains(w.remoteMessage(tip), "Abcd-Copy: "+cloudKey) {"
  evidence: internal/surface/cli/reserve_surface_test.go:228 — "if code != 0 || !strings.Contains(text, "took over iss-5") || !strings.Contains(text, "your copy on another computer") {"
- ac-53 — MET: After P's second computer takes the reservation over with TakeOver, the laptop lane's step at the arming is refused as contention, 'moved to your copy on another computer', and nothing is armed.
  evidence: internal/core/implement/loop/reservation_test.go:289 — "if res, err := claim.Reserve(p2, claim.Request{Record: "itd-10", Footprint: []string{"internal/widget"}, TakeOver: true}, clockAt(now.Add(-time.Minute))); err != nil || res.Act != claim.ActTakenOver {"
  evidence: internal/core/implement/loop/reservation_test.go:293 — "if want := "the reservation on itd-10 moved to your copy on another computer, so this lane does not arm its merge here"; r.Reason != want {"

Gap audit:
- honoured:
  - When an agent starts on an item, abcd reserves it for the person the agent works for: a build at its start, a drain as each lane opens.
    evidence: internal/core/implement/loop/build_reserve_test.go:220 — "if fp := footprintOf(t, bare, "itd-10"); !slices.Equal(fp, []string{"internal/widget", "commands"}) {"
    evidence: internal/core/implement/loop/drain_reserve_test.go:69 — "if fp := footprintOf(t, bare, secondIssue); !slices.Equal(fp, []string{"commands/drain.md"}) {"
  - The reservation is a branch on the shared remote that nobody merges, a chain of empty-tree commits that leaves the caller's HEAD, index and working tree alone.
    evidence: internal/core/claim/claim.go:42 — "RefPrefix = "refs/heads/abcd-reserved/""
    evidence: internal/core/claim/remote_test.go:163 — "tree != "4b825dc642cb6eb9a060e54bf8d69288fbee4904""
    evidence: internal/core/claim/remote_test.go:176 — "h != head || i != index || s != status"
  - If someone else already holds the item, or the files the work will touch, abcd says who and until when, and the agent takes something else.
    evidence: internal/core/claim/remote_test.go:195 — "want := "iss-5 is reserved by octo until 2026-10-03T16:00:00Z; back off and take other work""
    evidence: internal/core/claim/capoverlap_test.go:130 — "want := `"` + c.path + `" is in the footprint of iss-1` + held"
    evidence: internal/core/implement/loop/build_reserve_test.go:71 — "if e.Check != CheckPeers || !strings.Contains(e.Reason, "reserved by quinn until ") ||"
  - Everyone who works on the project sees the reservation, from any computer or clone, as last fetched and with the view's age.
    evidence: internal/surface/cli/peers_reservations_test.go:127 — "if got.LiveByHolder["octo"] != 2 || got.LiveByHolder["quinn"] != 1 || len(got.LiveByHolder) != 2 {"
    evidence: internal/surface/cli/reservation_render_test.go:131 — "!strings.HasPrefix(r.Text, "live, reserved by octo until ") || !strings.Contains(r.Text, ", as of the last fetch ") {"
  - A reservation is renewed at half its lease while the work goes on, and one left behind lapses and can be taken over.
    evidence: internal/core/implement/loop/reservation_test.go:181 — "if after.commit == before.commit || after.parent != before.commit {"
    evidence: internal/core/claim/remote_test.go:388 — "res.Replaced.Holder != "octo" ||"
  - It stops counting the moment the work is merged: a record terminal on origin's default branch releases its reservation, with no step needed.
    evidence: internal/core/claim/remote_test.go:500 — "if r.State != StateReleased || r.ReleasedBy != ReleasedByDefaultBranch {"
  - The one that loses finds out in seconds: a lost race is refused naming the winner and writes nothing.
    evidence: internal/core/claim/remote_test.go:254 — "want := "iss-5 was reserved by octo while this push was on its way, until 2026-10-03T16:00:00Z; back off and take other work""
  - Moving a piece of work from one of a person's copies to another is one flag, --take-over.
    evidence: internal/surface/cli/reserve_surface_test.go:228 — "if code != 0 || !strings.Contains(text, "took over iss-5") || !strings.Contains(text, "your copy on another computer") {"
    evidence: internal/core/claim/remote_test.go:553 — "if w.parentOf(tip) != first.Commit || !strings.Contains(w.remoteMessage(tip), "Abcd-Copy: "+cloudKey) {"
  - One liveness judgement serves the in-run implement claim and the reservation.
    evidence: internal/core/implement/claim.go:427 — "return lease.Judge(lease.Stamp{ClaimedAt: c.ClaimedAt, ExpiresAt: c.ExpiresAt}, UnreadableClaimGrace, now)"
    evidence: internal/core/claim/remote.go:632 — "j.verdict = lease.Judge(lease.Stamp{ClaimedAt: t.ClaimedAt, ExpiresAt: t.ExpiresAt, FirstSeen: t.FirstSeen}, lease.Default, p.now)"
  - The pre-push hook exempts reservation chains narrowly and still refuses a non-fast-forward push.
    evidence: .githooks/pre-push:195 — "reservation_shape() {"
    evidence: internal/surface/cli/githook_prepush_reserve_test.go:128 — "if err == nil || !strings.Contains(out, "non-fast-forward") {"
- diverged:
  - A reservation lasts two hours: once a lane reaches land the fence renews it for a day, lease.Max, mirror or not (decision 40, amending decision 29 and C44 after a two-hour lapse let a colleague's drain redo work only waiting to merge), and the queue's check timeout is no longer read.
    evidence: internal/core/implement/loop/reservation.go:216 — "const fenceLease = lease.Max"
    evidence: internal/core/implement/loop/parked_reserve_test.go:201 — "if tip.expires.Sub(tip.claimed) != 24*time.Hour {"
    evidence: .abcd/work/issues/resolved/iss-2610152020234609-fence-holds-a-day-without-a-mirror.md:19 — "a pull request the land step leaves waiting on a person holds its reservation on the shared remote for a day, mirror or not"
  - A person working by hand reserves with one short command: decision 39 made abcd reserve an agent's verb, kept off a person's /abcd: menu though still runnable from the CLI, and a person sees reservations on the board and a record's page.
    evidence: commands/reserve.md:6 — "user-invocable: false"
  - With neither a no-reply user.email nor the machine key, every write is refused naming both sources (In Scope, decision 32, C19): after this delivery the act asks the session's forge token third, reserves under its account when gh answers, and otherwise refuses naming only the token's failure; the intent's text was not amended.
    evidence: internal/core/claim/holder.go:170 — "return tokenHolder(root, o)"
    evidence: internal/core/claim/message_test.go:178 — "want := "could not work out whose GitHub account this session uses: " + errForgeNoToken.Error() + "; nothing was sent or pushed""
    evidence: internal/core/claim/holder_test.go:219 — "func TestRepoConfigCannotBlockTokenFallback(t *testing.T) {"
  - A cloud session reserves, sees and honours reservations exactly as a second person's machine does (decision 4): as delivered it reserved only once its clone's identity named the person; a hosted clone carries the host's vendor identity (the 2026-10-15 probe) and was refused for want of a username until the token fallback merged after this delivery (PR #1037).
    evidence: c9d94ebd1:internal/core/claim/holder.go:116 — "return "", refuse("cannot tell which GitHub user you are: user.email is not a GitHub no-reply address, and %s sets no %s; "+"
    evidence: .abcd/work/issues/resolved/iss-2610152046379890-a-hosted-cloud-session-s-clone-carries.md:21 — "A hosted cloud session's clone carries the host's default vendor identity as its git author, not the person's"
  - The land fence catches a reservation lost mid-work before the merge is armed (Mechanism): it does, but as delivered it ran only at arming, after the lane's push and pull request, so a drain lane paused past its lease and taken over by another account became a duplicate public pull request; a push fence and drain-move renewals were added after this delivery (iss-2610161015393523, major).
    evidence: .abcd/work/issues/resolved/iss-2610161015393523-a-drain-s-in-progress-lane-is-renewed.md:20 — "the land fence runs only at arming, after the push and the pull request, so the duplicate becomes a public pull request"
    evidence: internal/core/implement/loop/land.go:722 — "holds, notice, err := landFence(c, *lane, fenceLease)"
  - The footprint refusal keeps apart work on different items that would touch the same files (Mechanism): as delivered, directory footprints turned away whole packages of work, one account holding internal/core/implement/loop blocking 32 eligible issues, and a same-person renewal had to drop files it changes; later records narrowed the check (iss-2610152125101468).
    evidence: .abcd/work/issues/resolved/iss-2610152125101468-a-reservation-footprint-can-hold-a-whole.md:22 — "while one account holds internal/core/implement/loop, every other account is blocked from all 32 eligible issues under it"
    evidence: .abcd/work/issues/resolved/iss-2610152125101468-a-reservation-footprint-can-hold-a-whole.md:20 — "the messaging reservation's renewal was refused twice and had to leave out .githooks/pre-push, the ruleset mirror and the generated CLI reference, which it does change"
  - A per-person cap of ten live reservations (decision 15): since iss-2610170717515674 the cap is the person's own machine setting reservation.max_live, 1 to 100, ten when unset.
    evidence: internal/core/claim/crowd.go:13 — "// MaxLive is the most live reservations one person may hold unless their"
    evidence: internal/core/claim/maxlive.go:12 — "reservation.max_live in ~/.abcd.noindex/config.json, the machine layer"
- missing: (none)

Scope-condition dispositions:
- cond-2610031753354810 — survived: Reservations live on origin alone: a checkout without a remote named origin is refused naming the remotes found, and a remote that refuses the push, standing for a missing write permission, makes no reservation.
  evidence: internal/core/claim/claim.go:38 — "Remote = "origin""
  evidence: internal/core/claim/remote_test.go:564 — "want := "reservations live on the remote named origin, and this checkout's remotes are: upstream; nothing was pushed""
  evidence: internal/core/claim/remote_test.go:607 — "!strings.HasPrefix(err.Error(), "origin refused the reservation push: ")"
- cond-2610031753352651 — survived: Every act is one plain fast-forward branch push under refs/heads/abcd-reserved/; the forge's ruleset there refuses only deletion and non-fast-forward, and this checkout's view as last fetched holds 138 reservation branches pushed to the real origin. The hosted proxy's acceptance still rests on the 2026-10-03 probe; nothing in the repository re-observes it.
  evidence: internal/core/claim/remote.go:256 — "return []string{"push", "--porcelain", Remote, sha + ":" + ref}"
  evidence: .abcd/work/rulesets/reservations-protection.json:24 — ""type": "non_fast_forward""
  evidence: refs/remotes/origin/abcd-reserved/iss-2610100649471088 — "abcd reservation: iss-2610100649471088"
- cond-2610031753357820 — untested: Every on: push trigger still filters on main or v* tags, and the one workflow added since, the reservation sweep, runs on a schedule and by hand, so the argument still stands; but nothing in the delivery or the repository records a real reservation push and the runs it did or did not start, which the condition names as its proof.
  evidence: .github/workflows/auto-release.yml:39 — "branches: [main]"
  evidence: .github/workflows/attribution.yml:52 — "branches: [main]"
  evidence: .github/workflows/release.yml:29 — "- 'v*'"
  evidence: .github/workflows/reservation-sweep.yml:27 — "schedule:"
- cond-2610031753357092 — survived: Each record has one reservation branch, a release adds a tombstone and deletes nothing, and origin as last fetched holds 150 branches, 138 of them reservations, far under 5,000; since this delivery a daily sweep (iss-2610142200389875, not itd-118) deletes spent ones.
  evidence: internal/core/claim/remote_test.go:349 — "if tip == "" || w.parentOf(tip) != first.Commit || !strings.HasPrefix(msg, "abcd reservation released: iss-5") ||"
  evidence: refs/remotes/origin/abcd-reserved/iss-2610100649471088 — "abcd reservation: iss-2610100649471088"
  evidence: .github/workflows/reservation-sweep.yml:28 — "- cron: "37 4 * * *""
- cond-2610031753351641 — narrowed: Reservations stay advisory among writers: the holder is a self-declared trailer and any writer can append a child to another's branch. But the delivery made deletion and force refused: the forge's 'reservations protection' ruleset refuses deletion and non-fast-forward under refs/heads/abcd-reserved/** for everyone but the sweep App, and the committed pre-push hook refuses a deletion there.
  narrowing: Holds for fast-forward appends only: any writer can still push a child naming any holder onto a reservation branch; deleting or rewriting one is refused by the forge ruleset (bypassed only by the sweep App) and, for a push through the hooks, by the pre-push hook.
  evidence: .abcd/work/rulesets/reservations-protection.json:21 — ""type": "deletion""
  evidence: .abcd/work/rulesets/reservations-protection.json:4 — ""actor_id": "RESERVATION_SWEEP_APP_ID","
  evidence: .githooks/pre-push:432 — "a branch under $row_prefix ends by a push, never a deletion"
  evidence: .abcd/development/specs/closed/spc-2610031753359849-nothing-tells-an-agent-that-a-record-it.md:778 — "people who share push access in good faith. Any writer with push access"
- cond-2610031753358669 — survived: One username is one person: a laptop resolving octo through the machine config and a cloud clone resolving it through the no-reply address are judged one person, two copies, and usernames compare without regard to case.
  evidence: internal/core/claim/remote_test.go:521 — "want := "you are already building iss-5 on another computer, until 2026-10-03T16:00:00Z; pass --take-over to continue here""
  evidence: internal/core/claim/holder.go:48 — "func sameHolder(a, b string) bool { return strings.EqualFold(a, b) }"
- cond-2610031753352100 — survived: Expiry is judged by the reader's clock against the writer's times with a five-minute take-over margin; the day clamp counts from the earlier of the claim and the reader's now (and from the first sighting for a claim dated more than five minutes ahead), and the code names the residual bound for a holder whose clock runs behind.
  evidence: internal/core/claim/claim.go:61 — "TakeoverMargin = lease.Skew"
  evidence: internal/core/lease/lease.go:33 — "const Skew = 5 * time.Minute"
  evidence: internal/core/lease/lease.go:94 — "if ceiling := earlier(from(s.ClaimedAt, s.FirstSeen, now), now).Add(Max); ceiling.Before(exp) {"
  evidence: internal/core/claim/claim.go:57 — "// A holder whose clock runs behind by more than half its lease plus this"
- cond-2610031753358608 — survived: Renewal happens only when something runs it, a step at half the lease (and, since the 2026-10-14 review, a receipt and each drain move over a parked lane); a reservation left to lapse is taken over and the land fence then refuses the merge, the failure the multi-account drain investigation of 2026-10-16 met (iss-2610161015393523).
  evidence: internal/core/implement/loop/reservation_test.go:181 — "if after.commit == before.commit || after.parent != before.commit {"
  evidence: internal/core/implement/loop/reservation_test.go:265 — "reserveRun(t, f.repo, f.runID, "itd-10", now.Add(-130*time.Minute), "internal/widget")"
  evidence: internal/core/implement/loop/reservation_test.go:272 — "want := "the reservation on itd-10 is now quinn's, until " + qTip.expires.Format(time.RFC3339) + ", so this lane does not arm its merge""
- cond-2610031753354769 — survived: The cap and the overlap are judged by crowding after the act's own fetch and before the commit and push, on that view, with no compare-and-swap across records.
  evidence: internal/core/claim/remote.go:169 — "gap, err := fetch(root, p.now, noReservation, reservations)"
  evidence: internal/core/claim/remote.go:196 — "if err := crowding(p, j, &res); err != nil {"
  evidence: internal/core/claim/crowd.go:48 — "// Both are judged on the view just fetched, not compared-and-swapped: each"
- cond-2610031753356382 — narrowed: The derived release covers the merge itself as assumed. But decision 40 made the fence's own renewal a day, mirror or not, and later renewals keep that term, so the merge queue's wait no longer depends on implement step renewing during the first day.
  narrowing: The queue's wait depends on continued renewal (a step, a receipt, or a drain move over the parked lane) only past the day the fence's own lease.Max renewal holds; within that day the fence covers it alone, and the derived release covers the merge.
  evidence: internal/core/implement/loop/reservation.go:216 — "const fenceLease = lease.Max"
  evidence: internal/core/implement/loop/parked_reserve_test.go:201 — "if tip.expires.Sub(tip.claimed) != 24*time.Hour {"
  evidence: internal/core/claim/remote_test.go:500 — "if r.State != StateReleased || r.ReleasedBy != ReleasedByDefaultBranch {"
- cond-2610031753358785 — survived: Renders and the build's peers check read refs/remotes/origin/abcd-reserved/* from disk: a reservation made after the last fetch reads as none, a view a plain git fetch left is shown with its age unknown, and the reserving push itself refuses the same item as a lost race.
  evidence: internal/surface/cli/reservation_render_test.go:146 — "if r := rec.Reservation; r == nil || r.State != "none" {"
  evidence: internal/core/claim/render.go:137 — "return "as of a fetch of unknown age""
  evidence: internal/core/claim/remote_test.go:254 — "want := "iss-5 was reserved by octo while this push was on its way, until 2026-10-03T16:00:00Z; back off and take other work""
- cond-2610031753359470 — survived: A reservation carries the record id, a username, a random copy key, repository-relative paths, UTC times and the abcd label; the written commit carries no home, checkout path or host, and the message passes the outbound policy before it is written.
  evidence: internal/core/claim/message_test.go:412 — "for _, leak := range []string{home, r.Root(), host} {"
  evidence: internal/core/claim/message_test.go:420 — "if err := checkMessage(r.Root(), p.reservation().text()); err != nil {"
  evidence: internal/core/claim/message_test.go:31 — "Abcd-Copy: 9f2c41d07be35a6c80e1f4b2a7d39c55"
- cond-2610031753355764 — narrowed: The cloud copy sees only origin: the reservation lives on origin, each clone keeps its own copy key, and the tests' cloud clone, carrying the person's no-reply address, is judged exactly as a second machine. But the 2026-10-15 probe found a hosted clone carries the host's vendor identity, and as delivered the holder resolver then refused for want of a username unless a person set user.email or the machine key.
  narrowing: As delivered it holds for a cloud clone whose git identity is the person's GitHub no-reply address or whose home sets reservation.github_username; a hosted clone with the host's vendor identity could not reserve until the forge-token holder, merged after this delivery (PR #1037, iss-2610152046379890), named the person from the session's token.
  evidence: internal/core/claim/remote_test.go:515 — "cloud := w.clone("1234+octo@users.noreply.github.com", "")"
  evidence: c9d94ebd1:internal/core/claim/holder.go:116 — "return "", refuse("cannot tell which GitHub user you are: user.email is not a GitHub no-reply address, and %s sets no %s; "+"
  evidence: .abcd/work/issues/resolved/iss-2610152046379890-a-hosted-cloud-session-s-clone-carries.md:21 — "A hosted cloud session's clone carries the host's default vendor identity as its git author, not the person's"
  evidence: internal/core/claim/holder.go:170 — "return tokenHolder(root, o)"
<!-- abcd-review-end receipt=rcp-0d4b99e46049 -->

## Grounds

- pursued: A second person joined today, and agents on two computers plus a cloud session now work on the project at once. We expect no item to be worked twice in the first month after it ships; one duplicate pull request or one issue fixed twice by two people would show it wrong.
