---
id: itd-2611251834407931
slug: keep-every-abcd-copy-current
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
related_issues: [iss-2611251742484774]
origin: extracted-from-record
production_mode: hand-written
---

# Every copy of abcd on a computer is known and compared, with the step that mends each

## Press Release

> abcd now knows every copy of itself on a computer, says which one answered, and says when they differ, with the one step that mends each. Alice, a product engineer, takes a plugin update in one project and believes it is in force. At their next session's start a line shown to them, and not to the agent, says that the session runs the copy at user scope, which is older, names the newer copy at project scope, and gives the host's own command that brings the older one level. `abcd ahoy` and the version line list the same copies: the plugin at each scope, the binary on the command path, and a build from source where there is one, each with its version and a mark on the one answering. Nothing is asked of the network for this. The comparison reads the disk, so it says that the copies agree or differ, and never that they are the newest.

## Why This Matters

Graduated from iss-2611251742484774, which was judged as an idea and reframed on 2026-11-25 (dcn-2611251807556458). This draft carries the first half of that reframing, and the Decomposition holds the second. On 2026-11-25 a plugin update taken at project scope left every session in that project on the user-scope copy, five releases older, while `abcd ahoy` said only that the plugin root was resolved (iss-2611251739422396). The same split was recorded on 2026-10-24 and again on 2026-11-21 (iss-2610241228359748). One binary in one plugin root gave three answers about its own staleness, by the data directory the environment named (iss-2611251806207803). The staleness check that exists (itd-111) reads the disk and does not compare the copies on one computer, and `abcd update` (itd-130) swaps the binary on the command path, refuses a plugin root, and says nothing of the copies it left. The verdict rests on this: agreement among the copies is the durable core, whatever is ruled about looking for a release.

## Mechanism

We expect a comparison from disk to end the unseen split because each of its inputs is already a file: the root a hook is handed names the copy that answers, each plugin root carries a manifest naming its version, the host records what it installed at each scope, and abcd records the binary it put on the command path. Nothing is inferred and nothing is asked of the network. Shown wrong if, on a computer whose copies stand at two versions, a session starts and the person's line does not name the copy that answers and the one that differs, or if the list gives a copy a version that neither its manifest nor its binary reports. Counted on the two layouts on record, those of 2026-10-24 and 2026-11-25, rebuilt as fixtures.

## Scope Conditions

- Holds on one computer and one operating-system account. Copies under another account, on another machine, or in a container are not looked for.
- Holds for the host configuration the session or the command runs under. Copies the host recorded under another configuration directory of the same account, the separate login of itd-2611251725203320 among them, are not read, and the list says so in one line.
- Holds for a host that records on disk the plugin copy it installed at each scope, in a form abcd can read. Where it does not, those copies are said to be unread and are not guessed.
- Holds for copies in the places abcd knows: the plugin roots the host installed, the binary abcd put on the command path, a build of abcd's own source in its own checkout, and the build a person opted into for the hooks. A copy put anywhere else by hand is found only when it is the one answering.
- Holds without the network. The comparison says agree or differ among what is installed, and never which release is the newest; that is the second intent's.
- Holds for reporting. abcd changes no copy it does not own and runs no command of the host's.

## Acceptance Criteria

