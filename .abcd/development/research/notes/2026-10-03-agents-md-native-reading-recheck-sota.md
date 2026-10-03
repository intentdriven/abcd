# SOTA re-check: one conventions file, `AGENTS.md`, read natively by the first harness

Dated 2026-10-03. Re-checks
[`2026-09-30-claude-md-consumers-and-agents-md-sota.md`](2026-09-30-claude-md-consumers-and-agents-md-sota.md)
for the draft intent itd-2610030814013772 (abcd's projects keep one
conventions file, `AGENTS.md`), graduated from iss-2609291925136841. One
host-run research pass over the harnesses' own documentation and release
pages, read on 2026-10-03, condensed by the orchestrating session from the
researcher's final report (the researcher had no write tool). This note
re-checks and removes nothing.

**Bottom line.** The 2026-09-30 verdict, "do not remove `CLAUDE.md` yet",
holds if removal means no `CLAUDE.md` file at all. The harness's
documentation still names a one-line `CLAUDE.md` holding only `@AGENTS.md`
as the fallback, and says it never loads `AGENTS.md` twice. That pointer
neutralises every failure case below (an old version, the `agents-md` plugin
disabled, the first session after an upgrade, the setting at `claude-md`, a
stray `CLAUDE.md` above the project) except the `managed-only` setting. The
question this leaves for the product thinker: does "one conventions file"
allow that pointer, or mean no `CLAUDE.md` at all?

## Corrections to the 2026-09-30 note (vendor tier, read 2026-10-03)

1. Confirmed: native reading of `AGENTS.md` from v2.1.277, and from v2.1.281
   for sessions with telemetry switched off and on some cloud providers.
2. The project-instructions setting is honoured only in the user's own
   settings file, a `--settings` argument or managed settings, never in a
   project's settings, so abcd cannot set it from a repository.
3. Any `CLAUDE.md`, `.claude/CLAUDE.md` or `CLAUDE.local.md` in the working
   directory or any directory above it switches native reading off; the
   user-level `~/.claude/CLAUDE.md` does not.
4. The harness also reads `.claude/AGENTS.md`, and never `AGENTS.local.md` or
   `AGENTS.override.md`. From v2.1.280 a natively read `AGENTS.md` appears in
   `/memory`; the "AGENTS.md loaded" line is printed in interactive sessions
   only.
5. The v2.1.277 to v2.1.280 gap came from a remote flag that defaulted off
   without telemetry, fixed in v2.1.281 per the documentation and a team
   comment on a public forum, not per its release notes.
6. The 2026-09-30 note's quotation about plugin-provided agents is no longer
   on the sub-agents page.
7. The `AGENTS.md` project's own list of tools does not include this harness.

## Other hosts (vendor and project tier)

- Native `AGENTS.md`: Codex (a 32 KiB combined default cap, and
  `AGENTS.override.md`), Copilot (nearest file wins; a root `CLAUDE.md` or
  `GEMINI.md` is an accepted alternative), Cursor, VS Code (behind a setting),
  opencode (its v2 reads `AGENTS.md` only), Amp, Jules, and Zed (first match
  wins in a fixed order, so `.rules`, `.cursorrules` and Copilot's
  instructions file shadow `AGENTS.md`).
- Configuration needed: Gemini CLI (its settings naming `AGENTS.md`, or a
  one-line `@AGENTS.md` `GEMINI.md`), and Aider (`read: AGENTS.md`).
- `AGENTS.md` is stewarded by the Agentic AI Foundation under the Linux
  Foundation, formed 2025-12-09.

## Ranked recommendations (the researcher's, unconfirmed)

1. Embark follows `docs.target`, or prefers `AGENTS.md`.
2. The success bar is "one file holds the conventions", allowing a one-line
   `@AGENTS.md` `CLAUDE.md`, which replaces this repository's symlink
   (portable to a Windows clone, and editable).
3. Acceptance by a canary word placed in `AGENTS.md` and a session asked for
   it, host-run and opt-in.
4. Only if no `CLAUDE.md` at all: a switch-off detector over the root and
   every directory above it, never offering to write the user's settings.
5. Only without the pointer: a version-floor warning below v2.1.281, never a
   refusal.
6. Adopted repositories: classify an existing `CLAUDE.md` (identical, a
   symlink or import-only: offer retirement; a prose pointer: offer the
   import; the owner's own text: offer to move shared content and keep the
   remainder under `@AGENTS.md`); report-only by default.
7. This repository's `AGENTS.md` is 37,109 bytes, past Codex's 32 KiB
   default. A study (arXiv 2602.11988) found context files raise cost by
   about 20% without lifting success while their instructions are well
   followed, which favours moving material into the on-demand rules loader.
8. Gemini: prefer its settings file naming `AGENTS.md` over a symlink;
   document Zed's shadowing.
9. Never scaffold `AGENTS.override.md` or `AGENTS.local.md`.

**Not worth adopting:** the `CLAUDE.md` symlink; setting the instructions
value from the repository; a session-start hook printing `AGENTS.md`; a prose
"read AGENTS.md" `CLAUDE.md`; an import command; generators of per-host files.

## Unverified

The v2.1.277 to v2.1.281 changelog text (release pages were used instead);
the v2.1.281 fix line; the `/memory` change note; the mechanism of the first
session after an upgrade; `AGENTS.md` in the agent SDK, cloud sessions and
remote control; plugin-provided agents; Copilot's and Cursor's precedence
when both files exist. Two sources returned 403.

## Review record

2026-10-03: one research pass, condensed by the orchestrating session; no
independent adversarial reviewer has read this note or the 2026-09-30 note.
Per the [research protocol](2026-08-22-sota-research-protocol.md) that review
is owed before a removal relies on either. Promoted from the local tier on
2026-10-03 so the draft intent and its proposed ADR cite a committed
artefact.

## Sources (all accessed 2026-10-03)

- [Claude Code: memory (CLAUDE.md, AGENTS.md)](https://code.claude.com/docs/en/memory), and its sub-agents, agent-teams, agent-SDK features, on-the-web, cloud-environments, settings, remote-control, env-vars and changelog pages
- Claude Code release pages v2.1.277, v2.1.280, v2.1.281 and v2.1.282 (github.com/anthropics/claude-code)
- [Hacker News item 49814947](https://news.ycombinator.com/item?id=49814947); secondary write-ups at devops.com (2026-09-21), blog.szypowi.cz (2026-09-23) and markhuang.ai (2026-09-18)
- Codex `AGENTS.md` guide (learn.chatgpt.com)
- [Gemini CLI: GEMINI.md context files](https://geminicli.com/docs/cli/gemini-md/)
- Cursor rules (cursor.com/docs/context/rules); GitHub Copilot custom instructions (docs.github.com); VS Code custom instructions (code.visualstudio.com); opencode rules v1 and v2 (opencode.ai); Aider conventions (aider.chat); Zed documentation source; Jules documentation (jules.google/docs); Amp `AGENTS.md` (ampcode.com)
- [AGENTS.md](https://agents.md/); the Linux Foundation's Agentic AI Foundation press release (linuxfoundation.org)
- [arXiv 2602.11988](https://arxiv.org/abs/2602.11988)
