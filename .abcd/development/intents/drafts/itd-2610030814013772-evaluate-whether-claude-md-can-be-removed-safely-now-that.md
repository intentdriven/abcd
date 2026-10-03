---
id: itd-2610030814013772
slug: evaluate-whether-claude-md-can-be-removed-safely-now-that
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
related_issues: [iss-2609291925136841]
origin: extracted-from-record
production_mode: hand-written
---

# abcd's projects keep one conventions file, AGENTS.md, and Claude Code reads it directly

## Press Release

> A project abcd looks after carries one conventions file, AGENTS.md, and nothing else of its kind: Claude Code reads it directly, so there is no CLAUDE.md copy to fall out of step. Setting up a project writes AGENTS.md only, and a project that already has a CLAUDE.md is offered its retirement, with nothing lost from what Claude Code loads. abcd's own project does the same.

_Proposed by the facilitator on filing (2026-10-03) from the product thinker's request "retire CLAUDE.md, use AGENTS.md instead in Claude Code"; to be confirmed or rewritten at the planning interview._

## Why This Matters

Graduated from `iss-2609291925136841` (open since 2026-09-29), which asked whether CLAUDE.md can be removed now that Claude Code reads AGENTS.md natively. The research note `.abcd/development/research/notes/2026-09-30-claude-md-consumers-and-agents-md-sota.md` answered: not yet, for named reasons. Claude Code reads AGENTS.md as project instructions from v2.1.277 (some session types from v2.1.281) when no CLAUDE.md sits beside it and the instructions setting is left at its default; it does not when the setting names CLAUDE.md, when its agents-md plugin is off, or when a CLAUDE.local.md sits in the tree. In abcd's own checkout CLAUDE.md is a symbolic link to AGENTS.md, and the unpack step (embark) hard-codes CLAUDE.md as the file it plants its marker block into.

The blockers the note names become this intent's build steps: embark follows the chosen conventions target as setup already does; prepare-this-repo stops scaffolding the link (or, for an older Claude Code, plants a one-line `@AGENTS.md` import); the install checks state a Claude Code version floor; a warning names a CLAUDE.local.md that would stop AGENTS.md loading; and abcd's own link is removed with a fresh session shown loading AGENTS.md.

The product thinker routed this on 2026-10-03: the capability here, and the rule that abcd writes one conventions file and no host-specific copy as its own decision (adr-2610030814023326). Typed links: refines iss-2609291925136841 (its evaluation, now a capability); the managed block that iss-2610020700529281 (open) is about would then land in AGENTS.md alone.

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## State of the art (re-checked 2026-10-03; full report in the local tier, reports/sota-agents-md.md)

- Claude Code reads AGENTS.md natively from v2.1.277, and reliably from v2.1.281 (between the two, a session with telemetry switched off, or on some cloud providers, skips it silently). Current version: 2.1.288.
- Any CLAUDE.md, .claude/CLAUDE.md or CLAUDE.local.md in the project folder or any folder above it switches native reading off; the setting that would keep both is honoured only in the user's own settings, never in a project's, so abcd cannot set it.
- Anthropic's documentation names a one-line CLAUDE.md holding only `@AGENTS.md` as the fallback, and says it never loads AGENTS.md twice whatever the setting. It covers every failure case above except a managed-only setting. A symbolic link (this repository's current CLAUDE.md) is worse: it breaks on Windows clones and the edit tools refuse to write through it.
- Codex, Copilot, Cursor, opencode, Amp, Jules and Zed read AGENTS.md natively; Gemini CLI and Aider need configuration (for Gemini, its settings file naming AGENTS.md, or a one-line `@AGENTS.md` GEMINI.md).
- A practical acceptance check: a canary word in AGENTS.md, and a session asked for it.
- Size: this repository's AGENTS.md is 37,109 bytes, past Codex's 32 KiB default limit (32,768 bytes), beyond which Codex stops reading further instruction text; Claude Code's documentation advises under 200 lines per file.

The decision this puts to the product thinker: does "one conventions file" allow a one-line `@AGENTS.md` pointer with no instructions of its own (the research's recommendation), or mean no CLAUDE.md at all (which makes a version floor and a wider switch-off warning mandatory)?

## Open Questions

- The version floor: a project opened in a Claude Code older than v2.1.277 (or v2.1.281 for some sessions) loses its instructions without a CLAUDE.md. Is the answer a stated floor in the install checks, a one-line `@AGENTS.md` pointer file kept for older versions, or both?
- Existing CLAUDE.md files in projects abcd adopts that hold the owner's own text (not a copy of AGENTS.md): merged into AGENTS.md, left alone with a warning, or offered for retirement only when identical.
- Other hosts: whether the same one-file rule covers the files other agent tools read (a GEMINI.md, a Cursor rules folder), or only Claude Code's.
- The research is from 2026-09-30; re-check the vendor documentation for the native-reading conditions before planning.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
