---
id: itd-167
slug: the-product-thinker-answers-a-stop-in-a
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
---

# The product thinker answers a stop in a medium they already use, and answers it whenever they get to it

## Press Release

> _Seeded from a quoted-text intent capture. Expand into the full press-release narrative before planning._

## Why This Matters

The product thinker is not at a terminal, and requiring them to be there reimposes the constraint this framework exists to remove. A stop that reaches them is answered later, by someone who was not present when it was raised, through a surface they already use.

That makes a stop a durable artefact rather than a prompt: it serialises whole, carries its options with it, outlives the session that raised it, and is resumable by whatever answers it. The loop parks on it and continues elsewhere rather than blocking. The synchronous command-line prompt is the first implementation of that contract, not the contract itself ([adr-2609151528057260](../../decisions/adrs/2609151528057260-three-roles-who-each-artefact-addresses.md)).

Selection over composition is what makes this possible at all: an option set survives a round trip through a web page or an app, and a request for free prose does not.

This does not remove the mediated case, it names it. The command line is the facilitator's surface and a human facilitator works there always, while the product thinker does not work there at all. The two are often in the same room or the same call, with the facilitator reading a question aloud and entering the answer given back. That path stays valid and needs nothing new. What this intent adds is the path where nobody relays: the product thinker answers directly, in their own medium, whenever they get to it, and the loop neither waits at a terminal nor depends on a facilitator being awake.

Whatever the product thinker's surface renders, it renders under the same bounds the register sets: three options at most, a null answer always available, no option marked as preferred, and an explicit statement that no recommendation is being made where the question is a judgement rather than a fact. A surface that quietly reorders options by a score, or that drops the null answer for lack of room, breaks the rule while appearing to keep it.

## Acceptance Criteria

> _Required (the itd-1 discipline): add at least one Given-When-Then bullet describing the verifiable bar for "shipped" before this draft can be planned._

## SOTA

Surveyed on 2026-10-23; sources, evidence tiers and the local fit risks are in
the [autonomous-loops SOTA note](../../research/notes/2026-10-23-autonomous-loops-sota.md).

- **The forge already has a durable, role-addressed approval.** A GitHub
  environment's required reviewers hold a job in "Waiting"; one of up to six
  approves, a rejection fails the run, and a run waiting on approval is
  cancelled when it reaches the 35-day workflow-run limit, which "includes
  execution duration, and time spent on waiting and approval" (Actions
  limits; the 30-day failure is stated only by the Enterprise Server 3.5
  page). It is yes-or-no, not three options with a null answer.
- **Chat buttons notify; they do not store.** Slack must be acknowledged within
  3 seconds and a `response_url` works five times within 30 minutes (Slack
  docs); the answer has to land in a store of abcd's own.
- **A phone needs no app store.** Web Push retains a message at the push
  service for a `TTL`, carries an `Urgency`, and guarantees a 4096-byte payload
  (RFC 8030); iOS and iPadOS 16.4 deliver Web Push to home-screen web apps with
  no developer-programme membership, on a user gesture (WebKit blog). The push
  carries the stop's id; the page fetches the options.
- **The host's own media are bound to a live session or to its vendor.** A
  `PushNotification` tool reaches a phone through the vendor's infrastructure
  when remote control is connected; channels (Telegram, Discord, iMessage) push
  into a running session only while it is open, and `-p` disables
  multiple-choice questions so the session never stalls (Claude Code docs).
- **Waiting indefinitely is normal.** LangGraph's interrupt "waits indefinitely
  until you resume", and the OpenAI Agents SDK serialises a durable run state
  whose approvals "may sit for a while" (framework docs). Escalation after a
  timeout exists but is configured, with a 30-minute default and a repeat count
  (PagerDuty docs); the ruling that it is off by default stands.

**Path 2**: the medium is an adapter over the stop record of itd-169; the record
is the truth, each medium is a notification plus one way to write an answer
back. First adapter: the web page with Web Push for reaching a phone, as
decided, and MCP's URL-mode elicitation (specification 2025-11-25) as the
standard way an MCP client hands the person to that page: the server sends a
URL, the client shows it and gets consent, and a completion notification
follows, which pairs with itd-169's record; second: the forge's approval, the
only other medium that is durable and role-addressed on its own, linking to
the page for the choice. The Web Push sender (VAPID and payload encryption) is
a Go dependency wherever it is built; it belongs in the relay, not the binary,
and needs explicit sign-off either way. A relay-notified phone sits against
the local-only stance on agent messaging; the plan-time fit-challenge must say
where that stance ends. The independent fit-challenge runs at plan time.

## Open Questions

**Decided 2026-08-29 (product thinker).** The first surface is a web page the product thinker opens when they have time, with the loop carrying on meanwhile. The mediated path, where a facilitator reads a question aloud, stays valid and is not replaced.

- (SOTA 2026-10-23) Where the Web Push sender lives (the relay or the binary),
  given that it needs a dependency for VAPID and payload encryption, and how a
  stop's answer written from the page reaches the record on the machine that
  raised it.
- (SOTA 2026-10-23) Whether the forge's required-reviewer approval becomes the
  second adapter, carrying only the acknowledgement and linking to the page for
  the three options.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
