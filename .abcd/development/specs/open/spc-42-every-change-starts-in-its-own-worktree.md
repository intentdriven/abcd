---
id: spc-42
slug: every-change-starts-in-its-own-worktree
intent: itd-148
---
# every-change-starts-in-its-own-worktree-the-primary-checkout

## Summary

Delivers itd-148: the primary checkout of an abcd-managed repo becomes a
read-only surface for every session but a declared coordinator, and every
change starts in a worktree the store under the home folder holds
(itd-2609091014076309, adr-2609091248200336). All behaviour lives in the
transport-agnostic core; the CLI and the plugin markdown surface are thin
front doors, and the enforcement hooks are thin callers into the binary.

**Rewritten on 2026-09-29 to the product thinker's rulings** (itd-148
`## Decisions`). This spec no longer mints `abcd worktree new <slug>` or
`abcd worktree sweep`, and it no longer composes a sweep of its own: the
store draft owns adding, listing and clearing away worktrees, and this spec
calls those verbs. It **reverses** the scope of the 2026-08-26 planning
interview in exactly that respect; the block, the allow-set, the seeding and
the peer visibility stand.

## Blocked on

- **The store draft's planning interview** (itd-2609091014076309). Ruling Q4
  of 2026-09-29 plans the store draft first: nothing from the store draft,
  the merged-worktree clean-up included, is built before its planning
  interview. This spec waits on it because it calls the store's verbs
  (ruling Q2), which is what itd-148's `blocked_by` on the store draft
  carries.
- **The store's add, list and clean-up verbs shipping**, because the
  refusal names the add verb as the route and the seeding runs on the
  worktrees the store adds.
- **A walk of the criteria after the store is planned**, which settles how a
  session declares itself the coordinator (itd-148's open questions).

## Scope

1. **Core package** (`internal/core`): tree classification (primary vs
   worktree, resolved via the git common directory, never a path prefix),
   the coordinator declaration and its check, and the visibility ledger.
   Worktree creation, listing and removal are the store's, not this
   package's.
2. **Verb**: the hook entry point, a `check` sub-verb beside the store's
   worktree verbs (its final name follows the store's naming, one of that
   draft's open questions); the exit code carries the block verdict. It
   performs zero writes.
3. **Hooks**: a PreToolUse-shaped hook routing the host's file-edit and shell
   tools through the check; a git pre-commit backstop installed in the
   primary checkout via the ahoy defaults; both honour the coordinator
   exemption.
4. **Scaffolding**: the prepare/ahoy path installs the hooks and the
   pre-populated private banlist floor into abcd-managed repos.
5. **Record changes**: resolution of iss-213 and iss-2608230847432285 in the
   shipping changes. The three other issues this spec once bundled
   (iss-370, iss-2608230957104179, iss-2608210738378295) were resolved by
   other changes.

## Approach, by acceptance criterion

**AC 1 — the mutation block with allow-set.** The check receives the tool
input (cwd, file path or command) and answers allow/refuse. Refusal only
when: the tree resolves to the primary checkout of an abcd-managed repo, the
session is not the declared coordinator, AND the write targets a tracked
file or the command is history-moving. The allow-set is explicit:
gitignored/untracked paths, `.abcd/.work.local/`, fetch/fast-forward of the
default branch, `git worktree` administration, and the capture carve-out (a
new timestamp-named file under `.abcd/work/issues/open/` only). Shell
interception is command parsing at the guard-registry rung, stated as
mitigation, not filesystem guarantee; the pre-commit backstop (same check,
commit-time) catches what parsing misses. Refusal messages name the store's
add verb. Tests: table-driven check tests per allow/refuse case; hook wiring
exercised via the smoke harness.

**AC 2 — the declared coordinator.** A session is the coordinator only by
its own declaration, never by inference; the check allows the coordinator at
both layers and refuses every other session. The declaration's form and the
refusal of a second declaration are settled at the walk named under Blocked
on; the lane is security-reviewed, since the declaration lifts a write
block.

**AC 3 — guard seeding and peer visibility on entry.** Seeding runs on
detection, not only creation: any hook fire that finds itself in a worktree
of an abcd-managed repo whose local tier lacks the name-guard layer seeds the
pointer to the primary checkout's store. The floor's four categories
(home/user paths, machine hostnames, personal email, real surname) are
populated by one-time setup prompt or from the user-level home; values are
machine-local, never derived silently, never committed. Entry/exit events
append to a visibility ledger in the primary checkout's local tier;
peer-session hook fires inject unseen entries (the record-then-inject shape
the rules loader already uses). Delivery is at-next-prompt by design.

**AC 4 — the route lands in the store.** The refusal's route is the store's
add verb; a test follows it from a refused write and asserts the worktree is
under the store and nothing new sits in or beside the primary checkout.

**AC 5 — mint visibility.** Record-id mints append a family-and-checkout
event to the same visibility ledger, injected to peers at next hook fire
(kept or dropped at the walk named under Blocked on; see itd-148's open
questions).

**AC 6 — zero cost when idle.** The check and bare verbs stay zero-write. A
repo with no worktrees and no peers takes one tree-classification stat call
in the hook path and nothing else. Read-only verbs are untouched by
construction (the block sits on write paths only), verified by tests
asserting no writes from bare invocations.

## Out of scope

Adding, listing and clearing away worktrees (the store draft,
itd-2609091014076309, including the merge proof and the abandoned-worktree
dossier this spec once carried), coordination claims (itd-33), post-merge
remote/branch residue mechanics (itd-118), presence leases
(iss-2608220750029993).

## Delivery

Staged PRs after the store ships, each preflight-clean with tests watched
fail first: (1) core classification + the check + hooks; (2) the coordinator
declaration (security review); (3) seeding + visibility ledger; (4)
scaffolding + the bundled-issue resolutions with `Resolves:` trailers in
their fixing changes.
