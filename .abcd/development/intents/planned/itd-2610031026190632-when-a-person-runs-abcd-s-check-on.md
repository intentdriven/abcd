---
id: itd-2610031026190632
slug: when-a-person-runs-abcd-s-check-on
spec_id: spc-2610031342374947
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-111, itd-2610030814013772]
severity: minor
origin: researcher-authored
production_mode: hand-written
impact: additive
---

# abcd's update check says when a harness is older than abcd needs

## Press Release

> When a person runs abcd's update check, abcd also looks at each agent harness it works with that is installed on this computer, and says whether that harness is older than the version abcd needs. It reads only this computer: it asks the internet nothing about any harness. A harness at or above what abcd needs gets no warning, and newer releases are not mentioned. A harness below it gets one line naming the version installed, the version abcd needs, and the harness's own way to update. A harness abcd cannot read a version from is named as not checked, with the reason, and never as up to date.

_Proposed by the facilitator from decisions 1 to 8; confirmed as written by the product thinker at the planning interview, 2026-10-03 (decision 9)._

Previous wording (superseded by decisions 3 to 8):

> When a person runs abcd's check on purpose, abcd also says whether the harness it runs in is behind that harness's latest release: it reads the installed version from this computer, asks the internet for the newest one only because the person asked, and names the update to run. Nothing leaves the computer unless the person runs the check.

> _Proposed by the facilitator on filing (2026-10-03) from the product thinker's request "Warn the user if the harness isn't on it's latest version (if possible to test)"; to be confirmed at the planning interview._

## Why This Matters

The product thinker asked on 2026-10-03: "Warn the user if the harness isn't on it's latest version (if possible to test)".

What the facilitator's feasibility test found that day, on the product thinker's machine: the harness's command on the search path prints its installed version (the version the command printed that day is kept in the local-tier test note, not here); the harness's native installer keeps downloaded versions in a versions folder beside the active one, three of them that day, downloaded on three consecutive days; and the harness's own configuration read its automatic updates as off, a reading taken from an undocumented state file whose authority was not checked. The harness's newest release is not on disk: knowing it means asking the internet, which decision 7 rules out.

The facilitator's inference, not tested: a harness below the version abcd relies on lacks behaviour abcd assumes (for the one floor known today, reading AGENTS.md by itself), and outside setup nothing in abcd says so. The test does not show that an old version runs unnoticed: the same machine moved three releases in three days with the switch read as off, and whether the harness announces its own updates was not tested. The Mechanism below carries the inference as a claim that can be shown wrong.

abcd already says when its own binary is stale, in the same update check (itd-111); this adds the harnesses abcd works with to that check, read from this computer alone.

