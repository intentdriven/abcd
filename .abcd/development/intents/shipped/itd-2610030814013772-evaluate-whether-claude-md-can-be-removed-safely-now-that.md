---
id: itd-2610030814013772
slug: evaluate-whether-claude-md-can-be-removed-safely-now-that
spec_id: spc-2610031156364295
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2609291925136841]
origin: extracted-from-record
production_mode: hand-written
impact: breaking
---

# abcd's projects keep one conventions file, AGENTS.md, and Claude Code reads it directly

## Press Release

> When a person sets up a project with abcd, or works in abcd's own, there is one conventions file to read and keep: AGENTS.md. abcd writes no separate copy of it for Claude Code or for any other tool. If the project already holds a tool's own file with the owner's words in it, abcd leaves that file exactly as it is and warns, loudly, that AGENTS.md stays hidden until the owner moves those words across and removes the file; a file that only repeats or links to AGENTS.md is offered for removal, and goes only on a yes. Setup also warns when an older Claude Code, or a CLAUDE.md in a folder above the project, would hide AGENTS.md. A project set up earlier with a saved choice of CLAUDE.md stops at setup, and abcd explains the one setting to change.

_Proposed by the facilitator from decisions 1 to 8; confirmed as written by the product thinker at the planning interview, 2026-10-03 (decision 9)._

Previous wording (superseded at the interview):

> A project abcd looks after carries one conventions file, AGENTS.md, and nothing else of its kind: Claude Code reads it directly, so there is no CLAUDE.md copy to fall out of step. Setting up a project writes AGENTS.md only, and a project that already has a CLAUDE.md is offered its retirement, with nothing lost from what Claude Code loads. abcd's own project does the same.

_Proposed by the facilitator on filing (2026-10-03) from the product thinker's request "retire CLAUDE.md, use AGENTS.md instead in Claude Code"; to be confirmed or rewritten at the planning interview. Its reach ("a project abcd looks after") and "nothing else of its kind" both read as answers to questions the interview has yet to rule on (Q1, Q2 and Q5 below)._

## Why This Matters

Graduated from `iss-2609291925136841` (open since 2026-09-29), which asked whether CLAUDE.md can be removed now that Claude Code reads AGENTS.md natively. The research note [2026-09-30-claude-md-consumers-and-agents-md-sota](../../research/notes/2026-09-30-claude-md-consumers-and-agents-md-sota.md) answered: not yet, for named reasons, and the [2026-10-03 re-check](../../research/notes/2026-10-03-agents-md-native-reading-recheck-sota.md) holds that answer. Claude Code reads AGENTS.md as project instructions from v2.1.277 (v2.1.281 for sessions with telemetry off and on some cloud providers). It does not when any CLAUDE.md, .claude/CLAUDE.md or CLAUDE.local.md sits in the working directory or any directory above it, when the instructions setting names CLAUDE.md, when its agents-md plugin is off, and in some cases on the first session after an upgrade. In abcd's own checkout CLAUDE.md and GEMINI.md are two committed symbolic links to AGENTS.md, and the unpack step (embark) hard-codes CLAUDE.md as the file it plants its marker block into.

Today's links are not a safe status quo whichever way the interview rules: a Windows clone without `core.symlinks` checks a link out as a text file containing `AGENTS.md` with no `@`, which is a CLAUDE.md that switches native reading off, so that clone loads no instructions at all. The two adoption paths also contradict each other: prepare-this-repo scaffolds the CLAUDE.md link, and embark's `EnsureMarker` refuses to write through a link.

_Facilitator-written, from the research's verdict; the steps that depend on the product thinker's rulings are marked._ The blockers the note names become this intent's build steps: embark follows the chosen conventions target as setup already does; prepare-this-repo stops scaffolding the link (or plants a one-line `@AGENTS.md` import, if the pointer is ruled in); a warning names a file that would stop AGENTS.md loading (if no pointer is kept); and abcd's own CLAUDE.md and GEMINI.md links are removed with a fresh session shown loading AGENTS.md. Whether a host-version floor is stated anywhere is an open question, not a step.

