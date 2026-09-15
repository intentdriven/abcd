---
id: itd-148
slug: every-change-starts-in-its-own-worktree-the-primary-checkout
spec_id: spc-42
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-118, itd-33]
severity: major
impact: additive
---

# Every change starts in its own worktree: the primary checkout is a read-only surface, abcd blocks mutations there, and stale worktrees are swept with a dossier the human can decide on

## Press Release

> **abcd turns the primary checkout into a read-only surface: every change — a one-line docs fix included, peers present or not — begins in its own worktree, and abcd enforces the boundary mechanically.** The worktree verb family (CLI and plugin surface) creates and enters a per-change worktree, records the entry where every peer session sees it on its next prompt, and seeds the worktree with the protections the primary checkout has — the private name-guard layer included, pre-populated to a floor of four categories (home/user paths, machine hostnames, personal email, real surname), each value held on the machine and never committed. A host-hook backed by the binary refuses file writes and git mutations attempted in the primary checkout, so a session whose working directory silently reverts to the shared tree is stopped at its first recognisable write instead of contaminating a tree another session is about to move. A worktree whose change has provably merged is removed at session start with no ceremony; a worktree that went quiet without merging is never silently deleted — abcd names the candidates and the sweep hands the human a dossier per worktree (whether it holds uncommitted work, what the branch was about, the records it references, its last activity) plus a recommendation, so the decision is informed and one keystroke rather than archaeology.
>
> "I had two sessions going and asked one for a two-line docs fix, and it just did it — in its own worktree, visible to the other session, PR up, and days later the worktree was simply gone because the merge was provable," said Kira, an open-source maintainer running concurrent agents. "The worktree left over from an abandoned experiment didn't rot either: at my next session start they showed me it was clean, what it had been about, and a suggestion to remove. I pressed yes. I never had to reconstruct what a stale branch was."

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

## What's In Scope

- **The worktree verb family**, wired to CLI and plugin surface: create/enter
  a per-change worktree; a status render answering which worktrees exist, who
  is in them, and what each is about; the sweep.
- **The mutation block**: a host-hook (thin caller; the check lives in the
  binary) that refuses file writes and git mutations in the primary checkout
  of an abcd-managed repo, naming the worktree route instead. The check is
  git-aware (resolved via the tree's git common directory, never a path
  prefix, since worktrees physically live inside the primary checkout) and
  honest about its rung: hook-level interception of the host's file-edit and
  shell tools is a mitigation that catches recognisable writes, not a
  filesystem guarantee. A git-level pre-commit backstop in the primary
  checkout is the second layer, refusing commits there for humans and
  non-hooked tools, subject to the same allow-set.
- **The read-only surface's allow-set, stated explicitly**: writes to the
  local tier (`.abcd/.work.local/`) and other gitignored or untracked paths
  (build output included), fetch and fast-forward of the default branch (what
  keeps the surface current), and `git worktree` administration are permitted
  in the primary checkout; tracked-file writes and history-moving operations
  are what the block refuses. The ledger write of `abcd capture` joins the
  allow-set as a narrow carve-out (a new timestamp-named file under the issue
  ledger only, so concurrent captures cannot collide), paired with an
  adoption path: a capture written in the primary checkout is uncommitted by
  construction, and the sweep or the next change worktree adopts orphan
  captures into a change so they reach the default branch rather than sitting
  untracked.
- **Peer visibility as a mechanical duty** (the convention layer): worktree
  entry and exit and record-id mints are recorded such that every peer
  session in the repository sees them at its next hook fire (the
  record-then-inject shape the rules loader already uses; delivery is
  at-next-prompt, not instantaneous). The coordination mechanism — typed
  claims, take or yield or escalate — remains itd-33's; this intent makes
  today's AGENTS.md announce convention mechanical rather than remembered,
  without adding the delivery channel itd-33 deliberately cut.
- **Sweep, merged half**: a worktree whose change has provably merged is
  removed at session start or explicit sweep — never as a side effect of
  other verbs, which keeps bare invocations zero-write — consuming itd-118's
  tidy mechanics. Because this repo allows squash and rebase merges, which
  rewrite the branch's shas out of existence, provably-merged means the
  remote PR's recorded merge state where a PR exists, with patch-equivalence
  as the local fallback; plain sha reachability is only the fast path, and a
  worktree whose merge cannot be proven is demoted to the dossier path, never
  guessed at. Operation is remote-optional (ruled at the planning interview):
  without a forge, patch-equivalence alone decides, at the price of more
  dossier demotions.
- **Sweep, abandoned half**: an unmerged worktree inactive past a threshold
  is surfaced at session start and never auto-deleted. The sweep composes a
  dossier per candidate — leading with whether the tree is dirty, then branch
  subjects, diff summary, referenced records (iss-N or itd-N), last activity
  — plus a recommendation whose vocabulary includes capture-or-commit-first
  for dirty trees; removal is per-item human-confirmed, and `git worktree
  remove` refusing a dirty tree without force is kept as the backstop.
