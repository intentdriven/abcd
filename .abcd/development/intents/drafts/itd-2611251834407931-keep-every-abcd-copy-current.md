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
related_issues: [iss-2611251739422396]
origin: extracted-from-record
production_mode: hand-written
---

# Every copy of abcd on a computer is known and compared, with the step that mends each

## Press Release

> abcd now knows every copy of itself on a computer, says which one answered, and says when they differ, with the one step that mends each. Alice, a product engineer, takes a plugin update in one project and believes it is in force. At their next session's start a line shown to them, and not to the agent, says that the session runs the copy at user scope, which is older, names the newer copy at project scope, and gives the host's own command that brings the older one level. `abcd ahoy` and the version line list the same copies: the plugin at each scope, the binary on the command path, and a build from source where there is one, each with its version and a mark on the one answering. Nothing is asked of the network for this. The comparison reads the disk, so it says that the copies agree or differ, and never that they are the newest.

## Why This Matters

Graduated from iss-2611251742484774, which was judged as an idea and reframed on 2026-11-25 (dcn-2611251807556458). This draft carries the first half of that reframing, and itd-2611252040542374 the second. On 2026-11-25 a plugin update taken at project scope left every session in that project on the user-scope copy, five releases older, while `abcd ahoy` said only that the plugin root was resolved (iss-2611251739422396). The same split was recorded on 2026-10-24 and again on 2026-11-21 (iss-2610241228359748). One binary in one plugin root gave three answers about its own staleness, by the data directory the environment named (iss-2611251806207803). The staleness check that exists (itd-111) reads the disk and does not compare the copies on one computer, and `abcd update` (itd-130) swaps the binary on the command path, refuses a plugin root, and says nothing of the copies it left. The verdict rests on this: agreement among the copies is the durable core, whatever is ruled about looking for a release.

## Mechanism

We expect a comparison from disk to end the unseen split because each of its inputs is already a file: the root a hook is handed names the copy that answers, each plugin root carries a manifest naming its version, the host records what it installed at each scope, and abcd records the binary it put on the command path. Nothing is inferred and nothing is asked of the network. Shown wrong if, on a computer whose copies stand at two versions, a session starts and the person's line does not name the copy that answers and the one that differs, or if the list gives a copy a version that neither its manifest nor its binary reports. Counted on the two layouts on record, those of 2026-10-24 and 2026-11-25, rebuilt as fixtures.

## Scope Conditions

- Holds on one computer and one operating-system account. Copies under another account, on another machine, or in a container are not looked for.
- Holds for the host configuration the session or the command runs under. Copies the host recorded under another configuration directory of the same account, the separate login of itd-2611251725203320 among them, are not read, and the list says so in one line.
- Holds for a host that records on disk the plugin copy it installed at each scope, in a form abcd can read. Where it does not, those copies are said to be unread and are not guessed.
- Holds for copies in the places abcd knows: the plugin roots the host installed, the binary abcd put on the command path, a build of abcd's own source in its own checkout, and the build a person opted into for the hooks. A copy put anywhere else by hand is found only when it is the one answering.
- Holds without the network. The comparison says agree or differ among what is installed, and never which release is the newest; that is itd-2611252040542374's.
- Holds for reporting. abcd changes no copy it does not own and runs no command of the host's.

## Acceptance Criteria

