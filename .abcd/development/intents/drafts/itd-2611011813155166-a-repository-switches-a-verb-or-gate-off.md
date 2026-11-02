---
id: itd-2611011813155166
slug: a-repository-switches-a-verb-or-gate-off
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
origin: researcher-authored
production_mode: hand-written
---

# A repository switches an abcd verb or project gate off

## Press Release

> A repository's owner switches an abcd verb or a project gate off for their repository, and abcd says so wherever it matters: a declared key on the dormant pattern rules.json domains already use, read through the shared resolver, shown on the board and in help, with no code removed from the binary.

## Why This Matters

> _Why this matters to the user — replace before planning._

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Decisions

Answered by the product thinker on 2026-11-01.

1. **Who may switch what off is decided by whom a gate protects.** A gate that protects only the project's own work (the docs-fidelity check, the record gates) may be switched off by the repository's owner, for everyone working in it, and abcd shows that it is off on the board, in help and in the verb's own output. A protection that also guards the person or their machine (the shell guard's hazards such as refusing to remove the home directory, the secret scan, which stops a contributor's own token leaving their machine, and the runner sandbox once it is built) gets no off switch from this intent.
2. **The same split applies to the repository's committed `.abcd/guard.json`, which today can disable the whole guard or downgrade any hazard, including the one refusing `rm -rf ~`** (`internal/core/guard/config.go`). A repository keeps its committed, reviewable way to tune the hazards it declared itself and the project-level ones, so a rule that misfires on its own build is fixed once for everyone; it can no longer disable or downgrade a hazard classed as protecting the person, and the whole-guard kill switch moves to the person's own layer. This needs each bundled hazard classed as project-level or person-protecting, with person-protecting the default for an unclassed one.
3. **The person needs no standing switch for their own protections.** A false refusal of one command is met by the person running that command themselves, outside the agent; `abcd guard allow --once` covers only a branch or worktree delete the guard could not prove safe. Running without the runner's sandbox is decided as a choice only the person's own home configuration can make (adr-2610161621415708, proposed); the sandbox and its opt-out are not built yet.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
