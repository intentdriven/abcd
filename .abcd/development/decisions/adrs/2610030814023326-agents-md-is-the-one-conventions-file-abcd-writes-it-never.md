---
id: adr-2610030814023326
slug: agents-md-is-the-one-conventions-file-abcd-writes-it-never
status: proposed
date: 2026-10-03
supersedes: null
superseded_by: null
related_intents: [itd-2610030814013772]
related_rfcs: []
related_adrs: []
---

# ADR-2610030814023326: AGENTS.md is the one conventions file abcd writes; it never writes a host-specific copy

## Context

abcd writes a conventions file into the projects it looks after, and in its own project it keeps AGENTS.md with CLAUDE.md as a link to it. Two files holding one set of instructions can drift apart, and each new agent tool brings its own file name. Claude Code reads AGENTS.md natively from v2.1.277 under stated conditions (research note 2026-09-30-claude-md-consumers-and-agents-md-sota). The product thinker asked on 2026-10-03 to retire CLAUDE.md, and routed the standing rule to this decision.

## Decision

AGENTS.md is the one conventions file abcd writes, in its own project and in every project it sets up. abcd does not write a host-specific copy or link (a CLAUDE.md, a GEMINI.md, a tool's rules folder) holding the same instructions. Where a host needs a pointer to find AGENTS.md (an older Claude Code), the pointer carries no instructions of its own, only the reference. It stays `proposed` until the intent itd-2610030814013772 settles the version-floor question at its planning interview.

## Alternatives Considered

- One file, AGENTS.md, with at most a bare pointer for an older host (chosen): one source of instructions, nothing to drift.
- AGENTS.md with a CLAUDE.md link or copy (today's state): works in every Claude Code version, but two names for one file and a pattern each new tool repeats.
- A file per host, each generated from one source: no drift if the generator always runs, but more files in the person's project and a generator to keep correct.

## Consequences

Setup, prepare-this-repo and embark write AGENTS.md alone; abcd's own CLAUDE.md link is removed when the intent ships. A person on a Claude Code older than the floor needs the pointer or an upgrade, which the install checks must say. A brief platform-constraint line stating the rule is owed in the change that ships the intent.