- **Worktree protection seeding closes iss-370**: the guard layer is seeded
  on detection, not only on creation — a hook fire that finds itself in a
  worktree of an abcd-managed repo whose local tier lacks the name-guard
  layer seeds the pointer to the primary checkout's store, so worktrees the
  host created without abcd get the protection too. abcd-managed repos
  pre-populate the private banlist to the four-category floor above, values
  gathered by one-time setup prompt or from the user-level home, never
  derived silently and never committed.
- **AGENTS.md Concurrent sessions rewrite** while this intent rewrites that
  ground anyway: the scan-before-mutating rule restated by blast radius, with
  the four git operations and shared build artefacts as examples rather than
  the set (iss-2608230957104179).
- **Push-time gates lint the tree that ships**: the same machinery gives the
  pre-push gate a clean worktree of HEAD to lint, so a working-tree and index
  divergence cannot pass locally and fail CI (iss-2608210738378295).
- **Scaffolded to all abcd-managed repos** via the prepare and ahoy path.

## What's Out of Scope

- Post-merge residue mechanics (remote PR branch, local branch, tracking
  ref): itd-118 owns them; this intent consumes them for the merged half of
  the sweep.
- The coordination layer itself (claims, yield, escalation): that is itd-33,
  whose revisit triggers have fired and which owes a SOTA sweep first
  (iss-2608230943533581). This intent adds no delivery channel and no
  agent-to-agent negotiation.
- Session-presence leases for two sessions in ONE checkout
  (iss-2608220750029993): narrowed but not closed by this intent; it stays
  open.
- The timestamp id mint for the sequential families: adr-45 ruling 3 already
  adopts timestamp ids and schedules the itd and spc migration after the
  captures family runs one release cycle (iss-2608210737260468). This intent
  does not do that work and does not accelerate the recorded schedule (ruled
  at the planning interview): the mint-visibility duty above is the standing
  bridge until the migration arrives. The live collision evidence is
  iss-2608221126066632 (its sibling iss-2608220150157512 was closed wontfix
  in favour of the migration).

## Scope Conditions

None stated.

## Acceptance Criteria

> _Seeded by the drafting session and revised under two adversarial reviews;
> unconfirmed proposals until the planning interview walks them._

- **Given** a session in the primary checkout of an abcd-managed repo,
  **when** it attempts a tracked-file write or a history-moving git operation
  through a hooked surface, **then** the block refuses with a message naming
  the worktree route, and read-only operations and allow-set writes are
  untouched.
- **Given** abcd creates and enters a worktree, or a hook fire detects a
  host-created worktree without the guard layer, **when** the session works
  there, **then** the private name-guard layer is active in that worktree,
  pre-populated to at least the four floor categories with values that appear
  in no committed file, **and** the entry is recorded such that every live
  peer session sees it at its next hook fire.
- **Given** a worktree whose change's merge is provable (remote PR merge
  state, or patch-equivalence locally; squash and rebase merges therefore
  covered), **when** a session starts or the sweep runs, **then** the
  worktree and its tidy-work (per itd-118) are removed without prompting and
  the removal is reported; a worktree whose merge cannot be proven is
  surfaced on the dossier path instead, never removed on a guess.
- **Given** an unmerged worktree with no open PR and no activity for 14 days
  (the default; repo-configurable), **when** a session starts, **then** it is named as a sweep candidate; **when** the
  human runs the sweep, **then** each candidate presents its dossier (dirty
  state first) and recommendation, and nothing is removed without per-item
  confirmation.
- **Given** a record-id mint from any worktree while a peer session is live,
  **when** the mint runs, **then** the family and checkout are recorded for
  peer visibility at next hook fire: the mechanical form of today's
  convention, standing until the timestamp migration retires the collision
  class.
- **Given** a repo where no worktree exists and no peer runs, **when** a
  read-only verb runs in the primary checkout, **then** nothing about its
  behaviour or cost has changed, and no bare invocation acquires a write as
  a side effect of the sweep.

## Open Questions

_None open. The drafting session's questions (capture carve-out, git-level
backstop, abandoned threshold, severity, adr-45 sequencing, remote-optional
merged proof) were each resolved with the maintainer at the 2026-08-26
planning interview; the resolutions are folded into scope above._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## References

- Resolved by shipping: iss-2608230847432285, iss-213, iss-370,
  iss-2608230957104179, iss-2608210738378295.
- Adjacent intents: itd-118 (merged work leaves no residue: consumed),
  itd-33 (coordination mechanism: this intent carries only the
  record-then-inject visibility duty), itd-115 (merge without churn),
  itd-107 (dispatches subagents into per-worktree isolation and asks the
  independent-peers question this intent answers).
- Sequencing: adr-45 ruling 3 and iss-2608210737260468 (timestamp mint
  schedule), with the open collision recurrence iss-2608221126066632 and the
  wontfix sibling iss-2608220150157512.
- Host prior art: the Claude Code harness's own worktree isolation
  (per-session worktrees under `.claude/worktrees/`, base-ref policy). This
  intent is the host-agnostic, abcd-owned form of that behaviour.
