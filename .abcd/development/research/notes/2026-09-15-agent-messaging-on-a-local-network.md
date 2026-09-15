# SOTA survey — messaging between agent sessions on one machine and one local network

Dated 2026-09-15. Compiled from one host-run research pass, challenged per
[`prefer-sota`](../../principles/prefer-sota.md) against this repository's
conventions, and shaped by two maintainer constraints stated the same day:
every option must stay on the local machine or the local network, never a
vendor relay; and abcd ships a basic that needs no dependency, with an
external dependency as the opt-in that brings full power
([`basics-built-in-adapters-bring-power`](../../principles/basics-built-in-adapters-bring-power.md)).
Evidence tiers follow the ladder the context-window survey of 2026-08-22
uses.

**Why this note.** Two abcd sessions under two operating-system accounts on
one machine spent 2026-09-15 passing notes through a markdown file on a
shared desktop, one dated heading per message. It worked, and it has two
failure modes: two appends can collide, and nothing acknowledges. The
maintainer asked what the state of the art is, whether two named tools
answer it, and whether abcd should carry the capability itself. The routing
that followed is in the decomposition-calibration note; this note holds the
survey.

## What the harness offers today

The first harness has native cross-session messaging (its documentation,
vendor tier): same-machine delivery over a per-session socket that is
restricted to one operating-system user, so two accounts on one machine
cannot reach each other; cross-machine delivery through the vendor's servers
over its remote-control connection. Discovery is by files on disk: two
sessions reach each other only when they can see the same files. Its
channels feature pushes events into a session and needs vendor
authentication. Under the local-only bound, none of this is the transport;
the documented socket is usable only as a same-user wake-up, and that use is
noted below.

## The two named tools

**Hermes Agent** (Nous Research, MIT; primary docs). An agent runtime whose
"gateway" is a chat-platform gateway, not a message router. It implements the
A2A protocol (agent card at a well-known path, JSON-RPC on a fixed port,
localhost-only without a token, per-peer tokens and an explicit host binding
for remote exposure), so it qualifies under the local-network bound, but the
peer on the other end must itself be an A2A server; a session of the first
harness is not one. Its own documentation steers agents on one machine to
in-process delegation or its kanban board, a single-host SQLite file under a
trusted-local-user model. Verdict: a possible future peer through an A2A
adapter, not a message path.

**OpenClaw** (the project renamed twice in early 2026; independent
foundation, MIT; primary docs). A single local gateway process: agents run
inside it and message each other through it, agent-to-agent open by default
until an allowlist is written, cross-machine only by tunnelling to the one
hub, with a token-theft vulnerability fixed this year. Verdict: poor fit
against minimal daemons and host-agnosticism; relevant only if such an agent
later becomes a peer.

## Ranked options

1. **Built-in basic: a shared-directory mailbox in Maildir shape.** One
   record per message; the sender writes to a `tmp/` directory and renames
   into `new/`; the recipient moves `new/` to `cur/` as the acknowledgement.
   Rename delivery was designed to need no locking (Maildir specification,
   spec tier). Across accounts the directory is one both can write; across
   machines it is a mount the operating system already provides, so abcd
   runs no daemon. Each message carries the sender's identity and a
   signature over the body with a key held under the sender's `~/.abcd`, the
   same shape an independent file-mailbox project converged on (anecdote
   tier), and the first harness's own agent teams use a file mailbox per
   agent (vendor tier). Delivery into a running session is the prompt hook
   abcd already owns, injecting unread mail on the next prompt; a same-user
   watcher may post a wake-up line through the harness's documented socket.
   Limits, from the sources: keep `tmp/`, `new/` and `cur/` on one mount
   because rename must not cross devices; unique names carry the host name
   and entropy; never place an SQLite index on a network mount (the SQLite
   project's own corruption warning, vendor tier); on macOS a shared
   directory needs an inherited ACL for the group, which the community
   documents and Apple does not. It cannot wake an idle session across
   accounts, deliver in under a second, fan out to subscribers, or reach a
   machine without a mount.

2. **Opt-in adapter for full power: an embedded NATS server with
   JetStream, inside the abcd binary as a Go module.** Adds push delivery
   and subscriptions, cross-machine reach without a mount, subject
   addressing keyed on repository identity, at-least-once persistence, and
   real authentication (NKeys sign a server nonce so no secret crosses the
   wire; tokens; TLS; subject-level permissions). Costs one long-running
   process on the network, mandatory authentication configuration because
   an unconfigured server admits every connection (NATS documentation,
   vendor tier), and a mirrored file record per delivered message so the
   audit trail stays where the basic keeps it. Runner-up adapter: an
   MCP-based mailbox server with markdown mailboxes in a git repository
   (Python daemon, bearer token, poll-only from the session; under active
   development, anecdote tier).

3. **Hermes Agent as a peer**, through an official A2A Go library, once one
   is a session worth messaging. Keep the record format mappable to an A2A
   message so the adapter is cheap.

4. **OpenClaw as a peer**, below Hermes for the reasons above.

5. **A2A direct, one server per session**: deferred. The specification
   assumes each agent exposes a served endpoint with TLS and OAuth-class
   authentication and says nothing about local-network deployment; a
   server per note-passing session is the wrong weight.

## Not adopted

- The harness's native messaging as the transport: user-scoped socket and
  a vendor-relayed cross-machine path.
- The harness's channels: vendor-gated and not host-agnostic.
- ZeroMQ with CurveZMQ: strong authentication and encryption without a
  broker, but no persistence, so no record and both sessions must be up.
- Mosquitto and Redis Streams: credible brokers that do not embed in a Go
  binary, so each is a second install; a fallback if NATS is rejected.
- SQLite on the shared mount, per the corruption warning.
- A coordination protocol rather than a note-passing one: the practitioner
  report that works keeps writes single-threaded and calls free-form agent
  swarms a distraction (position piece, contested); the mailbox carries
  findings and handovers, which is what the desktop file already carries.

## Fit judgement

The mailbox is the formalisation of the hand protocol the two sessions ran
today, with the two failure modes removed; it is a documented protocol
before any automation, host-agnostic because any harness that can write a
file participates, auditable because the message is the record, and
transport-agnostic for the core because `internal/core` produces a record
and a surface writes it. The broker adapter matches the basics-plus-adapter
principle exactly and carries the same record contract. Both are filed as
intents; the security posture of a message from another account (untrusted
input: signed, size-capped, sanitised before a terminal, injected only from
a mailbox the recipient owns) is an ADR for the mailbox's planning.
