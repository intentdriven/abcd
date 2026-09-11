---
id: itd-2609091416304128
slug: capture-resolve-refuses-a-record-the-default-branch-has-alre
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: major
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609091034175565, itd-2609091416295622]
---

# capture resolve refuses a record the default branch has already closed, judged by the last fetch

Typed links: `related_intents` [itd-2609091034175565](itd-2609091034175565-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md) (the claim record this was split from on the maintainer's ruling of 2026-09-09; the promotion trail from iss-2609020716570699 runs through it, and this intent carries the third part of that issue's remedy), [itd-2609091416295622](itd-2609091416295622-a-session-sees-the-records-its-sibling-worktrees-hold-before.md) (the sibling-worktree listing, split out the same day, which covers the local case as this one covers the upstream case). Prose cross-reference, not a typed link, because no schema field carries the relation ([iss-2609091256264547](../../../work/issues/open/iss-2609091256264547-three-of-the-four-mandated-typed-relations-cannot-be-written.md)): [iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md), whose third clause — "capture resolve refusing a record that is already terminal on the fetched origin/main" — this intent is, with the fetch withdrawn.

## Press Release

> **`capture resolve` says no when the default branch has already closed the record, before the push gate has to.** When a session resolves an issue that the local `origin/main` ref — as of the last fetch — already holds under `resolved/` or `wontfix/`, the verb refuses, names the terminal path at that ref and the commit that put it there, states how old the ref is, and writes nothing. `capture wontfix` refuses the same way. The verb performs no fetch: what it knows is what the last fetch brought, and it says when that was, so a stale answer is a legible one. The same judgement is rendered read-only earlier, where it costs nothing: `abcd <record-id>` on an open issue says, before the next moves, that the default branch already holds it terminal, so a session that looks before it fixes learns it before the fix and not after. Nothing about the claim, the lease or presence is involved: this is a comparison of two trees git already holds.
>
> "Two of my open pull requests turned out to duplicate work that had merged the day before, and the first thing that told me was the resolution gate refusing the push," said Maya, an autonomous-development practitioner who runs agent sessions against a shared default branch. "The ref that proved it had been sitting in my checkout since the morning fetch. Now the resolve refuses on the spot and tells me how stale my view is, and the record's own page told me first."

## Why This Matters

On the night of 2026-09-01, two open pull requests from one session duplicated work already merged to the default branch, and the resolution gate — `scripts/check-issue-resolution.sh`, rule RS001 — refused the push, correctly, after the fix was written, tested and reviewed ([iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md)). RS001 already tells the stale-branch shape apart from a record left open or missing: it names a record that is terminal at the base as the case where a rebase is the remedy and "resolve it" is not (iss-2609012023256534). The shape is a comparison between the ledger at the head and the ledger at the base, and the base is a ref the checkout holds locally from its last fetch. Every input to the judgement was on disk when the session started the fix; only the moment of asking was wrong.

The source issue proposed this refusal as the third of three remote-mediated parts, judged "on the fetched origin/main". The claim record it was promoted into corrected the fetch away — a read-adjacent verb must not do surprising network I/O, and a fetch moves refs under the feet of every peer sharing the checkout — and kept the refusal inside a design that also carried a lease, a stamp and a pushed duplicate guard. The two adversarial reviews of 2026-09-09 found the claim's parts unsound as drafted and this part independent of all of them, and the maintainer ruled it its own record. It stands alone: no claim is consulted, no lease is read, nothing is written that is not written today.

One thing is said rather than implied. A refusal at `capture resolve` lands after the fix is written, because `AGENTS.md` requires the resolve in the same change as the fix; what it saves is the commit, the push, the CI minutes and the review, not the fix. The earlier moment is the read-only render: the same comparison on `abcd <record-id>`, which is the surface an agent consults for its next move before it starts one. Both surfaces are in scope so that the answer is available before the work for a session that looks, and enforced after it for one that does not.

## What's In Scope

- **`capture resolve <iss-N>` and `capture wontfix <iss-N>` refuse a record terminal at the local default-branch ref.** The verb reads the ledger at `refs/remotes/origin/main` through git's object store — `git ls-tree` over the two terminal folders, by id in the filename, the test RS001 makes — and refuses when the id is present under either. The refusal names the terminal path at the ref, the ref's commit, and the ref's age; the record stays where it is, no grounds are appended, and nothing is written.
- **The ref's age is stated, and no fetch is performed.** The refusal and the render say when the ref was last brought up to date, from what git records locally; which timestamp is the honest one is an open question below. A checkout with no `origin/main` ref — never fetched, or no remote — proceeds exactly as today, and says in one line that no upstream ref was available to judge by.
- **`abcd <record-id>` on an open issue** renders the same judgement, read-only, before `next_moves`: terminal at `origin/main` as of the stated age, with the terminal path. A record open at the ref renders as today.
- **The two judgements agree by construction.** RS001 is shell and cannot import Go, so the two spellings of "terminal at the base" are pinned to each other by a test that runs both over one fixture, the way the ledger's status-folder list is already pinned between `issueschema.StatusDirs` and the shell gate.
- **`--json`** carries the ref, its commit, its age and the terminal path in the refusal and in the render.

## What's Out of Scope

- **Fetching.** The verb never touches the network. Bringing the ref up to date is the user's `git fetch`, and the age line is what tells them to.
- **The claim, the lease, and any refusal that names a peer session.** Those are the claim record's, with their open questions.
- **A record terminal in a sibling worktree but not on the default branch**, which the sibling-worktree listing renders.
- **Refusing `capture promote` or `intent plan`** on the same ground. Promoting a record the default branch has closed is a different act with a different remedy; whether it should refuse is an open question below, not a scope item.
- **Any change to RS001.** The push gate stays, unchanged, as the last line; this intent moves its answer earlier and leaves it in place.

