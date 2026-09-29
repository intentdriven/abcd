---
id: itd-148
slug: every-change-starts-in-its-own-worktree-the-primary-checkout
spec_id: spc-42
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-118, itd-33]
blocked_by: [itd-2609091014076309]
related_adrs: [adr-2609091248200336]
severity: major
impact: additive
---

# Every change starts in its own worktree in abcd's store: the primary checkout is a read-only surface, and abcd blocks mutations there for every session but a declared coordinator

Typed links: `blocked_by` [itd-2609091014076309](../drafts/itd-2609091014076309-session-and-agent-worktrees-live-in-a-machine-scoped-store-t.md) (the store under the home folder, which owns adding, listing and clearing away worktrees; this intent uses those verbs, so it waits on that draft); `related_adrs` [adr-2609091248200336](../../decisions/adrs/2609091248200336-a-tool-never-creates-directories-in-user-owned-project-space.md) (the rule the store enacts). This record was rewritten on 2026-09-29 to the product thinker's rulings (see `## Decisions`); the rewrite **reverses** the scope its 2026-08-26 planning interview gave it, a worktree verb family and a sweep of its own, with worktrees placed inside the primary checkout.

## Press Release

> **abcd turns the primary checkout into a read-only surface: every change — a one-line docs fix included, peers present or not — begins in its own worktree in abcd's store under the home folder, and abcd enforces the boundary mechanically.** A session gets its worktree from the store (itd-2609091014076309), which places it under `~/.abcd/worktrees/<root-sha>/` and never beside or inside the user's own project; abcd records the entry where every peer session sees it on its next prompt, and seeds the worktree with the protections the primary checkout has — the private name-guard layer included, pre-populated to a floor of four categories (home/user paths, machine hostnames, personal email, real surname), each value held on the machine and never committed. A host-hook backed by the binary refuses file writes and git mutations attempted in the primary checkout, so a session whose working directory silently reverts to the shared tree is stopped at its first recognisable write instead of contaminating a tree another session is about to move. One session may declare itself the coordinator, the session that lands and records the others' work, and only that session is exempt from the block. Listing the worktrees and clearing away the ones whose work is done are the store's verbs, so there is one place to ask what exists and one way to reclaim it.
>
> "I had two sessions going and asked one for a two-line docs fix, and it just did it — in its own worktree, kept with abcd's other machine state rather than beside my projects, visible to the other session, PR up," said Kira, an open-source project lead running concurrent agents. "When the other session's shell slipped back into my main checkout, abcd stopped its first edit and named the worktree route instead. The one session I told to coordinate could still land work in the main checkout; nobody else could."

## Why This Matters

The checkout-is-the-unit-of-isolation convention has been carried by vigilance,
and the ledger records vigilance failing safe rather than the setup holding:

- iss-213: several agents in ONE worktree produced a false-green preflight
  spanning two branch switches, and a `git rebase` silently rebased main.
- iss-2608230847432285 (major): per-agent worktrees did not isolate sessions
  whose shell cwd silently reverted to the shared checkout. Two sessions wrote
  into the primary tree believing they were in their worktrees, and only the
  diff-you-did-not-make convention prevented a bad commit. That record asks in
  terms for "whatever makes a session's tree unambiguous rather than
  remembered". The mutation block in the primary checkout is that durable
  form, aimed at the exact failure direction observed.
- The lint gates read the whole working tree, so any foreign work-in-progress
  in a shared checkout fails `make preflight` in both directions; a per-change
  worktree makes a gate's verdict describe the change it gates.
- itd-107 leaves open (its orchestration caveats) whether independent peer
  sessions are kept apart by policy or by per-session worktrees; this intent
  supplies that answer.

Making the primary checkout read-only also gives it a positive role: it is the
always-current surface of the repository's state — status renders, rules
inspection, browsing — that is never mid-anything.

Per-change worktree cost is negligible in this ecosystem: worktrees share the
object store, and Go's build cache is user-global.

Where the worktree lives is settled elsewhere: a tool never creates directories
in space the user did not hand it (adr-2609091248200336), so a per-change
worktree goes in the machine-scoped store the store draft
(itd-2609091014076309) describes, keyed on the repository's root commit, and
that draft owns the verbs that add, list and clear away worktrees. This intent
is what makes the store the only route: the block in the primary checkout, the
seeding of the protections, and the peer visibility.

## What's In Scope

