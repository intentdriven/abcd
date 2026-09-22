---
id: itd-2609221842494980
slug: a-dependency-bump-lands-without-a-person-re-authoring-it-a
spec_id: spc-2609221843355061
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-92]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
related_intents: [itd-2609150819432059, itd-149]
related_adrs: [iss-2609221820487644]
---

# A dependency bump lands without a person re-authoring it, and no machine enters the contributor graph

## Press Release

> **A bot-opened dependency bump is re-authored as the repository's owner when its diff is only a manifest and its lock file, so it merges on its own; the attribution gate is untouched and everything else is left alone.**
>
> "Every bump sat blocked with auto-merge armed, looking as though it wanted a review no reviewer could give," said a product thinker who had just landed one by hand at the end of a long day. "Now the workflow re-authors the lock-file bumps as me, the message says which bot proposed it and which workflow re-authored it, and anything that is not a bump is left exactly where it was."

## Why This Matters

The attribution gate refuses a machine in the author role, on two independent signals, because the contributor graph is built from the author and committer fields and a squash merge re-appends a mis-identified branch author as a co-author. The consequence is deliberate and written down: a bot's pull request is not mergeable as authored. What was not intended is the cost of it, paid on every bump in every ecosystem: the pull request sits with auto-merge armed and every other check green, showing a blocked state that reads as a missing review, and the only way through is a person re-authoring the same two-line diff. On 2026-09-22 that cost a hand-authored branch and two full gate runs for one patch version (iss-2609221820487644). The product thinker ruled the route: re-author the bump before the gate runs, and change the gate not at all.

## Mechanism

We expect a bounded re-authoring to end the hand-landing of bumps without letting a machine into the contributor graph, because the bound is the shape of the diff rather than trust in the bot; shown wrong if a commit outside the bound is ever re-authored, or if bumps still need a person after it ships.

## Scope Conditions

- Holds where the forge marks a branch as the bot's and the bump's diff is confined to a manifest and its lock file; a bump that edits anything else is a person's. <!-- cond: cond-2609221843355728 -->
- Holds for the ecosystems the repository declares; one abcd does not know is left alone rather than guessed at. <!-- cond: cond-2609221843350672 -->
- Holds where the repository owner accepts that a workflow re-authors on their behalf under that bound: the authorship claim is the owner's, made once in the configuration rather than per commit. <!-- cond: cond-2609221843355081 -->

## What's In Scope

- **The bound**: a commit is re-authored only when its branch is one the forge marks as a declared bot's AND its diff touches nothing but that ecosystem's manifest and lock file (`go.mod` with `go.sum`, `package.json` with `package-lock.json`, `pyproject.toml` with its lock, `Gemfile` with `Gemfile.lock`, and their kin). Every other diff, author or bot is left untouched, and the run says which test the commit failed.
- **The re-authoring**: the commit is replayed with the repository owner as author and committer, its message naming the bot that proposed the bump and the workflow that re-authored it, and carrying `Assisted-by: None`, since no model wrote it; the pull request is updated to the re-authored commit.
- **The gate, unchanged**: `scripts/check-attribution.sh` is not edited; it passes the re-authored commit for the reason it always would, and still refuses the bot's original.
- **The declaration**: the repository names the bots and the ecosystems that qualify; an unlisted bot or an unknown ecosystem is left alone.
- **The scaffold**: the same verb that writes a managed repository's release workflows writes this one, opt-in per repository; abcd's own repository carries it.
- **The record**: every re-authoring is written where the repository can see it, naming the bot, the bump and the resulting commit.
- **Security review** before it ships: it rewrites authorship and pushes to a branch under the repository's own credentials.

## What's Out of Scope

- Any change to the attribution gate's rules or its refusals.
- Merging the bump: the queue and the repository's own rules decide that as they do for any pull request.
- Choosing which dependencies to take, or reviewing what a bump contains.

## Decisions

Ruled by the product thinker on 2026-09-21, in the interview that filed and planned this intent (iss-2609221820487644 records the vocabulary rulings it rests on):

1. Route 2 of the three the capture named (ruled 2026-09-22): re-author before the gate runs; the gate itself does not change.
2. The bound is the diff's shape, and it covers every ecosystem dependabot opens, not Go alone.
3. The message names the bot and the workflow, and the trailer is `Assisted-by: None`.
4. Scaffolded like the release gate, opt-in per managed repository.
5. The authorship point is named rather than assumed: a workflow asserts the owner's authorship with no person in the loop, which the owner accepts once in the configuration, under the bound above.

## Open Questions

_None open._

## Acceptance Criteria

- **Given** a bot-opened bump on a branch the forge marks as that bot's, whose diff touches only that ecosystem's manifest and lock file, **when** the workflow runs, **then** the commit is re-authored as the repository owner and the pull request is updated to it.
- **Given** a commit on such a branch whose diff touches anything else, or a commit by an unlisted bot or a person, **when** the workflow runs, **then** nothing is re-authored and the run names which test the commit failed.
- **Given** a re-authored commit, **when** its message is read, **then** it names the bot that proposed the bump and the workflow that re-authored it, and carries `Assisted-by: None`.
- **Given** the attribution gate, **when** this ships, **then** its script is unchanged, it passes the re-authored commit, and it still refuses the bot's original.
- **Given** an ecosystem or a bot the repository has not declared, **when** a bump arrives from it, **then** nothing is re-authored and the run says so.
- **Given** a managed repository that has opted in, **when** the scaffolding verb runs, **then** the workflow is written beside its release workflows; a repository that has not opted in is unchanged.
- **Given** any re-authoring, **when** the repository is inspected, **then** a record names the bot, the bump and the resulting commit.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: dependabot opens these continually, every one blocks with auto-merge armed and looks like it wants a review, and tonight one cost a hand-authored branch and two full gate runs; we expect the bounded re-authoring to end that without letting a machine into the contributor graph; shown wrong if a commit outside the bound is ever re-authored or bumps still need a person