## Mechanism

We expect a local comparison against the last-fetched default-branch ref to catch the merged-work duplicate before the push because in the case on record every input to RS001's judgement was already in the checkout when the fix began — the ref had been fetched and the ledger at that ref held the record terminal — and RS001 proves that the comparison classifies the shape correctly. It is falsified where the ref is older than the upstream close: the record went terminal on the default branch after the last fetch, the verb passes, and RS001 still refuses at the push; the age line makes that case legible in hindsight, and if it turns out to be the common case rather than the rare one, the answer is not a fetch inside the verb but a fetch the session is told to run.

## Scope Conditions

- Holds only as fresh as the last fetch. A record closed on the default branch after the ref was last updated is not seen here and is caught by RS001 at the push, as today; the age line is the disclosure that makes the miss legible.
- Holds where the checkout has an `origin/main` ref. A checkout that has never fetched, has no remote, or names its remote or default branch otherwise proceeds as today and says the check did not run; what to do about a repository whose default branch is not `origin/main` is an open question, not a silent assumption.
- Holds for the ledger as the default branch commits it: terminal means a file whose name carries the id under `resolved/` or `wontfix/` at the ref, the same test RS001 makes. A record absent at the ref is not terminal; a record open at the ref is not terminal.
- Reads the ref through git's object store, so the judgement is the same whatever branch is checked out, whatever the working tree holds, and whatever is stashed; and it is the same under merge, squash and rebase merges, because it compares trees and not ancestry.
- Holds for repositories where the resolve is the last step of the work, which `AGENTS.md` requires here. The refusal saves the commit, the push, the CI run and the review, never the fix; the fix is saved only by a session that consults the render first.
- Holds offline. The outcome with the network unplugged is byte-identical to the outcome with it connected.

## Acceptance Criteria

- **Given** a record open in this tree and present under `resolved/` at the local `origin/main` ref, **when** `capture resolve <iss-N>` runs, **then** it refuses, names the terminal path at the ref, the ref's commit and its age, the record remains under `open/` byte-identical, no grounds entry is appended, and the exit code is non-zero.
- **Given** the same record, **when** `capture wontfix <iss-N>` runs, **then** it refuses the same way.
- **Given** a record open both here and at the ref, **when** `capture resolve` runs, **then** it proceeds exactly as today, and its output carries no upstream line.
- **Given** a checkout with no `origin/main` ref, **when** `capture resolve` runs, **then** it proceeds as today and says in one line that no upstream ref was available to judge by.
- **Given** the network unavailable, **when** `capture resolve` runs against a record terminal at the ref, **then** the refusal is byte-identical to the refusal with the network available, and no fetch is attempted — the test runs with the remote URL pointed at a path that does not exist.
- **Given** a record terminal at the ref, **when** `abcd <iss-N>` runs on the open copy, **then** the render says the default branch holds it terminal, names the terminal path and the ref's age, before `next_moves`, and offers no resolve move.
- **Given** one fixture repository in which a record is terminal at the base and open at the head, **when** RS001 and the verb's judgement both run over it, **then** both classify it as terminal at the base, and the test fails if either disagrees.
- **Given** `--json`, **when** the refusal or the render fires, **then** the payload carries the ref, its commit, its age and the terminal path.
- **Given** a record whose id appears under `resolved/` at the ref only in a filename with a different slug, **when** the verb runs, **then** it is judged terminal, because the id and not the slug is the record's identity.

## Prior Art

- `scripts/check-issue-resolution.sh` — RS001, which already tells "terminal at the base" apart from the other three shapes at push time (iss-2609012023256534); the classification this intent moves earlier and leaves in place.
- [iss-2609020716570699](../../../work/issues/open/iss-2609020716570699-nothing-tells-an-agent-that-a-record-it-is-about-to-fix-has.md) — the source: its third part is this refusal, proposed with a fetch; the fetch was withdrawn in the claim record and stays withdrawn here.
- `internal/gitutil` — `Run`, `IsAncestor`, the scrubbed environment; the ledger at a ref is one `ls-tree` through it.
- `internal/core/issueschema.StatusDirs` and its shell pin — the precedent for holding a Go spelling and a shell spelling of one ledger fact to each other by test.
- `AGENTS.md` § Definition of done — the rule that the resolve lands in the same change as the fix, which is why the refusal is after the fix and the render is the earlier surface.

## Open Questions

- **Which age to state.** Git records no per-ref fetch time; what it has is the modification time of `FETCH_HEAD` (the last fetch of anything) and the ref tip's commit date (when the upstream last changed, not when we last looked). Stating both is honest and noisy; the interview picks the line.
- **The remote and branch names.** `origin/main` is this repository's convention. Whether a managed repository declares its default branch in `.abcd/config/identity.json`, or the verb asks git for `refs/remotes/origin/HEAD`, is the interview's; until then a repository without the ref gets the as-today path and the one-line notice.
- **Whether `capture promote` refuses on the same ground.** Promoting a record the default branch has closed produces an intent from a finding already answered; the remedy differs from a resolve's, and a warning may fit better than a refusal.
- **Whether the refusal is overridable.** A record deliberately re-opened locally after an upstream close is conceivable; no case is on record, so no flag is proposed.
- **Impact.** Additive on its face — a new refusal and a new line, no existing invocation changed; left unset until the interview judges it.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