- **Given** a computer holding more than one copy of abcd, **when** a person runs `abcd ahoy` or asks for the version in a terminal, **then** they read one line for each copy: what it is (the plugin at user scope, the plugin at project scope for the project the command runs in, the binary on the command path, a build from source), its version, and a mark on the one that is answering. The lines the version block prints today and the `superseded_root` note stay as they are, and the list is added beneath them. Copies at project scope for other projects are counted and not named, and `--json` carries the same list and the same count and no more, since an agent reads it. Closes the naming half of iss-2611251739422396, and delivers the reporting half of iss-2610241228359748, which stays open for its ruling.
- **Given** copies at more than one version, **when** the list is shown, **then** the person reads which copies differ from the newest one installed and, for each, the one step that mends it: for a plugin copy the host's own update command naming that scope, as the installed host's own help gives it, with the note that a new session is needed before it loads; for the binary on the command path, `abcd update`. abcd prints the step and runs none of the host's. Where the host's help shows no update command that takes a scope, the step is said in words and no command is printed. One live check on the installed host, with a dated receipt in the local tier naming the host's version, shows the printed command is the one its help gives. Closes the rest of iss-2611251739422396.
- **Given** a session that starts on a copy older than another installed one, **when** the session-start hook runs, **then** the person reads one line, on the channel the host shows the person and not the model, naming the copy that answers, the newer one, and the step. What the hook hands the model carries no such notice and no command. The line is shown at each start while the copies differ, and the hook writes nothing for it. It reads the disk and discovers nothing: no release is looked for, so it is not the session-start nudge itd-130 puts out as ambient discovery. With the two criteria above, delivers the first half of iss-2611251742484774, which stays open for the second.
- **Given** copies that all report one version, **when** the list is shown or a session starts, **then** the list says they agree at that version and that no release was looked for, the start says nothing, and neither calls the version current or the newest.
- **Given** one plugin root and its binary, **when** a test asks its staleness with no plugin data directory named in the environment, with the directory of a marketplace install, and with that of an install loaded by path, **then** the answer is the same each time: judged against the version the root's own manifest names, and the download record, where it is shown, is called a download record and not the manifest's pin. Closes iss-2611251806207803.
- **Given** a plugin root whose binary reports another version than the root's manifest names, **when** the copies are listed, **then** the person reads that root named as differing from itself, with both versions, and no step is claimed to mend it. Reports what iss-2610241228359748 observed; the cause is iss-2611251828533623, which the second intent closes.
- **Given** the person runs `abcd update` or `abcd update --check`, **when** its receipt or its answer is printed, **then** it names each other copy still at another version than the one it took or found, with that copy's step, or says that every copy it found agrees. Answers the part of iss-2611251742484774 where an act on one copy left the others behind with nothing saying so; that issue stays open.
- **Given** the host's record of installed plugins is absent, unreadable, or in a shape abcd does not know, or a copy's version cannot be read, **when** the copies are listed or a session starts, **then** the list names what could not be read and says the comparison is partial, the start says nothing it cannot show, nothing is guessed from a directory's name, and neither the command nor the hook fails for it.
- **Given** a build from source, or the build a person opted into for the hooks, **when** the copies are listed, **then** it is named as that, with the commit it was built from where the build records one, is left out of the comparison of releases, and is given no update step.
- **Given** the list, the comparison, and the line at a session's start, **when** each runs under the test harness that holds the network out, **then** it passes, and a test shows that none of them changes a copy or a file of the host's.

## SOTA

From the primary sources the first leg of the verdict read on 2026-11-25 ([the verdict](../../research/ideas/488a0aa96ac5de805348635b27036addf15cddc2/2026-11-25-ideate-abcd-checks-for-updates.md), dcn-2611251807556458). Nothing was searched again for this draft, and how other tools list and compare their own copies on one computer was not searched as a question of its own.

- **The host's own plugin commands:** usable. One updates a named scope from the command line (verified there), and the host's own updating of a third-party marketplace is off unless a user or an admin turns it on (verified). Which of two scopes' copies loads could not be told from its documentation (unverifiable), and nothing of the host's knows abcd's binary on the command path or a build from source, so it cannot give the whole list.
- **A tool's own look for a newer release** (one command-line tool asks at most daily, on a terminal, with an off switch; verified): mature. It compares one copy with the newest release and never the copies with each other; it is the second intent's ground and none of this one's.
- **Not searched:** doctor commands that report every installed copy of a tool and the one the command path reaches. The writing session knows of them and read none for this.

Path 2: a native list read from disk, and no dependency. The seam is the host's own record of what it installed at each scope and its own update command: abcd reads the one and prints the other, so a host that comes to list or level its copies itself takes that part over, and the criterion for a record abcd cannot read already says what the list does without it.

## Open Questions

- **Deferred** Whether `abcd update` brings the copies abcd itself provisioned to one release, as the verdict's second ambition says, is the product thinker's: itd-130 keeps that verb out of every plugin root. Captured as its own issue before iss-2611251742484774 is resolved.
- **Deferred** Whether one stated precedence among the copies is still owed once each is named and compared, the ADR that iss-2610241228359748 asks for, is the product thinker's. Nothing here needs it: this reports which copy answered and does not rule which should. That issue stays open, narrowed to the ruling.

## Decisions

