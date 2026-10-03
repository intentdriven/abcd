---
id: adr-2610030814023326
slug: agents-md-is-the-one-conventions-file-abcd-writes-it-never
status: proposed
date: 2026-10-03
supersedes: null
superseded_by: null
related_intents: [itd-2610030814013772, itd-3, itd-22]
related_rfcs: []
related_adrs: [adr-2609091248200336, adr-53]
---

# ADR-2610030814023326: AGENTS.md is the one conventions file abcd writes; it never writes a host-specific copy

_Drafted by the facilitator on 2026-10-03 from the product thinker's request quoted below, and revised the same day from two adversarial reviews; the title is the facilitator's wording. The Decision was brought into line with the product thinker's rulings at the planning interview of itd-2610030814013772 (2026-10-03, its decisions 1, 2, 3, 6, 7 and 8); the status stays `proposed`._

## Context

The product thinker's request, 2026-10-03, verbatim: "retire CLAUDE.md, use AGENTS.md instead in Claude Code". The product thinker routed the standing rule to this decision and the capability to the intent itd-2610030814013772.

abcd writes a conventions file into the projects it looks after. In its own project it keeps AGENTS.md, with CLAUDE.md and GEMINI.md as two committed symbolic links to it. Claude Code reads AGENTS.md natively from v2.1.277 (v2.1.281 for sessions with telemetry off and on some cloud providers), when no CLAUDE.md, .claude/CLAUDE.md or CLAUDE.local.md sits in the working directory or any directory above it, the instructions setting is at its default, and its agents-md plugin is on ([2026-09-30 note](../../research/notes/2026-09-30-claude-md-consumers-and-agents-md-sota.md), re-checked in the [2026-10-03 note](../../research/notes/2026-10-03-agents-md-native-reading-recheck-sota.md)).

Today's symbolic link is not a safe status quo: a Windows clone without `core.symlinks` checks it out as a text file containing `AGENTS.md` with no `@`, which is a CLAUDE.md that switches native reading off, so that clone loads no instructions at all. abcd's own embark also refuses to write its marker block through a symbolic link, while prepare-this-repo scaffolds exactly that link.

## Decision

AGENTS.md is the one conventions file abcd writes, in abcd's own project and in every project abcd sets up (decision 2). abcd writes no CLAUDE.md, neither a copy, a link nor a one-line pointer (decision 1), and no tool-specific conventions file for any tool, a GEMINI.md or a tool's rules folder included (decision 7): every tool's instructions live in AGENTS.md alone. A tool-specific file abcd finds in a project is never merged into AGENTS.md or edited: one holding the owner's own words is left untouched, with a loud warning that AGENTS.md stays hidden until the owner moves those words across and removes it; one that only links to or copies AGENTS.md is offered for retirement and removed only on an explicit yes (decisions 6 and 7). Where an older Claude Code, or a CLAUDE.md, .claude/CLAUDE.md or CLAUDE.local.md at the project root or in a folder above it, would hide AGENTS.md, setup warns and never refuses (decision 3). A saved setup choice naming CLAUDE.md or both files stops setup with an explanation naming the one setting to change, and stays readable for detection and uninstall (decision 8). Each numbered decision is one of itd-2610030814013772's, ruled at its planning interview on 2026-10-03; this record's status stays `proposed`.

## Alternatives Considered

- One file, AGENTS.md, with a bare one-line CLAUDE.md pointer for a host that cannot read it (an older Claude Code), carrying no instructions of its own: the research recommends it; rejected by the product thinker at the planning interview (decision 1), who chose no CLAUDE.md at all and a setup warning instead.
- The rule for Claude Code only, leaving other agent tools' files to itd-22 (harness portability): rejected by the product thinker at the planning interview (decision 7), who extended the rule to every tool's own file; itd-22 is amended at planning accordingly.
- AGENTS.md with a CLAUDE.md link or copy (today's state): two names for one file, a pattern each new tool repeats, and a link that breaks on a Windows clone as stated in the Context.
- A file per host, each generated from one source: no drift if the generator always runs, but more files in the person's project and a generator to keep correct.
- Merging an adopted repository's own CLAUDE.md text into AGENTS.md: settled against by [`the-users-directory-is-theirs`](../../principles/the-users-directory-is-theirs.md) and adr-2609091248200336; a merge has no inverse and abcd did not create the file.

## Consequences

_Facilitator-written; each follows from the Decision and is confirmed with it._

Setup, prepare-this-repo and embark write AGENTS.md alone, and abcd's own CLAUDE.md and GEMINI.md links are removed first, ahead of the intent (its decision 5). A person on a Claude Code too old to read AGENTS.md needs an upgrade, and setup's warning says so. A project set up with a saved choice of CLAUDE.md stops at setup until the setting is changed, so the change is breaking. itd-22 is amended at planning, because the rule now covers other tools' files. The rule reverses the CLAUDE.md clause of itd-3's shipped acceptance criterion ("CLAUDE.md is < 50 lines and contains the abcd marker block"), and itd-21 and itd-10, both drafts, are amended at planning where they name a marker block in CLAUDE.md. A brief platform-constraint line stating the rule is owed in the change that ships the intent.
