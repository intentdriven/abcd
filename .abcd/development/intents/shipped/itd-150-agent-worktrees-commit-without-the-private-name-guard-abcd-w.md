---
id: itd-150
shipped_in: v0.6.8
slug: agent-worktrees-commit-without-the-private-name-guard-abcd-w
spec_id: spc-43
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-370]
impact: fix
---

# Agent worktrees commit without the private name-guard: .abcd/.work.local/ is per-worktree, so every isolated-worktree agent commit runs with the banlist layer absent — loudly warned, per design, but the isolated-agent pattern now systematically bypasses a protection the main checkout has. Candidate remedies: the worktree-creation path seeds a pointer to the primary checkout's store, or the hook falls back to reading the primary worktree's local tier

## Press Release

> _Seeded by promotion from iss-370. Expand into the full press-release narrative before planning._

## Why This Matters

Graduated from `iss-370`: Agent worktrees commit without the private name-guard: .abcd/.work.local/ is per-worktree, so every isolated-worktree agent commit runs with the banlist layer absent — loudly warned, per design, but the isolated-agent pattern now systematically bypasses a protection the main checkout has. Candidate remedies: the worktree-creation path seeds a pointer to the primary checkout's store, or the hook falls back to reading the primary worktree's local tier. Read that issue record for the source observation.

## Scope Conditions

None stated.

## Acceptance Criteria

- **Given** a name recorded in the private banlist of the primary checkout's `.work.local` store but absent from any store inside a linked git worktree, **when** an agent stages a commit whose content carries that name from within the linked worktree, **then** the guard hook resolves the banlist from the primary checkout and blocks the commit on that name.
- **Given** a freshly created linked worktree with no private banlist file of its own, **when** the guard hook runs there, **then** it enforces the same banlist as the main checkout without any per-worktree banlist setup having been performed.
- **Given** a name banned in the linked worktree's own local store, **when** a commit runs in that worktree, **then** the guard still blocks it, so the primary store is a fallback and never overrides a worktree-local entry.
- **Given** the guard runs in a standalone checkout that is not a linked worktree, **when** no primary-checkout store can be resolved, **then** it behaves as it does today and emits the existing loud warning rather than failing closed on a resolution error.

## Open Questions

_None recorded yet._

## Audit Notes

<!-- abcd-review: INGESTED receipt=rcp-4b7ee348ffdd -->
Fidelity review — receipt rcp-4b7ee348ffdd (verifier intent-auditor claude-fable-5-1).

Provenance: intent-auditor@claude-fable-5-1 · rubric_hash sha256:effa65b3e9e88ff29433b443ec2be159522a8b0b71cf1434526514aa61edb13e · prompt_hash sha256:d11a1ac6e32a935bec2f2bde66eb8d8a999c960bef0a60ba5b3d4ed5fd164e3c
Input attestations: diff:328a6755^1..328a6755 (PR #555), judged against the tree at bad1c73e@-;

Acceptance rollup: MET 3 · MET_WITH_CONCERNS 1 · NOT_MET 0 · INCONCLUSIVE 0

Per-criterion verdicts:
- ac-1 — MET: the scaffolded hook resolves the primary checkout via --git-common-dir plus `git worktree list`, loads its store as a fallback layer, and the linked-worktree test blocks on a name only the primary store bans
  evidence: internal/core/ahoy/defaults/pre-commit:198 — "common_dir=$(git rev-parse --path-format=absolute --git-common-dir"
  evidence: internal/core/ahoy/defaults/pre-commit:247 — "primary_banlist="$primary_root/$banlist""
  evidence: internal/core/ahoy/defaults/pre-commit:656 — "load_store "$primary_banlist" "$primary_label" 1"
  evidence: internal/core/banlist/hook_test.go:885 — "func TestPreCommitHook_LinkedWorktreeInheritsThePrimaryStore"
- ac-2 — MET: store_present is true when either store exists, and the test fixture asserts the linked worktree carries no local tier at all before the block is observed
  evidence: internal/core/ahoy/defaults/pre-commit:412 — "if [ -n "$primary_banlist" ] && [ -f "$primary_banlist" ]; then"
  evidence: internal/core/banlist/hook_test.go:888 — "the linked worktree already carries a local tier; the fixture proves nothing"
- ac-3 — MET_WITH_CONCERNS: the two stores are loaded as a union (primary first, local second) so a local entry blocks and the primary never displaces it; concern: spc-43 described a duplicate-key local-wins override, and the hook deliberately delivers a union instead, documented in place as the safer direction
  evidence: internal/core/ahoy/defaults/pre-commit:453 — "The two stores are a UNION, never an override."
  evidence: internal/core/ahoy/defaults/pre-commit:659 — "load_store "$banlist" "$banlist" 0"
  evidence: internal/core/banlist/hook_test.go:912 — "func TestPreCommitHook_LinkedWorktreeStoreWinsOverThePrimary"
- ac-4 — MET: resolution failure leaves primary_banlist empty and the hook takes the existing INACTIVE warning branch; the pre-existing warning tests and the no-store-anywhere worktree test all pass at BASE
  evidence: internal/core/ahoy/defaults/pre-commit:190 — "Resolution NEVER fails the commit."
  evidence: internal/core/ahoy/defaults/pre-commit:419 — "WARNING — the private name guard is INACTIVE on this machine."
  evidence: internal/core/banlist/hook_test.go:155 — "func TestPreCommitHook_AbsentBanlistWarnsLoudly"
  evidence: internal/core/banlist/hook_test.go:965 — "func TestPreCommitHook_LinkedWorktreeWithNoStoreAnywhereStillWarns"

Gap audit:
- honoured:
  - the guard hook resolves the primary checkout's store from a linked worktree, asking git which tree is primary rather than path arithmetic
    evidence: internal/core/ahoy/defaults/pre-commit:223 — "worktree_list=$(git worktree list --porcelain"
    evidence: internal/core/banlist/hook_test.go:987 — "func TestPreCommitHook_BareRepoWorktreeDoesNotInheritASiblingStore"
  - abcd's own committed hook carries the same resolution as the scaffolded default
    evidence: .githooks/pre-commit:198 — "common_dir=$(git rev-parse --path-format=absolute --git-common-dir"
    evidence: .githooks/pre-commit:247 — "primary_banlist="$primary_root/$banlist""
  - CLI parity: `abcd banlist` renders the inherited layer the guard enforces
    evidence: internal/core/banlist/worktree.go:78 — "func PrimaryWorktreeRoot(repoRoot string) (string, bool)"
    evidence: internal/surface/cli/banlist.go:84 — "inherited, err := banlist.InheritedPrivate(root)"
    evidence: internal/core/banlist/worktree_test.go:137 — "func TestListCarriesTheInheritedLayer"
  - a standalone checkout keeps today's loud warning instead of failing closed
    evidence: internal/core/ahoy/defaults/pre-commit:416 — "if [ "$local_present" -eq 0 ] && [ "$primary_present" -eq 0 ]; then"
- diverged:
  - spc-43 said a duplicate key from the local store overrides the primary entry; the delivered hook merges the two stores as a union with no override, so both patterns are enforced (strictly more refusals, never fewer)
    evidence: internal/core/ahoy/defaults/pre-commit:453 — "The two stores are a UNION, never an override."
    evidence: .abcd/development/specs/closed/spc-43-agent-worktrees-commit-without-the-private-name-guard-abcd-w.md:60 — "a duplicate key from the local store overrides the primary entry"
- missing: (none)