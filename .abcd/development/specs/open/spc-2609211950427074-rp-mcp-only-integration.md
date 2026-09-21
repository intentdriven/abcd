---
id: spc-2609211950427074
slug: rp-mcp-only-integration
intent: itd-6
origin: researcher-authored
production_mode: hand-written
---
# rp-mcp-only-integration

## Summary

The design record for itd-6 as re-filed on 2026-09-21: RepoPrompt over MCP
as one opt-in reviewer adapter for `build`'s validator stage, falling back
to the host with a receipt that says so.

## Scope

1. **Configuration**: `oracle.review` read through the layered resolver the
   model-tier and pacing intents share (flag, repository, machine, bundled
   default `host`); value `rp` selects the adapter (criteria 1, 3).
2. **The adapter**: `internal/adapter/repoprompt`: connects to the MCP
   server the person configured, sends the review request file the loop
   renders (the ruthless or security prompt, the diff, the record), and
   returns the verdict in the validated shape every validator returns; the
   loop records it (criterion 1).
3. **Fallback**: connection refused or a timeout marks the receipt
   `fell_back: host` with the reason, and the loop hands the same request to
   the host's agent; the run record and the summary name every fallback
   (criterion 2).
4. **Nothing spawned**: the adapter never starts RepoPrompt, writes its
   config or discovers it; absent configuration is the host path (criterion 3).
5. **The pages**: an entry in the brief's adapters chapter beside the CLI
   runner (itd-2609201916056194) and the opt-in on the command page
   (criterion 4).

## Out of scope

- Audits over RepoPrompt (decision 2); the cascade; setup discovery; the
  non-Mac flow.
- The workspace definition in the lifeboat (itd-7, waiting).

## Approach

The adapter implements the validator interface the implement spec's
validator stage defines (request in, validated verdict out), so the loop
does not know which route ran; the fallback is one branch in the stage. MCP
client: the smallest dependency that speaks the protocol, subject to the
new-dependency sign-off rule, or the transport already used by the
host-reuse hook if it exists.

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 reviews over MCP, verdict recorded by the loop | scope 1, 2 |
| 2 fallback with a receipt | scope 3 |
| 3 unconfigured unchanged; nothing spawned | scope 1, 4 |
| 4 pages | scope 5 |