- **Given** a computer holding more than one copy of abcd, **when** a person runs `abcd ahoy` or asks for the version in a terminal, **then** they read one line for each copy: what it is (the plugin at user scope, the plugin at project scope for the project the command runs in, the binary on the command path, a build from source), its version, and a mark on the one that is answering. The lines the version block prints today and the `superseded_root` note stay as they are, and the list is added beneath them. Copies at project scope for other projects are counted and not named, and `--json` carries the same list and the same count and no more, since an agent reads it. Closes the naming half of iss-2611251739422396, and delivers the reporting half of iss-2610241228359748, which stays open for its ruling.
- **Given** copies at more than one version, **when** the list is shown, **then** the person reads which copies differ from the newest one installed and, for each, the one step that mends it: for a plugin copy the host's own update command naming that scope, as the installed host's own help gives it, with the note that a new session is needed before it loads; for the binary on the command path, `abcd update`. abcd prints the step and runs none of the host's. Where the host's help shows no update command that takes a scope, the step is said in words and no command is printed. One live check on the installed host, with a dated receipt in the local tier naming the host's version, shows the printed command is the one its help gives. Closes the rest of iss-2611251739422396.
- **Given** a session that starts on a copy older than another installed one, **when** the session-start hook runs, **then** the person reads one line, on the channel the host shows the person and not the model, naming the copy that answers, the newer one, and the step. What the hook hands the model carries no such notice and no command. The line is shown at each start while the copies differ, and the hook writes nothing for it. It reads the disk and discovers nothing: no release is looked for, so it is not the session-start nudge itd-130 puts out as ambient discovery. With the two criteria above, delivers the first half of iss-2611251742484774, which stays open for itd-2611252040542374.
- **Given** copies that all report one version, **when** the list is shown or a session starts, **then** the list says they agree at that version and that no release was looked for, the start says nothing, and neither calls the version current or the newest.
- **Given** one plugin root and its binary, **when** a test asks its staleness with no plugin data directory named in the environment, with the directory of a marketplace install, and with that of an install loaded by path, **then** the answer is the same each time: judged against the version the root's own manifest names, and the download record, where it is shown, is called a download record and not the manifest's pin. Closes iss-2611251806207803.
- **Given** a plugin root whose binary reports another version than the root's manifest names, **when** the copies are listed, **then** the person reads that root named as differing from itself, with both versions, and no step is claimed to mend it. Reports what iss-2610241228359748 observed; the cause is iss-2611251828533623, which itd-2611252040542374 closes.
- **Given** the person runs `abcd update` or `abcd update --check`, **when** its receipt or its answer is printed, **then** it names each other copy still at another version than the one it took or found, with that copy's step, or says that every copy it found agrees. Answers the part of iss-2611251742484774 where an act on one copy left the others behind with nothing saying so; that issue stays open.
- **Given** the host's record of installed plugins is absent, unreadable, or in a shape abcd does not know, or a copy's version cannot be read, **when** the copies are listed or a session starts, **then** the list names what could not be read and says the comparison is partial, the start says nothing it cannot show, nothing is guessed from a directory's name, and neither the command nor the hook fails for it.
- **Given** a build from source, or the build a person opted into for the hooks, **when** the copies are listed, **then** it is named as that, with the commit it was built from where the build records one, is left out of the comparison of releases, and is given no update step.
- **Given** the list, the comparison, and the line at a session's start, **when** each runs under the test harness that holds the network out, **then** it passes, and a test shows that none of them changes a copy or a file of the host's.

## SOTA

From the primary sources the first leg of the verdict read on 2026-11-25 ([the verdict](../../research/ideas/488a0aa96ac5de805348635b27036addf15cddc2/2026-11-25-ideate-abcd-checks-for-updates.md), dcn-2611251807556458). Nothing was searched again for this draft, and how other tools list and compare their own copies on one computer was not searched as a question of its own.

- **The host's own plugin commands:** usable. One updates a named scope from the command line (verified there), and the host's own updating of a third-party marketplace is off unless a user or an admin turns it on (verified). Which of two scopes' copies loads could not be told from its documentation (unverifiable), and nothing of the host's knows abcd's binary on the command path or a build from source, so it cannot give the whole list.
- **A tool's own look for a newer release** (one command-line tool asks at most daily, on a terminal, with an off switch; verified): mature. It compares one copy with the newest release and never the copies with each other; it is the ground of itd-2611252040542374 and none of this one's.
- **Not searched:** doctor commands that report every installed copy of a tool and the one the command path reaches. The writing session knows of them and read none for this.

Path 2: a native list read from disk, and no dependency. The seam is the host's own record of what it installed at each scope and its own update command: abcd reads the one and prints the other, so a host that comes to list or level its copies itself takes that part over, and the criterion for a record abcd cannot read already says what the list does without it.

## Open Questions

