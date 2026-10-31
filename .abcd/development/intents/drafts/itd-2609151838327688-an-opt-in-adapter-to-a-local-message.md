---
id: itd-2609151838327688
slug: an-opt-in-adapter-to-a-local-message
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2609151838312703]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# An opt-in adapter to a local message broker brings push delivery and cross-machine reach to the session mailbox: with the adapter enabled a session is woken the moment a message arrives, machines on the local network exchange messages without a shared mount, and every delivered message is still mirrored to the mailbox as a record; without it the mailbox works as before

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

An opt-in adapter to a local message broker brings push delivery and cross-machine reach to the session mailbox: with the adapter enabled a session is woken the moment a message arrives, machines on the local network exchange messages without a shared mount, and every delivered message is still mirrored to the mailbox as a record; without it the mailbox works as before

## Mechanism

> _Prompted (the claim-recording gradient): why the authors expect this to work, as a falsifiable "we expect X because Y" — not the outcome restated. Replace this line with the claim, or with the exact token `None stated.` alone on its line to record the claim as considered and declined._

## Scope Conditions

> _Required (the claim-recording gradient): the population, platform, scale, or assumptions this claim holds under, one per top-level bullet — `abcd intent plan` stamps each with a persistent identity. Replace this line with those bullets, or with the exact token `None stated.` alone on its line._

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## Prior Art

- [itd-2609151838312703](../planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md) — the basic this adapter extends; sequenced after it, and it shares the mailbox's record contract.
- [2026-09-15-agent-messaging-on-a-local-network](../../research/notes/2026-09-15-agent-messaging-on-a-local-network.md) — ranks an embedded NATS server with JetStream as the adapter, with an MCP mailbox server as the runner-up.
- [basics-built-in-adapters-bring-power](../../principles/basics-built-in-adapters-bring-power.md) — the stance; the adapter is opt-in and never a precondition.

## SOTA

Surveyed on 2026-09-15 in the [local-network messaging note](../../research/notes/2026-09-15-agent-messaging-on-a-local-network.md);
re-checked on 2026-10-23 against the basic's 2026-10-15 ruling and the broker's
current documentation.

- **The basic this adapter extends has changed shape.** The 2026-09-15 survey
  ranked a Maildir-shaped shared-directory mailbox as the basic. The product
  thinker's 2026-10-15 ruling on
  [itd-2609151838312703](../planned/itd-2609151838312703-sessions-announce-what-they-are-doing.md)
  replaced it with fixed-form notices on the shared remote and no free text
  ([session-messages note](../../research/notes/2026-10-14-session-messages-across-accounts.md)).
  This draft's "mirrored to the mailbox as a record" and "without a shared
  mount" premises now point at a mailbox that will not be built. What is left
  for an adapter is push delivery: waking a session the moment a notice lands,
  instead of on its next fetch.
- **With persistence on the remote, JetStream is not needed.** JetStream is
  NATS's persistence layer, and the reason it ranked first on 2026-09-15 goes
  once the remote is the record of truth and the broker only wakes sessions.
  The candidate is core NATS embedded: the server is a Go package
  (`nats-server/v2/server`, Apache-2.0, v2.15.1 published 2026-10-09; a newer
  release exists, to be pinned at sign-off). It exposes `NewServer`, `Start`,
  `ReadyForConnections` and an in-process `InProcessConn`, so one abcd binary
  can host it with no second install
  ([package documentation](https://pkg.go.dev/github.com/nats-io/nats-server/v2/server)).
  The pages opened describe only the standalone binary and say nothing about
  embedding ([the NATS ecosystem page](https://docs.nats.io/concepts/ecosystem),
  where the clients URL now lands). Embedding is a property of the Go package,
  not a documented deployment mode.
- **An unconfigured server admits every connection** (NATS documentation, per
  the 2026-09-15 note), so authentication (NKeys) is mandatory configuration,
  not an option.
- **Free text stays out.** The incidents and the CaMeL-style defence behind the
  basic's no-free-text rule apply to anything the broker carries, so a broker
  message is a notice of the same fixed kinds, never a channel for prose.
- **Runners-up, re-weighed for wake-up only**: Mosquitto and Redis Streams do
  not embed in a Go binary, but an embeddable Go MQTT v5 broker does exist
  (mochi-mqtt/server, MIT, active September 2026) and is the lighter
  alternative to embedded NATS for this role, to be weighed by dependency
  weight against nats-server at sign-off; ZeroMQ has no persistence, which
  this role no longer needs; A2A assumes a served endpoint per agent.

**Path 2**: an opt-in adapter carrying the basic's notice record unchanged,
with the remote as the record of truth and the broker as a wake-up layer only.
The broker is the smaller embeddable one that meets wake-up only (core NATS
without JetStream, or mochi-mqtt/server); either is a new Go dependency and
needs explicit sign-off before `go get`, for the smaller one. The independent fit-challenge runs at plan time.

## Open Questions

_None recorded yet._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
