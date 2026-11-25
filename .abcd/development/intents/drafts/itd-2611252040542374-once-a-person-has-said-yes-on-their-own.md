---
id: itd-2611252040542374
slug: once-a-person-has-said-yes-on-their-own
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611251834407931]
severity: minor
refines: [iss-2610290709017008]
impact: additive
origin: researcher-authored
production_mode: hand-written
related_issues: [iss-2611251742484774]
---

# abcd asks for its newest release on a standing yes, and tells the person alone

## Press Release

> Once a person has said yes on their own computer, abcd asks at most once a day whether a newer release exists, and tells them alone. Alice, a product engineer, runs abcd on one copy and never thinks to ask; a fix they need shipped last week. Having given the standing yes once, they read a line at their next session's start, shown to them and not to the agent, that names the release they run, the newer one, and the step that takes it. abcd reads the release's metadata and nothing else: it installs, stages, and runs nothing, and sends no version and no identifier. When the check fails it says it could not tell, and never that they are current. They withdraw the yes with one command, and until they give it abcd asks the network nothing on its own.

## Why This Matters

A session start says nothing when the installed copy is older than the newest published release (iss-2610290709017008): two days after a release was cut, sessions still ran an older one with nothing saying so. The ruling that issue waited on is given (adr-2611251808065076, accepted on 2026-11-25 by dcn-2611251823160636). The comparison of itd-2611251834407931 cannot say it, since only the release origin knows that a newer release exists. This is the second half of the reframing of iss-2611251742484774 (dcn-2611251807556458), split from the first by the product thinker's confirmation on 2026-11-25 (dcn-2611252038550260), and it carries what is left of that issue once the first has shipped.

## Mechanism

We expect a check on a standing yes to reach the person the disk cannot, because the one who never asks is the one a release never reaches, and a daily question that carries nothing about them costs them nothing they did not agree to. Shown wrong if a person who gave the yes starts a session more than a day after a stable release was published and is shown no line, counted from the check's own record of when it last asked and what it was told.

## Scope Conditions

- Holds on the computer where the person gave the yes. No repository file, environment variable, or agent gives it.
- Holds for stable releases. A pre-release is never offered.
- Holds for a notice. Taking the release stays the person's act, and nothing here installs one.
- Holds where the host has a channel to the person alone at a session's start.

## Acceptance Criteria

- **Given** no standing yes, **when** any operation of abcd's runs other than a verb whose meaning is a fetch, **then** it makes no request it does not make today, and the tests that hold the network out pass as they do today.
- **Given** a person at their own terminal, **when** they give or withdraw the yes with the one command, **then** it is kept under abcd's home on that account, and `abcd ahoy` says whether it stands and when abcd last asked. A repository file or an environment variable gives none, and the command refuses in a shell that carries a session's markers, the ones a drive refuses on. The refusal rests on those markers and on nothing stronger: the ADR admits that anything running as the account can write the yes, which is why the yes covers a notice and no more.
- **Given** the yes and more than a day since abcd last asked, **when** a session starts, **then** abcd asks the release origin once for the newest stable release's metadata, over the transport `abcd update` uses, sending no version, identifier, or token, and installing, staging, and running nothing. Within the day it asks nothing, and no hook that runs for each command ever asks.
- **Given** a stable release newer than the copy that answers, **when** the check returns, **then** the person reads one line on the channel to them alone, naming both versions and the step for each copy behind, taken from the list of itd-2611251834407931; the model's context carries nothing of it. Closes iss-2610290709017008.
- **Given** the check fails, runs out of time, or is told something it cannot read, **when** the line is composed, **then** it says abcd could not tell and when it last could, never that the copy is current, and the session starts all the same inside the hook's bound.
- **Given** a newer release was found, **when** the line is shown, **then** nothing is downloaded, staged, or applied by the check or by the line; taking the release is the person's act, by the step the line names (`abcd update` for the binary on the command path). Installing one unasked is neither built nor prepared for here (iss-2611252038564722).
- **Given** a fresh plugin root, **when** the bootstrap provisions its binary, **then** it fetches the release that the version in the root's own manifest names, with the download cache keyed by tag, and makes no request that discovers a release; the opted-in build's request stays and is named in invariant 7. The release mark stands: a root whose manifest names a release older than the mark is refused as an older release is today, and the refusal names both and the person's way back. Closes iss-2611251828533623.
- **Given** the change that adds the request, **when** it lands, **then** invariant 7, the update chapter, the update page, and the two tests that hold the network out (`TestOnlyUpdateCheckTouchesTheNetwork` and `TestUpdateSeamUntouchedByOtherVerbs`) change in it, as the ADR's consequences say.

## SOTA

From the primary sources the first leg of the verdict read on 2026-11-25 ([the verdict](../../research/ideas/488a0aa96ac5de805348635b27036addf15cddc2/2026-11-25-ideate-abcd-checks-for-updates.md), dcn-2611251807556458). Nothing was searched again for this draft, and the independent fit-challenge is owed before the build.