1. The product thinker's, on 2026-11-25 (dcn-2611251921171761): this was filed as an issue and is an intent, it matters, and it is built in their six-day unattended run with nobody to ask. They chose it from a sentence or two and gave no wording for a criterion; the criteria were not listed to them. The criteria, the Mechanism, and the scope conditions are the writing session's, written from the verdict and the issues named, for their review on return; a change then is made through the record. To reverse: leave it a draft until they have read the criteria.
2. The product thinker's, on 2026-11-25 (dcn-2611251823160636, adr-2611251808065076): abcd may ask for its newest release on a standing yes. That is the second intent's ground and none of this one's: the ADR's first consequence is that the disk comparison needs none of it and comes first. Applying a release unasked was not put and is not ruled on; it is in neither intent.
3. The verdict's (dcn-2611251807556458), carried as it stands, no question put, since a graduation carries the reframing: a notice goes to the person alone and never into the model's context; abcd names the owner's own step for a copy it does not own, never runs it, and never switches the host's own updating; and it never reports a copy as current. One clause of the reframing is carried by neither intent, "abcd brings the copies it owns to one release"; the Open Questions defer it.
4. The writing session's, no question put: the line at a session's start is shown at each start while the copies differ, and nothing is written to remember it. A split that stops being said is the fault being fixed. To reverse: show it once for each pair of versions.
5. The writing session's, for review: the split into two intents below follows the ADR's own order, and which issue each criterion closes is stated in the criterion. The second is not filed: a person confirms a split's routing before anything is filed from it, and that has been put to the product thinker.
6. The writing session's, no question put, tested on a copy of this tree on 2026-11-25: this intent ships with iss-2611251742484774 still open by moving that link off its `related_issues`, as the Decomposition sets out. The issue-drift gate refused the shipped intent with the link in place (exit 1, `shipped_unresolved`) and passed with it moved (exit 0). To reverse: file the second first and resolve the issue once both have shipped.

## Decomposition

Hand-run four-piece table, 2026-11-25. Two capabilities. The first is this intent: every copy known, named, and compared from disk. The second is a new intent, and it is not filed: abcd asks for its newest release on a standing yes and tells the person alone. The repository's rule is that a person confirms a split's routing before anything is filed from it, and that has been put to the product thinker; until they confirm, the text below is the only place the second is written. The trust rule, what a check may do and who gives the yes, has its home already: adr-2611251808065076, accepted, which refines adr-38 for that one request, and brief invariant 7, which names the exception. A second trust rule, applying a release unasked, is not ruled on and has no home; it is a person's ruling first and is in neither intent. No new stance. The list of copies, the reading of a manifest, the line at a start, and the receipt are plumbing, for the spec and for the brief's ahoy and update chapters (`04-surfaces/01-ahoy.md`, `04-surfaces/21-update.md`). Typed links: `refines` itd-111, widening its disk check from one copy against its binary to every copy; `refines` itd-130, whose receipt now names what an update left; `related_issues` iss-2611251742484774. No reversal is flagged for the first: its line at a session's start reads the disk, which adr-38 allows, and looks for no release. For the second, flagged for the product thinker and never auto-classified: it `reverses` in part itd-130, whose scope puts a nudge at a session's start out unless adr-38 is changed in the open, on the ground of the accepted ADR; and itd-2610031026190632 holds the product thinker's answer of 2026-10-03 that a daily look for the host's version goes out only when asked, which this does not touch. Verdict: SPLIT, the routing for a person to confirm.

What ships with which, while the second is not filed. A run that builds this intent builds the first alone. Its change resolves iss-2611251739422396 and iss-2611251806207803, each with a `Resolves:` trailer, and resolves nothing else. iss-2611251742484774 stays open, because half of it, the unprompted notice of a newer release, is in no record until the second is filed; closing it would leave that half nowhere. Open too stay iss-2610241228359748, narrowed to its ruling, and iss-2610290709017008 and iss-2611251828533623, which are the second's.

How it ships under the issue-drift gate. This intent names iss-2611251742484774 in `related_issues`, and the gate refuses a shipped intent that names an issue not in `resolved/`; the landing asks the same before the close and refuses there, offering to resolve the issue or to move the link. Resolving it is the wrong one of the two. What the gate accepts is a shipped intent whose every `related_issues` entry is in `resolved/`; an issue that names an intent in `related_intents` which does not name it back is not drift. So, in the lane's worktree before the close, the link is moved:

1. `abcd capture promote iss-2611251739422396 --intent itd-2611251834407931` names on this intent an issue its change resolves whole, and writes both halves of that join.
2. iss-2611251742484774 is taken out of this intent's `related_issues`. No verb removes one (iss-2611101319036237 asks for the move to be offered), so it is an edit of that one line. The issue keeps its `related_intents` line, so the trail from it to this intent stands.
3. `abcd capture remedy iss-2611251742484774` rewrites that issue's remedy to what is left of it: the standing check of the text below.

