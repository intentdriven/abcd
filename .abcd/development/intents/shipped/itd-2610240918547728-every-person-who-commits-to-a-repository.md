---
id: itd-2610240918547728
slug: every-person-who-commits-to-a-repository
spec_id: spc-2610241007568207
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-2610240823055889]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# Every contributor commits as themselves

## Press Release

> Every person who commits to a repository abcd manages commits as themselves. Setup and the commit check know a person by their own identity, never by someone else's: a second contributor is never told to commit under the maintainer's name, a commit made under a bot's or an AI tool's name is still refused, and a mistyped email of one's own is still caught.

## Why This Matters

_Split out of itd-2610240823055889 by the product thinker on 2026-10-24; drafted by the agent, not yet confirmed._

On 2026-10-24 the product thinker asked whether a pinned identity would confuse anyone else working on abcd. It does: the repository commits one person's identity, so setup reports any other contributor's identity as a required problem and offers to make them commit as the maintainer, and the commit check refuses their commits with a fix line naming the maintainer (iss-2610240830397793). The pin was meant to catch commits under a bot's name or a mistyped email; it should keep doing that without ever naming another person.

## Mechanism

- We expect no check to ever propose someone else because each person's record names only themselves, while a wrong email of one's own is still caught.


## Scope Conditions

- Any number of people committing to one repository. <!-- cond: cond-2610241007563822 -->
- One identity per person per machine. <!-- cond: cond-2610241007565826 -->
- macOS and Linux. <!-- cond: cond-2610241007567901 -->
- Attended and unattended runs. <!-- cond: cond-2610241007564932 -->


## Acceptance Criteria

_Each criterion confirmed by the product thinker, one at a time, on 2026-10-24._

- Given a person who has not recorded who they are on this machine, when setup runs at a terminal, then it proposes their usual git identity and records it on their machine on a yes.
- Given a contributor who is not the maintainer, when they set up and commit, then nothing names the maintainer, and their commits are checked against their own record.
- Given a commit under a different email from the one the person recorded, or under a bot's or an AI tool's name, when it is committed, then it is refused, naming the person's own record and how to change it.
- Given a repository that stores an identity today, when setup runs after the update, then it moves that identity to its owner's machine, removes it from the repository, keeps the repository's other settings, and says so.
- Given an unattended run on a fresh machine with no identity recorded, when it would commit, then it uses the human identity whatever launched it supplied, or is refused, never committing as the tool.


## Open Questions

- resolved: Decision 2026-10-24 on more than one contributor reverses part of shipped itd-131, which checks commits "against the identity the repo pins"; the reviews also found that a pin kept on each machine leaves an autonomous routine in a fresh cloud container with no human identity to establish, and that the same committed file carries the repository's `production_mode`. Confirm the reversal, or choose instead to drop the pin and check only that the author is a human and matches the committer. (raised 2026-10-24 by the design and record reviews; flagged for the product thinker, never auto-classified) (answered 2026-10-24; see Decisions)

## Decisions

- 2026-10-24, the product thinker: setup works for more than one person committing to a repository. The commit identity is each person's own, kept on their own machine, and nothing in the repository names who may commit; no contributor is ever told to commit as someone else (iss-2610240830397793).
- 2026-10-24, the product thinker, confirming the reversal of part of shipped itd-131: each person records their own identity once, on their own machine, and every commit is checked against that record, not against an identity the repository names; the alternative of dropping the pin was considered and declined, because the record catches a wrong email of one's own. Still owed by this plan: how an autonomous routine on a fresh machine gets its human identity before its first commit, and keeping the repository's `production_mode` committed when the identity leaves the repository.

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-4c9b5e516397 -->
Fidelity review OWED (receipt rcp-4c9b5e516397).
<!-- abcd-review-end receipt=rcp-4c9b5e516397 -->

## Grounds

- pursued: we expect a second contributor and an account without admin rights to set abcd up with no surprises; it is shown wrong if either still hits a step they cannot do.