- **A tool's own look for a newer release:** mature. One command-line tool asks at most daily, on a terminal, with an off switch, and does so by default (verified). Here the check is off until the person says yes, and a session's start stands in for the terminal, which a hook has not got.
- **A check that also installs what it checks:** an attack's way in, in one tool's advisory (verified). Reading metadata alone runs none of what it reads.
- **Signed metadata that expires:** the only thing that detects a frozen answer (verified); a mark that never goes down removes a rollback and not a freeze. Not adopted here: the line says when abcd last could tell, which bounds a freeze for a person who reads it and does not detect it.
- **The host's own updating of a plugin:** off by default for a third-party marketplace, and only a user or an admin turns it on (verified). It knows nothing of the binary on the command path, and abcd never switches it.

Path 2: a native check on the transport `abcd update` already has, and no dependency. The seam is the question put to the release origin: a host or a package manager that comes to own every copy and to tell the person answers it in place of abcd, and the standing yes is then withdrawn.

## Open Questions

- **Deferred** Whether `abcd update` brings the copies abcd itself provisioned to one release, as the verdict's second ambition says, is the product thinker's: itd-130 keeps that verb out of every plugin root. Neither this intent nor itd-2611251834407931 carries it, and nothing here needs the answer. It is captured as its own issue before the change that ships this resolves iss-2611251742484774.

## Decisions

1. The product thinker's, on 2026-11-25 (dcn-2611252038550260): the split is confirmed, this is an intent of its own, and the six-day unattended run builds both, itd-2611251834407931 first. The criteria were not listed to them. The criteria, the Mechanism, and the scope conditions are the writing session's, carried from the text that stood under the first's Decomposition, for their review on return; a change then is made through the record. To reverse: keep this as a draft.
2. The product thinker's, on 2026-11-25 (dcn-2611251823160636, adr-2611251808065076): abcd may ask for its newest release on a standing yes, as the ADR's first point bounds it. To reverse: withdraw the ADR's acceptance.
3. The product thinker's, on 2026-11-25 (dcn-2611252038565989): abcd may install a newer release of itself without asking each time where the technical facilitator is set to auto, and not by default. It is in neither this intent nor the first: nothing is built on it yet, iss-2611252038564722 holds it, and the ADR is refined when it is planned. The ADR's third point stands meanwhile, so the yes given here covers a notice and is never read as a yes to installing. The setting it follows is being made the person's own (dcn-2611252040395990).
4. The verdict's (dcn-2611251807556458), carried as it stands, no question put: a notice goes to the person alone and never into the model's context; the check never installs or executes anything, runs in no hook for each command, and sends no version, identifier, or token; abcd never switches the host's own updating; and it never reports a copy as current when the check failed.
5. The writing session's, no question put: the bootstrap's own look for the newest release is removed in this change and not only named in invariant 7, since the ADR's consequences ask for one or the other, the root's manifest already names the release, and iss-2611251828533623 asks for it. To reverse: name it in invariant 7 and leave that issue open.
6. The writing session's, for review, added on writing this out: the release mark in the bootstrap refuses any release older than the newest this account has installed or verified (read in `hooks/bootstrap.sh` on 2026-11-25). Once a root is given the release its own manifest names, a root behind the mark meets that refusal where today it is given the newest release, so the seventh criterion says so and keeps the refusal. To reverse: let a root take the release its manifest names whatever the mark says.
7. The writing session's, no question put: the names of the command and of the kept yes are the spec's, since a name is a technical choice.

## Decomposition

Hand-run four-piece table, 2026-11-25, as the second half of a split. One capability, this intent: abcd asks for its newest release on a standing yes and tells the person alone. The trust rule, what a check may do and who gives the yes, has its home: adr-2611251808065076, accepted, which refines adr-38 for this one request, and brief invariant 7, which names the exception. A second trust rule, installing unasked, is ruled on (dcn-2611252038565989) and is not this intent's: iss-2611252038564722 holds it, and its home is that ADR refined when it is planned. No new stance. The request, the record of when abcd last asked, the command, and the line are plumbing, for the spec and for the brief's ahoy and update chapters (`04-surfaces/01-ahoy.md`, `04-surfaces/21-update.md`). Typed links: `builds_on` itd-2611251834407931, whose list gives the step for each copy; `related_issues` iss-2611251742484774; `refines` iss-2610290709017008, written by the match at filing and not confirmed by a person, which the fourth criterion closes. Flagged for the product thinker and never auto-classified: it `reverses` in part itd-130, whose scope puts a nudge at a session's start out unless adr-38 is superseded in the open, on the ground of the accepted ADR, which refines adr-38 and does not supersede it; and itd-2610031026190632 holds the product thinker's answer of 2026-10-03 that a daily look for the host's version goes out only when asked, which this does not touch. Verdict: FILE-AS-IS, the split and its routing confirmed by the product thinker (dcn-2611252038550260).

What ships with it. The change that ships this resolves iss-2611251742484774, iss-2610290709017008 and iss-2611251828533623, each with a `Resolves:` trailer. The first of the three is in `related_issues`, so the issue-drift gate refuses the shipped intent until it is in `resolved/`; it is resolved only once the deferred question above is captured. iss-2610241228359748 stays open for its ruling on precedence, and iss-2611252038564722 for installing unasked.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