- **Worktrees come from the store.** This intent adds no verb that creates,
  lists or removes a worktree: the store draft (itd-2609091014076309) owns
  add, list and clean-up, and this intent calls them. Every refusal of the
  block names the store's add verb as the route, and nothing this intent
  writes lands beside or inside the primary checkout.
- **The mutation block**: a host-hook (thin caller; the check lives in the
  binary) that refuses file writes and git mutations in the primary checkout
  of an abcd-managed repo, naming the worktree route instead. The check is
  git-aware (resolved via the tree's git common directory, never a path
  prefix: which checkout a tree is comes from git, wherever it sits) and
  honest about its rung: hook-level interception of the host's file-edit and
  shell tools is a mitigation that catches recognisable writes, not a
  filesystem guarantee. A git-level pre-commit backstop in the primary
  checkout is the second layer, refusing commits there for humans and
  non-hooked tools, subject to the same allow-set.
- **The declared coordinator is exempt.** A session that declares itself the
  coordinator of the sessions working in the repository is exempt from the
  block in the primary checkout, at both layers; every other session is
  blocked. The exemption is never inferred from what a session does: it
  exists only by the session's own declaration. Because it lifts a write
  block, it is a trust path, and how the declaration is made and what stops
  a second session from making it are the spec's, reviewed as security
  work.
- **The read-only surface's allow-set, stated explicitly**: writes to the
  local tier (`.abcd/.work.local/`) and other gitignored or untracked paths
  (build output included), fetch and fast-forward of the default branch (what
  keeps the surface current), and `git worktree` administration are permitted
  in the primary checkout; tracked-file writes and history-moving operations
  are what the block refuses. The ledger write of `abcd capture` joins the
  allow-set as a narrow carve-out (a new timestamp-named file under the issue
  ledger only, so concurrent captures cannot collide), paired with an
  adoption path: a capture written in the primary checkout is uncommitted by
  construction, and the next worktree abcd seeds adopts orphan captures into
  its change so they reach the default branch rather than sitting untracked.
- **Peer visibility as a mechanical duty** (the convention layer): worktree
  entry and exit and record-id mints are recorded such that every peer
  session in the repository sees them at its next hook fire (the
  record-then-inject shape the rules loader already uses; delivery is
  at-next-prompt, not instantaneous). The coordination mechanism — typed
  claims, take or yield or escalate — remains itd-33's; this intent makes
  today's AGENTS.md announce convention mechanical rather than remembered,
  without adding the delivery channel itd-33 deliberately cut.
- **Protection seeding on detection**: a hook fire that finds itself in a
  worktree of an abcd-managed repo whose local tier lacks the name-guard
  layer seeds the pointer to the primary checkout's store, so worktrees the
  host created without abcd get the protection too. abcd-managed repos
  pre-populate the private banlist to the four-category floor above, values
  gathered by one-time setup prompt or from the user-level home, never
  derived silently and never committed.
- **Scaffolded to all abcd-managed repos** via the prepare and ahoy path.

## What's Out of Scope

- **Adding, listing and clearing away worktrees**: the store draft
  (itd-2609091014076309) owns all three. The two refinements this intent's
  2026-08-26 interview ruled for its own sweep — a merge proven by the
  forge's recorded pull-request state or by patch-equivalence where squash
  and rebase merges leave no ancestry, working without a forge; and an
  unmerged worktree inactive for 14 days surfaced with a dossier (dirty
  state first) and removed only on per-item confirmation — are carried to
  that draft's planning interview as input, not built here.
- Post-merge residue mechanics (remote PR branch, local branch, tracking
  ref): itd-118 owns them, and the store's clean-up is what its post-merge
  tidy calls.
- The coordination layer itself (claims, yield, escalation): that is itd-33,
  whose revisit triggers have fired and which owes a SOTA sweep first
  (iss-2608230943533581). This intent adds no delivery channel and no
  agent-to-agent negotiation.
- Session-presence leases for two sessions in ONE checkout
  (iss-2608220750029993): narrowed but not closed by this intent; it stays
  open.
- **Met elsewhere since the 2026-08-26 interview**, so no longer this
  intent's: the guard layer inside a linked worktree (iss-370, resolved by
  itd-150); the AGENTS.md concurrency rule restated by blast radius
  (iss-2608230957104179); push-time gates reading the committed tree through
  the preflight receipt (iss-2608210738378295); and the timestamp id mint for
  every record family (adr-45, iss-2608210737260468).

