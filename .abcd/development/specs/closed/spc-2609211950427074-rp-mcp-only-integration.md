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
  The server block is read only when the route is rp, so a fault in it never
  refuses a step in a repository that has not opted in (refined on review the
  same day).
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
  the lane's own changes whole as git's own diff (from the default-branch
  commit its last sync merged in, or its base, as the audit reads a lane; no
  external diff program, textconv or colour; at most 512 KiB), since the route
  reads no file of the checkout. The answer is the reviewer's whole return: the
  loop writes it to the return path the host's agent would have written, under
  the run's lock and only while the review still awaits it and nothing stands
  there, and verifies and records it through the same receipt, so a review the
  route ran differs from a host-run one in its `route` alone (`asked` and `ran`
  both `rp`). A concurrent step can re-tell the open review to the host while
  the route works; a return the host's agent wrote first stands, and the
  route's answer is kept aside beside it (`rp-return.md`) and recorded
  nowhere. An answer kept aside is created, never replaced: one kept there
  already sends the next to the next numbered name (`rp-return-2.md`, and
  on), and what the step reports names the file it wrote. It is written whole
  to a temporary file beside it and then linked to its name, so a name the
  step reports never holds part of an answer, even when the process dies
  mid-write; on a filesystem with no hard links (FAT, exFAT, some network and
  FUSE mounts) it is created and written at its name directly instead, so it
  is still kept aside rather than dropped (refined on review 2026-10-14). The loop places the answer and records it on the review's await in
  the run's state, in one write under the run's lock: the route and the
  answer's SHA-256 (`answered`, state schema version 12). A return the step's
  own receipt could not verify, because another invocation held the run's
  lock, and that the person hands back with `abcd implement receipt` is
  recorded as the route's when the bytes the receipt's verifier reads, in its
  one read, have that digest; a return the host's agent wrote, or rewrote, is
  the host's. Only a ruthless or security reviewer's await is read this way. No
  file beside a return says who wrote it: the review's directory is the one
  the host's agent writes to, so a note there (the first reading of this
  refinement, an `answer-<sha256>.route` file) let that agent claim its own
  return as the route's with one file write, and is withdrawn. The run's
  state file is within that agent's reach too, so this opens no privilege the
  agent lacked; it moves the attribution into the lock-held state, where every
  other attribution the run records lives (refined on review 2026-10-14). The
  answer goes from the await wherever the review stops being the route's: a
  placed return the loop takes back off the path, a review that falls back to
  the host, and a restart that re-tells it to a fresh agent, so the same bytes
  handed back later are the host's. Taking a placed return back needs no
  lock, so a busy lock never leaves the route's answer standing where the
  host's agent writes, and it removes only the bytes the await records as the
  route's: a return the host's agent wrote over the placement stays. The
  check and the removal are not one act, since the removal takes no lock: in
  the narrow window between them, a concurrent receipt that verifies the
  placed return keeps its recorded verdict but loses the file it read, and a
  write by the host's agent over the placement is removed with it (refined on
  review 2026-10-14). An
  answer the loop could not place, because another invocation held the run's
  lock past the retries, is kept aside the same way, and the contention
  refusal names where it is; with the lock busy nothing records it as the
  route's, so copied to the return path and handed back it is the host's
  (refined on review 2026-10-14).
- 2026-10-12 (agent's choice, for review): **Which reviews take the route, and
  where.** The ruthless and the security reviewer, when the step hands them to
  the host: the intent-auditor stays on the host (decision 2, reviews and not
  audits), and a reviewer the machine routes to a command-line runner keeps its
  runner. `implement step` reaches the route after the stage hands the review
  out, outside the run's lock, as it starts a runner; the loop's change is
  confined to the validator stage's file (`validate.go`), but for the
  receipt's stamping of the route the verifier recognised (`loop.go`), the
  await's `answered` and its schema version (`state.go`), its masking
  (`state_mask.go`), and its dropping on a fallback (`drive.go`) and a
  restart (`restart.go`) (refined on review 2026-10-14).
- 2026-10-12 (agent's choice, for review): **Bounds.** Ten seconds for the
  connection and the session's set-up, thirty minutes for the review call,
  8 MiB for an answer; the address is never proxied, a redirect is never
  followed, and only a loopback address is dialled, whatever a host name such
  as `localhost` resolves to.
- 2026-10-12 (agent's choice, for review): **`fell_back: host` is the run's
  fallback receipt.** A review the route does not return is recorded as the
  run already records a role a runner did not run: one `runner.FallbackReceipt`
  in the state's `fallbacks` and a `fallback` line in the record (the role,
  `asked: rp`, the reason, abcd's own detail, `ran: host`), which `implement
  status` and `implement record` count per route and per role. No state field
  is added for a fallback, so it leaves the run-state schema as it was (the
  route's placed answer is version 12's `answered`, above). The host is handed the same
  brief and return path the step named, and no return is written for the
  route, so nothing stands where the host's agent writes.
- 2026-10-12 (agent's choice, for review): **Every review the route does not
  return falls back, not only a refused connection or a timeout.** No server
  named on this machine (`absent`), a refused connection or one closed before
  an answer (`unreachable`), a bound run out (`timeout`), the server's error
  (`refused`), an answer that is not the protocol's (`unparsable`), a return
  with no verdict the loop can record or larger than a return may be
  (`invalid`), a request that could not be composed (`failed`), and a step
  interrupted while the route worked, by the person or by a host's command
  timeout shorter than the route's bound (`interrupted`), each hand the review
  to the host with its receipt, as the runner falls back on every failure but
  a rate limit, so no review is stranded behind a refusal and every review the
  route was asked for and did not return has a receipt. The fallback's write
  is retried while another invocation holds the run's lock.