Typed links: builds on itd-111 (its stance that implicit checks read only the disk and only an explicit check asks the network, adr-38 and brief invariant 7, untouched because the harness line never goes online, decision 7). Builds on itd-2610030814013772: its spec, spc-2610031156364295, owns the installed-version reading and the floor in its step 5 and ships first, and this draft reuses that reading rather than writing its own (decision 8; the sharing was first named in the spec's open question 5). <!-- record-lint: forward-looking --> The downloaded-but-not-running signal is its own draft, itd-2610031325050110 (decision 6).

## Decisions

1. 2026-10-03, the product thinker, confirming the routing (itd-84, hand-run): split. The warning is this draft; the network rule stays with adr-38 and invariant 7; the installed-version reading is this draft's plumbing, shared with the AGENTS.md draft's floor warning.
2. 2026-10-03, the product thinker, asked when abcd may look up the latest harness version (only when asked; once a day at session start, which reverses the standing rule; from this computer only; decide later): only when asked. The warning appears when the person runs abcd's check on purpose; nothing goes out on its own, and adr-38 and invariant 7 stand unchanged. The product thinker accepted the cost shown: a person who never runs the check never sees the warning.
3. 2026-10-03, the product thinker, asked which check carries the line (abcd's update check, asking two sites; its own option; the setup check, which would reverse itd-111 decision 6): abcd's update check.
4. 2026-10-03, the product thinker, asked when the line warns (any newer version; only below the version abcd relies on; both, told apart): only below the version abcd relies on (the first that reads AGENTS.md by itself, the floor spc-2610031156364295 step 5 names). The product thinker accepted the cost shown: other updates are not mentioned. <!-- record-lint: forward-looking -->
5. 2026-10-03, the product thinker, asked which harnesses are checked (the one asked about; every harness abcd works with): every harness abcd has an adapter for; one with no version to read is named as not checked, never as current (brief invariant 16, settled).
6. 2026-10-03, the product thinker, on the newer-version-downloaded-but-not-running signal: file it separately, as its own draft.
7. 2026-10-03, the product thinker, told that decisions 3 to 5 need no internet at all (the floor is built into abcd, the installed version is on the computer) and that the AGENTS.md work's setup warning already covers Claude Code below the floor: keep this as its own feature, offline. The update check warns below the floor for every harness and never goes online for it; decision 2's network question is therefore moot, and adr-38 and invariant 7 are untouched. The cost accepted: a second place states the warning setup gives.
8. 2026-10-03, decided without a question (the record review's finding 1, settled): the installed-version reading is owned by spc-2610031156364295 step 5, which ships first; this draft reuses it and `builds_on` itd-2610030814013772. The reading is the harness installed on this computer (the command on the search path), never the running session's own copy. <!-- record-lint: forward-looking -->

## Mechanism

We expect a person whose harness sits below the version abcd relies on to update it, because the update check names that harness, the version installed, the version abcd needs, and the harness's own way to update, at the moment the person asked whether anything is out of date; a person who reads that line and is still below the floor at their next check shows the claim wrong.

_Proposed by the facilitator from decisions 1 to 8; confirmed by the product thinker, 2026-10-03._

## Scope Conditions

- Harnesses abcd has an adapter for; any other harness is named as not checked, never guessed at (decision 5). <!-- cond: cond-2610031342376215 -->
- The harness installed on this computer, meaning the command on the search path, never the running session's own copy, which can differ while a downloaded update waits for a restart (decision 8). <!-- cond: cond-2610031342379679 -->
- The floor abcd ships, a built-in version for each harness abcd relies on, with its source cited beside it, compared offline; a newer release above the floor is not mentioned (decisions 4 and 7). <!-- cond: cond-2610031342379902 -->
- Harnesses started from a terminal; a host that runs only inside an editor, with no command on the search path, is out of scope and reads as not checked. <!-- cond: cond-2610031342373695 -->

_Proposed by the facilitator from decisions 1 to 8; confirmed by the product thinker, 2026-10-03._

## Acceptance Criteria

_Proposed by the facilitator from decisions 1 to 8 and the design review's proposed criteria that survive them; each is unconfirmed until walked with its addressee. The examples use made-up version numbers, so no vendor's version string enters the record. Review criterion C1 is rewritten from "no verb without the check goes online" to "the harness line never goes online, even within the check" (decision 7); C2 is rewritten from the latest release to the floor (decisions 4 and 7); C3 is routed to itd-2610031325050110 (decision 6); C5, the release channel, is overtaken by decision 7; C7's dated receipt compares against the floor, not the vendor's release page._

- H1 (product thinker; CONFIRMED 2026-10-03) Given a counting stand-in behind the release-fetcher seam and a fake harness on the search path, when abcd's update check runs, then the stand-in sees only the request abcd's own release check makes, and the harness lines are the same when that stand-in refuses every request (example: with the stand-in failing, a fake harness below the floor still gets its warning line); a go test asserts both.
- H2 (product thinker; CONFIRMED 2026-10-03) Given a fake harness on the search path printing a version below abcd's floor for it, when the update check runs, then the text output carries one warning line, and the JSON names the harness, the version installed, the version needed, and the harness's own update step (example: a fake printing 1.4.0 against a floor of 1.5.0 names both numbers and the update step); a go test asserts the text and the JSON.
- H3 (product thinker; CONFIRMED 2026-10-03) Given a fake harness at the floor and another above it, when the update check runs, then neither gets a warning line and the JSON reads each as at or above the floor (example: 1.5.0 and 1.9.2 against a floor of 1.5.0); a go test asserts it.
- H4 (product thinker; CONFIRMED 2026-10-03) Given a harness with no adapter, a harness with an adapter whose command is not on the search path, or one whose version command prints nothing parsable, when the update check runs, then each is named as not checked with its reason, and none reads as current (example: a fake printing "unknown" reads as not checked because no version was printed); a go test asserts all three.
- H5 (technical facilitator; CONFIRMED 2026-10-03) Given today's update check output, when the harness lines are added, then abcd's own `check` JSON object is byte-identical to today's and the harnesses sit in a sibling field beside it (example: a stored copy of today's `check` object compared byte for byte, with and without harnesses present); a go test asserts it.
- H6 (technical facilitator; CONFIRMED 2026-10-03) Given the regenerated command reference and the how-to page that describe the line, when docs-lint runs, then its `harness/*` rules are clean and no new allow escape was added (example: the prose says "the agent harness", and a harness's name appears only in fenced sample output); the docs-lint go test in preflight asserts it.
- H7 (product thinker; CONFIRMED 2026-10-03) Given the product thinker's machine, when the update check runs from the source checkout, then each harness line agrees with what that harness's own version command prints the same day and with the floor abcd ships (example: a harness at the floor shows no warning, and a harness with no version command shows as not checked); a dated receipt in the local tier records it.

Impact expected: additive. The update check gains lines and a JSON field beside an unchanged one; nothing it says today changes meaning.

## Review findings (design and record discipline, 2026-10-03)

From reports/review-harnessver-design.md and reports/review-harnessver-records.md in the local tier. Many design findings assumed a lookup of the latest release; decision 7 (offline, against the floor) overtakes them.

Applied:

- Design 1 and records 2: the subject is the harness installed on this computer, not the running session (decision 8); the press release says so, and so does a scope condition.
- Design 2: the harnesses sit in a sibling field beside abcd's own `check` object, never inside it (criterion H5); the carrier itself is decision 3.
- Design 5 and records 6: a harness with no adapter or no readable version is named as not checked, with the reason, never as current (the press release and criterion H4).
- Design 8: the docs prose stays host-agnostic with no new allow escape (criterion H6).
- Design 9: Mechanism, Scope Conditions and Acceptance Criteria written, the mechanism restated against the floor.
- Records 1: one owner of the reading, spc-2610031156364295 step 5 (decision 8); the typed-links paragraph names it and cites the spec's open question 5, not the AGENTS.md intent's decision 3. <!-- record-lint: forward-looking -->
- Records 3: Why This Matters separates what the test found from the facilitator's inference, and the inference moves to the Mechanism.
- Records 4: no vendor version string in the record ("the version the command printed that day", kept in the local-tier test note); criterion examples use made-up numbers.
- Records 7: the disk signal is out of this record (decision 6).
- Record fixes: decisions 3 to 8, filed under Acceptance Criteria by mistake, now sit under Decisions unchanged except for the forward-looking markers record-lint needs on the lines that cite the spec; the title says what the line now does.

Overtaken:

- Design 3 (which release feed to ask) and criterion C5 (the stable channel): no feed is asked (decision 7).
- Design 4 ("behind" is transient while automatic updates run; a setting that holds updates): a floor makes "behind any release" moot (decisions 4 and 7); the per-install update step it proposed is kept as an open question below.
- Design 5's "undetermined" verdict (no network): there is no network to be without (decision 7).
- Design 6 and criterion C3 (fold the disk signal into the check): filed separately as itd-2610031325050110 (decision 6).
- Design 7 (whichever lands first owns the reading): settled by decision 8.
- Design 1's status-line route to the session's own version: the session's copy is out of scope (decision 8).
- Records 5 (the carrier as a reversal, and a second remote): the update check carries the line (decision 3), so itd-111 decision 6 is untouched, and no second remote is added (decision 7). The check's help text still widens from abcd alone to abcd and its harnesses, which criterion H6 covers.
- Records interview questions Q1 to Q9: answered by decisions 3 to 8; Q3 (the second remote) does not arise.

Left for its owner: the record review's correction to the decomposition-calibration note's "home" column belongs to that note, not this file.
9. 2026-10-03, the product thinker: the revised press release confirmed as written.

## Open Questions

The three filed questions are answered: which check carries the line (decision 3, abcd's update check); which hosts are checked and how (decisions 5 and 7, every harness with an adapter, offline, one with no version named as not checked); and the disk-only signal (decision 6, its own draft, itd-2610031325050110).

Owed at planning, for the product thinker to confirm: the revised press release, the Mechanism, the Scope Conditions, and each criterion walked with its example.

Open for the technical facilitator:

- A harness with an adapter whose version reads but for which abcd ships no floor (today the floor names one harness): proposed, it is named as having no floor to check against, never as current (brief invariant 16).
- The harness's own update step: one per adapter, or one per install shape read from where the command resolves (the design review's finding 4); proposed, one per adapter to start, and a harness whose own settings hold its updates is named as held rather than given a command.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: nobody runs abcd on a harness too old for it without being told (the product thinker, 2026-10-03).