## Scope Conditions

None stated.

## Acceptance Criteria

> _Rewritten on 2026-09-29 to the product thinker's rulings (`## Decisions`):
> the sweep's two criteria moved to the store draft, and the coordinator's
> criterion was added. The criteria are walked again after the store draft's
> planning interview, before anything is built._

- **Given** a session in the primary checkout of an abcd-managed repo that
  has not declared itself the coordinator, **when** it attempts a
  tracked-file write or a history-moving git operation through a hooked
  surface, **then** the block refuses with a message naming the store's add
  verb as the route, and read-only operations and allow-set writes are
  untouched.
- **Given** a session that has declared itself the coordinator, **when** it
  writes or commits in the primary checkout, **then** neither the hook nor
  the pre-commit backstop refuses it; **and given** a second session that has
  not declared itself, **when** it does the same, **then** it is refused.
- **Given** the store adds a worktree, or a hook fire detects a
  worktree without the guard layer, **when** the session works
  there, **then** the private name-guard layer is active in that worktree,
  pre-populated to at least the four floor categories with values that appear
  in no committed file, **and** the entry is recorded such that every live
  peer session sees it at its next hook fire.
- **Given** a session refused by the block, **when** it takes the route the
  refusal names, **then** its worktree is under `~/.abcd/worktrees/<root-sha>/`
  and neither the primary checkout nor its parent directory holds anything it
  did not hold before.
- **Given** a record-id mint from any worktree while a peer session is live,
  **when** the mint runs, **then** the family and checkout are recorded for
  peer visibility at next hook fire.
- **Given** a repo where no worktree exists and no peer runs, **when** a
  read-only verb runs in the primary checkout, **then** nothing about its
  behaviour or cost has changed, and no bare invocation acquires a write as
  a side effect of the block or the seeding.

## Decisions

Ruled by the product thinker on 2026-09-29 (interview by orchestrator abcd-23
of autonomous run A; recorded in `.abcd/work/DECISIONS.md` under that date):

1. **The store under the home folder wins** (adr-2609091248200336): a
   per-change worktree lives in the store, never inside or beside the
   checkout; this record is rewritten to match.
2. **The store draft owns add, list and clean-up** (itd-2609091014076309);
   this intent uses them and builds no worktree verb of its own.
3. **A session that declares itself the coordinator is exempt** from the
   write block in the primary checkout; every other session is blocked.
4. **The store draft is planned first**: nothing from the store draft, the
   merged-worktree clean-up included, is built before its planning
   interview. This intent's own wait is not part of the ruling: it follows
   from decision 2, because this intent uses the store's verbs, and is
   carried by its `blocked_by` on the store draft.

## Open Questions

- **How does a session declare itself the coordinator, and what stops a
  second one?** Decision 3 rules that the exemption exists; its mechanism (a
  flag, a mode, a local-tier file) and the refusal of a second declaration
  are the spec's, to be settled after the store draft's planning interview.
- **Does the mint-visibility criterion still earn its place?** It was the
  bridge until every record family minted timestamp ids, and that migration
  has since landed (adr-45): AGENTS.md now says record ids need no
  coordination between checkouts. Keep it as peer awareness, or drop it at
  the next walk of the criteria.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## References

- Resolved by shipping: iss-2608230847432285, iss-213. The three others
  this list once named (iss-370, iss-2608230957104179, iss-2608210738378295)
  were resolved by other changes (see Out of Scope).
- The store this intent is blocked by: itd-2609091014076309, and the rule it
  enacts, adr-2609091248200336.
- Adjacent intents: itd-118 (merged work leaves no residue: consumed),
  itd-33 (coordination mechanism: this intent carries only the
  record-then-inject visibility duty), itd-115 (merge without churn),
  itd-107 (dispatches subagents into per-worktree isolation and asks the
  independent-peers question this intent answers).
- Sequencing: adr-45 ruling 3 and iss-2608210737260468 (the timestamp mint,
  since landed), with the collision recurrence iss-2608221126066632 and the
  wontfix sibling iss-2608220150157512.
- Host prior art: the Claude Code harness's own worktree isolation
  (per-session worktrees inside the checkout, base-ref policy). This intent
  is the host-agnostic, abcd-owned form of that behaviour, with the worktree
  in the store rather than inside the checkout.
