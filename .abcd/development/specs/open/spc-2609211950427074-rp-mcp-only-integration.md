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

## Settled here

Choices made during the autonomous build on 2026-10-12 where this record is
silent, each the narrowest reading that meets the criteria. Each is an agent's
choice, recorded for review; none changes a criterion.

- 2026-10-12 (agent's choice, for review): **No flag sets `oracle.review` in
  this build.** The criteria name the repository's and the machine's files, so
  the route is read from those two layers over the bundled `host`, and the
  resolver's flag layer stays empty for this key.
- 2026-10-12 (agent's choice, for review): **The server's address is the
  machine's alone.** The rp route reaches the MCP server named by
  `oracle.rp.url` (with an optional `oracle.rp.tool`), which only
  `~/.abcd.noindex/config.json` may declare: the address a review request (the
  lane's diff and its record) is sent to is the person's own to say, as a
  provider block's is, so a repository declaring `oracle.rp` is refused naming
  the machine's file. `oracle.review` itself may sit in either file, the
  repository's winning, as criterion 1 names both. The URL is admitted only as
  http or https to localhost or a loopback address, with no credentials, query
  or fragment, and no refusal quotes it. A repository that selects `rp` on a
  machine naming no server is not a fault: the route reads as rp with no
  server, so the opt-in never breaks a collaborator who runs no RepoPrompt.
- 2026-10-12 (agent's choice, for review): **The transport is MCP's
  Streamable HTTP, spoken with the standard library.** The Approach names the
  smallest dependency that speaks the protocol, subject to the new-dependency
  sign-off, or the host-reuse hook's transport if it exists; no such transport
  is in this repository and no dependency was signed off, so the adapter
  (`internal/adapter/repoprompt`) speaks JSON-RPC 2.0 over HTTP itself. Stdio
  would mean abcd starting RepoPrompt's MCP binary, which scope 4 rules out, so
  the person runs the server and abcd connects to its address. One review is
  one session: initialize, the initialized notification, tools/list,
  tools/call, and a closing DELETE.
- 2026-10-12 (agent's choice, for review): **The tool and its arguments.** The
  request goes to `oracle_send` unless `oracle.rp.tool` names another tool, as
  its `message` argument, with `new_chat: true` when the tool's input schema
  declares it, so every reviewer is fresh: this loop's validators are each a
  fresh agent, and the intent's same-chat re-review rule belongs to the
  predecessor lineage the re-filing set aside. A tool the server does not
  offer, or one that takes no `message`, is refused before any call. RepoPrompt
  chooses the model; abcd names none.
- 2026-10-12 (agent's choice, for review): **The request and the return.** The
  loop renders `request.md` beside the reviewer's brief
  (`validate/round-<n>/<role>/`): the reviewer's agent definition (the
  plugin's `agents/<role>.md`), the lane's brief with the record it was
  rendered from, the implementer's report and each earlier fix round's, and
  the lane's diff whole (at most 512 KiB), since the route reads no file of
  the checkout. The answer is the reviewer's whole return: the loop writes it
  to the return path the host's agent would have written and verifies and
  records it through the same receipt, so a review the route ran differs from
  a host-run one in its `route` alone (`asked` and `ran` both `rp`).
- 2026-10-12 (agent's choice, for review): **Which reviews take the route, and
  where.** The ruthless and the security reviewer, when the step hands them to
  the host: the intent-auditor stays on the host (decision 2, reviews and not
  audits), and a reviewer the machine routes to a command-line runner keeps its
  runner. `implement step` reaches the route after the stage hands the review
  out, outside the run's lock, as it starts a runner; the loop's change is
  confined to the validator stage's file (`validate.go`).
- 2026-10-12 (agent's choice, for review): **Bounds.** Ten seconds for the
  connection and the session's set-up, thirty minutes for the review call,
  8 MiB for an answer; the address is never proxied and a redirect is never
  followed.
