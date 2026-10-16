---
id: adr-2610161621415708
slug: abcd-s-command-guard-steers-agents-an-os
status: proposed
date: 2026-10-16
supersedes: null
superseded_by: null
related_intents: [itd-2610161621364075]
related_rfcs: []
related_adrs: []
---

# ADR-2610161621415708: abcd's command guard steers agents; an OS sandbox is the boundary for unattended runs

## Context

abcd's shell guard reads each command an agent is about to run. It refuses the hazards in its registry and tells the agent how to rewrite them. It works at the parse layer, and every vendor that ships a check of that kind documents it as steering rather than a security boundary. The research note `.abcd/development/research/notes/2026-08-18-guard-wrapper-family-and-parse-layer-limits.md` records this:
- Claude Code says argument patterns are fragile and that sandbox restrictions hold even when a prompt injection gets past the model.
- Cursor's denylist, and then its allowlist, were bypassed.
- Codex keeps an OS-enforced `sandbox_mode` apart from an approval policy layered on top.
- GitHub's coding agent runs in a throwaway, firewalled environment.

abcd now runs agents unattended: drains, build lanes, and headless agents its runner starts. On 2026-10-16 the product thinker named sandboxing those runs as an important feature (dcn-2610161619296615), and intent itd-2610161621364075 asks for it.

## Decision

We will treat abcd's command guard as steering for every run. For a run no person watches, the boundary is a sandbox the operating system enforces:
- writes are limited to the run's worktree and abcd's home folder;
- network access is limited to origin and the forge.

The guard keeps teaching and refusing inside the sandbox. No document may describe the guard as what keeps an unattended run safe.

## Alternatives Considered

- **The guard as the boundary, made stricter.** Rejected. Parse-layer checks are bypassable by construction (`bash -c`, subshells, encodings, a poisoned environment), as the vendors' own records show.
- **Approve or refuse every command per launch, with no sandbox.** Chosen as a first step for headless agents (dcn-2610161612269647). On its own it is not a boundary, for the same reason as the first alternative.
- **A container per run.** Possible. It is heavier, it ties abcd to a container runtime, and the macOS and Linux sandboxes cover the same ground without one. Left to the intent's planning.
- **An OS sandbox per run, with the guard kept as steering.** Chosen.

## Consequences

- Planning the intent must settle two things: which mechanism to use on each platform (Seatbelt, Landlock or bubblewrap, or the host's own sandbox mode), and whether a missing sandbox blocks an unattended run or warns.
- A brief invariant states the rule above once this ADR is accepted.
- The guard's docs and refusals keep their current behaviour but must not claim to secure a run.
- A repository whose work needs wider writes or network access than the sandbox allows has to declare it, so the sandbox becomes a declared configuration rather than a fixed one.
