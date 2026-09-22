---
id: spc-2609221533057881
slug: abcd-runs-a-delegated-agent-through-a-command-line-model-run
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