- **Deferred** Whether one stated precedence among the copies is still owed once each is named and compared, the ADR that iss-2610241228359748 asks for, is the product thinker's. Nothing here needs it: this reports which copy answered and does not rule which should. That issue stays open, narrowed to the ruling.

## Decisions

1. The product thinker's, on 2026-11-25 (dcn-2611251921171761): this was filed as an issue and is an intent, it matters, and it is built in their six-day unattended run with nobody to ask. They chose it from a sentence or two and gave no wording for a criterion; the criteria were not listed to them. The criteria, the Mechanism, and the scope conditions are the writing session's, written from the verdict and the issues named, for their review on return; a change then is made through the record. To reverse: leave it a draft until they have read the criteria.
2. The product thinker's, on 2026-11-25 (dcn-2611251823160636, adr-2611251808065076): abcd may ask for its newest release on a standing yes. That is the ground of itd-2611252040542374 and none of this one's: the ADR's first consequence is that the disk comparison needs none of it and comes first. Installing a release unasked was ruled on afterwards (dcn-2611252038565989) and is in neither intent (iss-2611252038564722).
3. The verdict's (dcn-2611251807556458), carried as it stands, no question put, since a graduation carries the reframing: a notice goes to the person alone and never into the model's context; abcd names the owner's own step for a copy it does not own, never runs it, and never switches the host's own updating; and it never reports a copy as current. One clause of the reframing is carried by neither intent, "abcd brings the copies it owns to one release"; the Open Questions of itd-2611252040542374 defer it.
4. The writing session's, no question put: the line at a session's start is shown at each start while the copies differ, and nothing is written to remember it. A split that stops being said is the fault being fixed. To reverse: show it once for each pair of versions.
5. The product thinker's, on 2026-11-25 (dcn-2611252038550260): the split into two intents is confirmed, and the run builds both, this one first. Which issue each criterion closes is the writing session's and is stated in the criterion. To reverse: keep the second as a draft.
6. The writing session's, no question put: iss-2611251742484774 is named on the second intent and not on this one, so that this can ship while that issue stays open for its other half. Tested on a copy of this tree on 2026-11-25: with this intent shipped and iss-2611251739422396 in `resolved/`, `abcd intent audit --issue-drift --strict` exits 0 while iss-2611251742484774 stays open; with that issue's link put back, it exits 1 naming `shipped_unresolved`; with this intent shipped and iss-2611251739422396 still open, it exits 1. To reverse: name the issue here and resolve it only once both have shipped.

## Decomposition

Hand-run four-piece table, 2026-11-25. Two capabilities. The first is this intent: every copy known, named, and compared from disk. The second, abcd asking for its newest release on a standing yes and telling the person alone, is itd-2611252040542374, filed on 2026-11-25 once the product thinker had confirmed the split (dcn-2611252038550260); its text is there and no longer here. The trust rule, what a check may do and who gives the yes, has its home already: adr-2611251808065076, accepted, which refines adr-38 for that one request, and brief invariant 7, which names the exception. A second trust rule, installing a release unasked, is ruled on (dcn-2611252038565989), is in neither intent, and is held by iss-2611252038564722 until it is planned, when that ADR is refined. No new stance. The list of copies, the reading of a manifest, the line at a start, and the receipt are plumbing, for the spec and for the brief's ahoy and update chapters (`04-surfaces/01-ahoy.md`, `04-surfaces/21-update.md`). Typed links: `refines` itd-111, widening its disk check from one copy against its binary to every copy; `refines` itd-130, whose receipt now names what an update left; `related_issues` iss-2611251739422396. No reversal is flagged: the line at a session's start reads the disk, which adr-38 allows, and looks for no release. Verdict: SPLIT, the routing confirmed by the product thinker.

What ships with which. The change that ships this resolves iss-2611251739422396, the one issue in its `related_issues`, and iss-2611251806207803, each with a `Resolves:` trailer, and resolves nothing else. iss-2611251742484774 stays open: it is in the second's `related_issues` and off this intent's, its remedy is what is left of it, and the line by which it names this intent in `related_intents` stays, so the trail from it stands. Open too stay iss-2610241228359748, narrowed to its ruling, and iss-2610290709017008 and iss-2611251828533623, which the second closes.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