The inventory the build steps work from, at tree f2a603427 (the design review's finding 4). Write and scaffold side, which changes: `internal/core/ahoy/marker.go` (`markerTargets`, the `claude_md` and `both` targets), `internal/core/ahoy/prompt_help.go`, `internal/core/ahoy/ahoy.go` (the `DocsTarget` values), `internal/core/ahoy/apply.go` (Uninstall strips both files), `internal/core/ahoy/embark_marker.go` (`EnsureMarker`, refuses a link), `internal/core/lifeboat/embark.go` (the hard-coded target), `internal/core/lifeboat/embark_types.go`, `commands/prepare-this-repo.md` (scaffolds the link), and `commands/embark.md`. Read side, which stays, so a repository managed through `claude_md` yesterday still reads as managed and still uninstalls cleanly: `internal/core/ahoy/detect.go` (`classify`), `internal/core/ahoy/managed.go` (`Managed`), `internal/core/ahoy/apply.go` (Uninstall), `internal/core/lifeboat/sources_native.go` (packs both files into a lifeboat), `internal/core/repolint/rule_router.go`, `internal/core/lint/lint.go` (the `stray_root_docs` link exemption), `internal/core/lint/citations.go`, `internal/core/runner/claude.go`, `internal/core/runner/opencode.go`, `internal/surface/cli/build.go`, `internal/surface/cli/cli.go`, `internal/core/ahoy/rewritelock.go` and `internal/core/ahoy/store.go`. Docs and record the removal breaks: `docs/how-to/install.md` (names both files), `docs/reference/cli/commands.md`, the brief's `02-constraints/01-platform.md` (cites `../../../../CLAUDE.md` as the "where things live" map), `04-surfaces/01-ahoy.md`, `04-surfaces/03-embark.md`, `04-surfaces/15-prepare-this-repo.md`, `05-internals/02-adapters.md` and `05-internals/03-configuration.md`, and `.github/workflows/ci.yml` (the inert-path classifier). Twenty-nine test files build CLAUDE.md fixtures. Retiring the `claude_md` and `both` values outright would be a breaking change to the command line; keeping them readable and refusing them at install is the alternative, and the record declares its impact before close.

The product thinker routed this on 2026-10-03: the capability here, and the rule that abcd writes one conventions file as its own decision (adr-2610030814023326, proposed). Typed links: refines iss-2609291925136841 (its evaluation, now a capability). The rule reverses the CLAUDE.md clause of the shipped itd-3's acceptance criterion ("CLAUDE.md is < 50 lines and contains the abcd marker block"); the drafts itd-21 and itd-10, which name a marker block in CLAUDE.md, are amended at planning. Other hosts' files (a GEMINI.md setting, a Cursor rules folder) are itd-22's (harness portability) question unless the interview extends this rule to them. The managed block that iss-2610020700529281 (open) is about would then land in AGENTS.md alone; that issue stays open and is not resolved by this intent.

_Facilitator-written at planning, from decisions 1, 7 and 8:_ the interview settles the conditional steps above. No pointer is kept (decision 1), so prepare-this-repo scaffolds no CLAUDE.md of any kind and the switch-off warning is a build step. The rule reaches every tool's own file, not Claude Code's alone (decision 7), so itd-22 (harness portability, draft) is amended at planning for decision 7: abcd writes no tool-specific conventions file for any tool, and the question of other hosts' files is answered here rather than there. The impact this intent expects is breaking (decision 8): a project whose saved setup choice names CLAUDE.md, or both files, stops at setup until that one setting is changed.

## Mechanism

We expect a project's agent instructions to stop drifting between tools because abcd writes them into one file, AGENTS.md, and leaves no second copy that can be edited on its own; a project abcd sets up that later holds two diverging instruction files written by abcd shows the claim wrong.

_Proposed by the facilitator from decisions 1 to 8; confirmed by the product thinker, 2026-10-03._

## Scope Conditions

- abcd's own project and every project abcd sets up (decision 2). <!-- cond: cond-2610031156364414 -->
- A Claude Code recent enough to read AGENTS.md itself; an older one is named in a setup warning, never served by a second file (decisions 1 and 3). <!-- cond: cond-2610031156365364 -->
- No CLAUDE.md, .claude/CLAUDE.md or CLAUDE.local.md at the project root or in a folder above it; where one exists AGENTS.md is hidden and setup warns (decisions 1 and 3). The user-level CLAUDE.md in the person's home settings is not counted. <!-- cond: cond-2610031156360114 -->
- An owner's own words in a tool's file stay theirs: abcd never merges or edits that file, so until the owner acts abcd's rules do not load there (decision 6). <!-- cond: cond-2610031156364999 -->

_Proposed by the facilitator from decisions 1 to 8; confirmed by the product thinker, 2026-10-03._

## Acceptance Criteria

_Proposed by the facilitator from decisions 1 to 8 and the design review's proposed criteria that survive them; each is unconfirmed until walked with its addressee. Review criteria AC4 (it named a one-line pointer file) and AC8 (it refused at install by naming the replacement, where decision 8 stops and explains) are rewritten; AC6 is widened to decision 3's warnings._

- A1 (product thinker; CONFIRMED 2026-10-03) Given an empty repository, when setup runs, then it writes AGENTS.md and no CLAUDE.md, GEMINI.md or other tool's conventions file, and prepare-this-repo's instructions scaffold none either (example: after setup in a new folder the root holds AGENTS.md and no CLAUDE.md, link or copy); a go test asserts both.
- A2 (technical facilitator; CONFIRMED 2026-10-03) Given an unpack target holding only AGENTS.md, when embark runs, then it plants its marker block in AGENTS.md and creates no CLAUDE.md (example: the target's AGENTS.md gains the block and its file list is otherwise unchanged); a go test asserts it.
- A3 (product thinker; CONFIRMED 2026-10-03) Given an adopted project whose CLAUDE.md holds the owner's own words, when setup runs, then the file is byte-identical afterwards and one loud warning names it and says AGENTS.md stays hidden until the owner moves the words across and removes the file (example: a CLAUDE.md reading "Always run make check first" is untouched and named in the warning); a go test asserts both.
- A4 (product thinker; CONFIRMED 2026-10-03) Given an adopted project whose CLAUDE.md or GEMINI.md is a link to, or a byte-for-byte copy of, AGENTS.md, when setup runs, then it offers to retire each one and removes it only on an explicit yes, leaving it in place on a no or a decide-later (example: a GEMINI.md link to AGENTS.md is offered, the answer is no, and the link is still there); a go test asserts each answer.
- A5 (technical facilitator; CONFIRMED 2026-10-03) Given a CLAUDE.local.md at the project root, or a CLAUDE.md in a folder above it, when the install checks run, then each is named in a warning that never refuses, never reads the file's content, names no host version string, and says why this presence check differs from the rule that abcd reads no settings folder above the working tree, while the user-level CLAUDE.md in the person's home settings is not named (example: a CLAUDE.md one folder up produces one warning naming its path); a go test asserts it.
- A6 (technical facilitator; CONFIRMED 2026-10-03) Given a project whose saved setup choice names CLAUDE.md or both files, when setup runs, then it stops and explains, naming the one setting to change, while the same project still reads as managed and still uninstalls cleanly (example: a saved choice of both files stops setup with a message naming the setting; uninstall then strips the block from both files); a go test asserts each.
- A7 (technical facilitator; CONFIRMED 2026-10-03, expected already met by abcd's own earlier switch, decision 5) Given abcd's committed tree, when the go test over it runs, then no CLAUDE.md or GEMINI.md exists at the root, as a link or as a file (example: the test fails if either is restored).
- A8 (product thinker; CONFIRMED 2026-10-03) Given a fresh Claude Code session in abcd's own checkout and a canary word in AGENTS.md's first section, when the session is asked for the word, then it answers it, recorded as a dated receipt in the local tier naming the date and the Claude Code version (example: the receipt quotes the question, the canary word and the answer).

## State of the art (re-checked 2026-10-03; [research note](../../research/notes/2026-10-03-agents-md-native-reading-recheck-sota.md))

_Agent-written: condensed from the two research notes; nothing here is the product thinker's._

- Claude Code reads AGENTS.md natively from v2.1.277, and reliably from v2.1.281 (between the two, a session with telemetry switched off, or on some cloud providers, skips it silently).
- Any CLAUDE.md, .claude/CLAUDE.md or CLAUDE.local.md in the working directory or any directory above it switches native reading off; the setting that would keep both is honoured only in the user's own settings, never in a project's, so abcd cannot set it.
- Anthropic's documentation names a one-line CLAUDE.md holding only `@AGENTS.md` as the fallback, and says it never loads AGENTS.md twice whatever the setting. It covers every failure case above except a managed-only setting. A symbolic link (this repository's current CLAUDE.md and GEMINI.md) is worse: a Windows clone turns it into a CLAUDE.md that switches native reading off, and abcd's own embark refuses to write through it.
- Codex, Copilot, Cursor, opencode, Amp, Jules and Zed read AGENTS.md natively; Gemini CLI and Aider need configuration (for Gemini, its settings file naming AGENTS.md, or a one-line `@AGENTS.md` GEMINI.md).
- A practical acceptance check: a canary word in AGENTS.md, and a session asked for it.
- Size: this repository's AGENTS.md is 37,109 bytes, past Codex's 32 KiB default limit (32,768 bytes), beyond which Codex stops reading further instruction text. Moving material under the on-demand rules loader is a separate capture, iss-2610031012543135, not this intent; the coupling is that this intent's canary word must sit in AGENTS.md's first section, or the canary measures the cap and not the rule.

_Facilitator-written:_ the decision this puts to the product thinker: does "one conventions file" allow a one-line `@AGENTS.md` pointer with no instructions of its own (the research's recommendation), or mean no CLAUDE.md at all (which makes a version floor and a wider switch-off warning necessary)?

## Decisions

1. 2026-10-03, the product thinker at the planning interview, asked whether a project may keep a one-line CLAUDE.md that only points at AGENTS.md (gain: every Claude Code version loads the instructions; cost: a second file stays): no CLAUDE.md at all. Projects carry only AGENTS.md, and setup warns where an older Claude Code or a CLAUDE.md in a folder above the project would hide it. The ADR's pointer alternative is rejected by this answer.
2. 2026-10-03, decided without a question (the record settles it): the reach is abcd's own project and every project abcd sets up. The product thinker confirmed that routing on 2026-10-03 (the decomposition-calibration entry "retire CLAUDE.md in favour of AGENTS.md": routing survived confirmation, unchanged).
3. 2026-10-03, decided without a question (follows from decision 1, the technical facilitator's): the version check is a warning in the install checks, never a refusal, and its user-facing text names no host version string a docs-lint rule refuses. The switch-off warning checks only whether a CLAUDE.md, a .claude/CLAUDE.md or a CLAUDE.local.md exists at the project root or in a directory above it; it never reads their content, and its text says why this presence check differs from the rule that abcd reads no `.abcd/` above the working tree. The user-level `~/.claude/CLAUDE.md` is not flagged: the research finds it does not switch native reading off.
4. 2026-10-03, decided without a question (the record settles it; the review's Q3, Q7 and Q9): this repository's CLAUDE.md and GEMINI.md links are removed whichever way the rest falls, because a Windows clone turns each into a file that switches AGENTS.md off; embark follows the chosen conventions target as setup does; the `claude_md` and `both` values stay readable for detection and uninstall; acceptance is a go test on the committed tree plus a dated local receipt from a fresh session, the canary word in AGENTS.md's first section.
5. 2026-10-03, decided without a question (sequencing, the technical facilitator's): abcd's own switch (removing the two links, with a fresh session shown loading AGENTS.md) lands first, as the fix the origin issue's remedy describes; it needs none of this intent's code, and it ends the Windows-clone failure now. This intent then carries it as an acceptance criterion already met.

6. 2026-10-03, the product thinker, asked what setup does with an adopted project's CLAUDE.md that holds the owner's own words (merging was settled against): leave it untouched and warn loudly that AGENTS.md is hidden until the owner moves their text and removes the file. The product thinker accepted the cost shown: until the owner acts, abcd's rules do not load in that project. A CLAUDE.md that is only a link to, or a copy of, AGENTS.md is offered for retirement and removed only on an explicit yes.
7. 2026-10-03, the product thinker, asked which files the one-file rule covers (Claude Code and GEMINI.md; every tool's own file; Claude Code only): every tool's own file. abcd writes no tool-specific conventions file for any tool, and setup offers to retire each one it finds under decision 6's terms. This widens the request ("retire CLAUDE.md, use AGENTS.md instead in Claude Code") and reaches into itd-22's ground (harness portability, draft), which is amended at planning; the ADR's other-hosts alternative becomes its decision.
8. 2026-10-03, the product thinker, asked what setup does with a saved choice of CLAUDE.md (or both files) from an earlier setup: stop and explain, naming the one setting to change. The values stay readable for detection and uninstall (decision 4). The impact is therefore breaking.

9. 2026-10-03, the product thinker: the revised press release confirmed as written.
## Open Questions

None open: the interview of 2026-10-03 answered the review's questions (decisions 1 to 8). Owed at planning: the press release and the acceptance criteria rewritten to the decisions, for the product thinker to confirm; the Mechanism and Scope Conditions; the ADR's Decision brought into line with decisions 1 and 7; and itd-22 amended for decision 7.


## Review findings (design and record discipline, 2026-10-03)

Two adversarial reviews read this draft and its proposed ADR: a design and feasibility review, and a record-discipline review (reports in the local tier). Nothing here is settled for the product thinker; each item is put to its addressee at the interview.

Applied as record fixes: the ADR's pointer clause and other-hosts clause moved from its Decision into its Alternatives as the facilitator's proposals, unconfirmed, with `status: proposed` kept (design 1; records 1); the product thinker's request quoted verbatim in the ADR (records 7); the version floor struck from the build steps and left to the questions (records 3); the full write and read site list added as the build steps' inventory, the read side kept (design 4); GEMINI.md named beside CLAUDE.md as a second committed link, in scope with it (design 5; records 6); the Windows-clone failure of today's link stated (design 2d); the switch-off conditions corrected to the research's wording (records 5); the unsourced facts dropped (a current version string, a per-file line advice, a refusal by the host's edit tools, which is abcd's own `EnsureMarker`) (records 5); every facilitator- or agent-written section marked (records 7); itd-3 named in the ADR's `related_intents`, and itd-21 and itd-10 named as amended at planning (records 4a); adr-2609091248200336 and adr-53 named in the ADR's `related_adrs` (records 8); other hosts scoped by cross-reference to itd-22 (records 4b); the re-check question struck as done, and the version question rewritten without version strings (records 9); the State of the art pointed at the committed [2026-10-03 note](../../research/notes/2026-10-03-agents-md-native-reading-recheck-sota.md), promoted from the local tier (records 6); merging an adopted CLAUDE.md named as rejected in the ADR's alternatives (design 3); the prepare-this-repo and embark contradiction cited (design 8); the size coupling recorded and the size itself captured separately as iss-2610031012543135 (design 6); a note added to the issue's Grounds that it was promoted without one (records 4d).

Not applied, because each is the product thinker's or the technical facilitator's: the pointer ruling (design 1 and 2), the handling of adopted files as build steps (design 3), the acceptance split (design 7), the Mechanism and Scope Conditions (design 9), and the atomicity of abcd's own switch (records 2). They are the questions below.

Interview questions, in order (PT = product thinker, TF = technical facilitator; [design] = a real choice, [settled] = decided by record or code, stated and moved past):

- Q1 PT [design]. Confirm the moment, quoting the press release: does "a project abcd looks after" mean every project abcd sets up, or abcd's own, which was the request's scope? And does abcd's own switch land with this intent, or first, as the open issue's fix?
- Q2 PT [design]. When someone opens the project in a Claude Code too old to read AGENTS.md, do we leave a one-line CLAUDE.md that only says "read AGENTS.md", or nothing, and tell them to upgrade? (The research recommends the one-liner; the ADR's wording follows this answer.)
- Q3 PT [settled]. Today's link goes whichever way Q2 falls: a Windows clone turns it into a CLAUDE.md that switches AGENTS.md off, and abcd's own embark refuses to write through it.
- Q4 PT [design]. A project that already has a CLAUDE.md with the owner's own words: leave it and say so, or offer to retire it only when it says nothing AGENTS.md does not? (Merging is settled against; it is not offered.)
- Q5 PT [design]. Does the rule cover the files other agent tools read (abcd's own GEMINI.md link, a Cursor rules folder), or Claude Code only, with the rest left to itd-22?
- Q6 PT [design]. The mechanism claim ("we expect no drift because there is one file to drift from"), or `None stated.`; and the impact: additive, or breaking if a project set up with `docs.target: claude_md` is refused afterwards.
- Q7 TF [settled]. Embark follows the chosen conventions target as ahoy's `markerTargets` does; the brief's and the install guide's CLAUDE.md references are re-pointed; the `claude_md` and `both` values stay readable for uninstall and detection.
- Q8 TF [design, only if Q2 is "nothing"]. The floor is a warning in the install checks, never a refusal, phrased host-agnostically for docs-lint; and the switch-off warning reads the root and the directories above it, saying why that read is allowed.
- Q9 TF [settled]. Acceptance splits into a go test on the committed tree and a dated local-tier host receipt; the canary word sits in AGENTS.md's first section. Size is a separate capture (iss-2610031012543135), not this intent.

Proposed acceptance criteria (agent-seeded, unconfirmed; each presupposes the pointer ruling and the reading of Q1, so they are input to the interview, not this draft's criteria): AC1 an install into an empty repository writes AGENTS.md and no CLAUDE.md (product thinker); AC2 embark into a target holding only AGENTS.md plants the marker there (facilitator); AC3 an adopted CLAUDE.md with the owner's text is byte-identical afterwards and named in one warning (product thinker); AC4 an adopted CLAUDE.md that is a link, an identical copy or only `@AGENTS.md` is offered for retirement, removed only on an explicit yes (product thinker); AC5 this repository holds no linked CLAUDE.md or GEMINI.md, asserted by a go test (facilitator); AC6 the conventions-router rule warns on a CLAUDE.local.md or a CLAUDE.md above the root, never offering to write the user's settings (facilitator); AC7 a fresh session answers the canary word from AGENTS.md's first section, with a dated local receipt (product thinker); AC8 a `docs.target` of `claude_md` or `both` is refused at install naming `agents_md`, while an existing block still reads as managed and still uninstalls (facilitator). Full text in the design review; each is walked with an example at the interview.

The draft's title and its file name disagree (the slug is the issue's, reused by contract); the interview should know.

## Audit Notes

<!-- abcd-review: OWED receipt=rcp-2444af75d52c -->
Fidelity review OWED (receipt rcp-2444af75d52c).
<!-- abcd-review-end receipt=rcp-2444af75d52c -->

## Grounds

- pursued: Windows copies lose their instructions today, and copies drift; we expect both to stop (the product thinker, 2026-10-03).
