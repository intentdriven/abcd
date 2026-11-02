---
id: itd-2610161710316538
slug: when-abcd-drives-a-drain-or-a-build-from
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# Drains and builds a person's session drives report how isolated they were

## Press Release

> When abcd drives a drain or a build from a person's own session, the person can tell afterwards how isolated the run was. abcd never changes the person's host settings, so it cannot fence a session the person drives. Instead the run says at its start, and again in its morning summary, whether the host's own protections were on, and what that means for what the run could reach, so nothing is left to guesswork.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in the [security-isolation SOTA note](../../research/notes/2026-10-23-security-isolation-sota.md).

- **A hook knows the permission mode on most events**: `permission_mode` is listed among the common fields of hook input, beside `cwd` and `session_id`, with the caveat "Not all events receive this field"; the `UserPromptSubmit` example carries it, so a probe reads it from an event that does (Claude Code hooks).
- **Nothing documented tells a process whether the sandbox is on**: the environment page documents `CLAUDECODE=1` and `CLAUDE_CODE_CHILD_SESSION=1` as "spawned by the host" markers only (Claude Code environment variables).
- **The host's own test is a probe**: "Confirm commands run inside the sandbox" is `touch ~/sandbox-probe` failing with "Operation not permitted" on macOS or "Read-only file system" on Linux (Claude Code sandboxing).
- **"Protections on" has three answers**, not one: the mode, whether the Bash sandbox is on, and whether an unsandboxed retry is allowed, since an allow rule "also approves an unsandboxed retry" and file tools, MCP servers and hooks run "with your full access" even when it is on (same page).
- **Precedent for the summary line**: Copilot writes blocked network requests into the pull-request body; abcd's model-matrix criteria already say isolation in the host's harness is "asserted, not proven" (Copilot agent firewall; itd-2610161733316518).

**Path 2: a native probe shaped to the host's documented check**, run at the start of a drain or build through the run's own tool, reporting mode, write refusal and network refusal as three facts, each marked proven (a probe that failed as it should) or asserted (a host claim), repeated in the morning summary; a config read is the seam for a host that later exposes its state. The independent fit-challenge runs at plan time.

## Open Questions

- (SOTA 2026-10-23) What the probe targets: a write path and a host that are harmless when no fence is present, since the probe runs unfenced when it matters most.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