Tested on a copy of this tree on 2026-11-25: shipped with the link in place, `abcd intent audit --issue-drift --strict` exits 1 naming `shipped_unresolved`; with the three steps made and iss-2611251739422396 resolved, it exits 0, and record-lint has no finding on the moved link. If the product thinker confirms the split before this ships, the second is filed first, and `abcd capture promote iss-2611251742484774 --intent` with the second's id then names the issue on the intent that will deliver the rest.

The second intent's text, kept here until a person confirms the routing. It is then filed with `abcd intent`, its title given with `--title` and its impact with `--impact additive`, and takes a `builds_on` edge to this one through `abcd intent edge`.

**Title.** abcd asks for its newest release on a standing yes, and tells the person alone

**Press release.** Once a person has said yes on their own computer, abcd asks at most once a day whether a newer release exists, and tells them alone. Alice, a product engineer, runs abcd on one copy and never thinks to ask; a fix they need shipped last week. Having given the standing yes once, they read a line at their next session's start, shown to them and not to the agent, that names the release they run, the newer one, and the step that takes it. abcd reads the release's metadata and nothing else: it installs, stages, and runs nothing, and sends no version and no identifier. When the check fails it says it could not tell, and never that they are current. They withdraw the yes with one command, and until they give it abcd asks the network nothing on its own.

**Why.** A session start says nothing when the installed copy is older than the newest published release (iss-2610290709017008): two days after a release was cut, sessions still ran an older one with nothing saying so. The ruling that issue waited on is given (adr-2611251808065076, accepted on 2026-11-25 by dcn-2611251823160636). The comparison of this intent cannot say it, since only the release origin knows that a newer release exists.

**Mechanism.** We expect a check on a standing yes to reach the person the disk cannot, because the one who never asks is the one a release never reaches, and a daily question that carries nothing about them costs them nothing they did not agree to. Shown wrong if a person who gave the yes starts a session more than a day after a stable release was published and is shown no line, counted from the check's own record of when it last asked and what it was told.

**Scope conditions.** Holds on the computer where the person gave the yes; no repository file, environment variable, or agent gives it. Holds for stable releases; a pre-release is never offered. Holds for a notice; taking the release stays the person's act. Holds where the host has a channel to the person alone at a session's start.

**Criteria.** The writing session's, for the product thinker's review, as this intent's are:

1. **Given** no standing yes, **when** any operation of abcd's runs other than a verb whose meaning is a fetch, **then** no request is made, and the tests that hold the network out pass as they do today.
2. **Given** a person at their own terminal, **when** they give or withdraw the yes with the one command, **then** it is kept under abcd's home on that account, and `abcd ahoy` says whether it stands and when abcd last asked. A repository file or an environment variable gives none, and the command refuses in a shell that carries a session's markers, the ones a drive refuses on. The refusal rests on those markers and on nothing stronger: the ADR admits that anything running as the account can write the yes, which is why the yes covers a notice and no more.
3. **Given** the yes and more than a day since abcd last asked, **when** a session starts, **then** abcd asks the release origin once for the newest stable release's metadata, over the transport `abcd update` uses, sending no version, identifier, or token, and installing, staging, and running nothing. Within the day it asks nothing, and no hook that runs for each command ever asks.
4. **Given** a stable release newer than the copy that answers, **when** the check returns, **then** the person reads one line on the channel to them alone, naming both versions and the step for each copy behind, taken from the first intent's list; the model's context carries nothing of it. Closes iss-2610290709017008.
5. **Given** the check fails, runs out of time, or is told something it cannot read, **when** the line is composed, **then** it says abcd could not tell and when it last could, never that the copy is current, and the session starts all the same inside the hook's bound.
6. **Given** a newer release was found, **when** the line is shown, **then** nothing is downloaded or applied; taking it is `abcd update`, the person's act, and outside a drive nothing applies a release unasked.
7. **Given** a fresh plugin root, **when** the bootstrap provisions its binary, **then** it fetches the release that the version in the root's own manifest names, with the download cache keyed by tag, and makes no request that discovers a release; the opted-in build's request stays and is named in invariant 7. Closes iss-2611251828533623.
8. **Given** the change that adds the request, **when** it lands, **then** invariant 7, the update chapter, the update page, and the two tests that hold the network out change in it, as the ADR's consequences say.

The second carries what is left of iss-2611251742484774 once the first has shipped: its fourth criterion is the unprompted notice that issue asks for, and the change that ships the second resolves the issue. Applying a release unasked is not ruled on, and stays a question for the product thinker where the ADR's second point keeps it.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
