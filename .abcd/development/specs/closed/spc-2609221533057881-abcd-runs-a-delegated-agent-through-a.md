---
id: spc-2609221533057881
slug: abcd-runs-a-delegated-agent-through-a
intent: itd-2609201916056194
origin: researcher-authored
production_mode: hand-written
---
# abcd-runs-a-delegated-agent-through-a-command-line-model-run

## Summary

The design record for itd-2609201916056194: one runner interface, a route per role, a configured fallback and the fallback as recorded intel.

## Scope

1. **The interface** (`internal/core/runner`): `Run(role, brief, contract) (answer, transcript, err)`; the host implementation is the existing sub-agent dispatch, so the loop calls one thing (criteria 1, 2).
2. **The adapters**: `claude` (print mode, bare, structured output, allowed tools from the role's contract) and `opencode` (run mode, or the server's session and prompt endpoints where a server is already up); each parses the harness's structured events into the one answer shape and writes the transcript to abcd's store (criteria 1, 6, 7).
3. **The route**: `roles.<role>.runner` through the layered resolver; unset is host (criterion 2).
4. **The fallback**: one place decides it (runner error, absent binary, non-zero exit, unparsable answer); it calls the host implementation, or `runner.fallback_host` when no host session is present, and appends a receipt to the run's state; the summary counts them (criteria 3, 4).
5. **The allowlist** resolved before launch (adr-2609221009491186) (criterion 5).
6. **Review**: security reviewer on the lane (criterion 8).

## Out of scope

- The tier's choice; installing a harness; a served console.

## Approach

The runner interface is the same shape the validator stage already defines for reviewers, so a role's route is one lookup; the fallback is one branch with one receipt writer, which is what makes the count trustworthy.

## Footprint

- packages: internal/core/runner, internal/core/implement, internal/surface/cli
- tests: each adapter against a fake harness binary; the fallback receipt on each failure kind; the count in the summary; the allowlist refusal; the bare flag asserted in the launch

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 same brief, contract, store | scope 1, 2 |
| 2 unset unchanged | scope 3 |
| 3 fallback and receipt | scope 4 |
| 4 counts in the summary | scope 4 |
| 5 allowlist | scope 5 |
| 6 bare flag | scope 2 |
| 7 receipts differ only in route | scope 1, 2 |
| 8 security review | scope 6 |

## Progress

The spec stays open: the live proof is owed (iss-2609301519558538) and no
ruling yet lets the intent close on fake harnesses.

- **Landed: scopes 1 to 5 in the core** (`internal/core/runner`): the runner
  interface and the one dispatcher, the claude adapter (print mode, `--bare`,
  stream-json, `--permission-mode dontAsk`, the role's tools allowed) and the
  opencode adapter (run mode, JSON events, `--pure`), the route through the
  layered resolver, the fallback with one receipt writer and `Tally`, and the
  allowlist admitted at the read and again before a launch; a model a
  provider lists is admitted by its allowlist alone and refused only by a
  configured `oracle.denylist` entry (adr-2609300107513982).
- **Landed: the loop wiring** (spc-2609202134338445 piece 3): `implement step`
  drives through `loop.Drive`, which starts a routed role through the
  dispatcher with the brief and receipt path the host would get, validates
  its answer with the stage's own receipt verifier, stores its transcript in
  abcd's history store, and stamps the verified receipt or recorded return
  with the route that ran it (criteria 1 and 7, structurally); an unset role
  leaves the step and the state byte-identical (criterion 2); a fallback is
  recorded in the state's `fallbacks` and the record (criterion 3) and counted
  per runner and per role by `implement status` and `implement record`
  (criterion 4); `build` and `step` refuse a runner configuration fault, a
  model off its allowlist included, before anything is created or launched
  (criterion 5). The bare flag is asserted in the launch (criterion 6).
- **Not built: the no-host path at the surface.** `runner.fallback_host`
  works in the core, but a surface that runs without a host session is the
  process driver's reversal of the host-delegated boundary, which waits on
  itd-2609201916151817's decision-6 ADR.
- **Owed: the live proof** of criterion 7 and the phase-1 unverified points
  (iss-2609301519558538), and **criterion 8**, the security review, which
  the lane's report lists point by point for the reviewer.
