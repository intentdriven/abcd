---
id: itd-2611041531140641
slug: a-build-lane-that-waits-on-the-machine
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: []
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A build lane keeps what it knows through a long wait

## Press Release

> A build lane that waits on the machine no longer pays to reload everything it knows. Alice, a solo founder running several lanes at once, sees each lane start as abcd's own build agent, with only the tools a lane uses and a memory that outlasts a wait. The lane leaves the wait to a background command and is woken when it ends, so an afternoon of queued test runs costs what the work cost, not the work plus a reload every ten minutes.

## Why This Matters

Lanes spend most of their life waiting on the machine, and a delegated agent's
cached context lasts five minutes by default. In the seven days to 4 Nov 2026 on
one machine, at published API prices as a proxy (plan metering is unpublished,
though the proxy landed within a point or so of the host's usage panel on two
rows), 655 sub-agent calls after a pause of five minutes to an hour rewrote
125.9M tokens, a mean of 192k a call: about 22 percent of the week, nearly all of
it in lanes started as the host's general-purpose agent. The median pause was
592 seconds and 275 of the 648 pauses sat at 590 to 640 seconds, the shell tool's
ceiling, so most of this is a lane polling in ten-minute slices. Only 15.4M of
those tokens follow a command that names the gate or preflight outright; about
90M follow polling loops of no recorded purpose.

Lanes also start heavier than they need. A lane started as the general-purpose
agent carries every tool and every skill listing, while the implementer role the
loop grants holds six tools (`roleTools`, internal/core/implement/loop/drive.go).
First-call context differs by about 27k tokens between the lean reviewers and a
general-purpose lane; about 11k of that is identified as the skill and deferred
tool lists, and the rest is presumed.

## Mechanism

We expect one named build-lane agent to remove most of this, because a one-hour
cache outlasts the ten-minute waits that cause the rewrites and a background
command wakes the agent so it stops spending calls on polling. At published
prices a one-hour write costs 1.6 times a five-minute write, so a lane that never
waits pays more; the claim is that for lanes as measured, the net is about 18
percent of the week. It fails if plan usage does not weigh cache writes and reads
as the API prices them, which is why the criteria ask for a before-and-after
rather than taking the figure on trust.

## Scope Conditions

- Holds for a host whose sub-agent files can choose a cache lifetime and whose
  background commands wake the agent that left them; on any other host the role
  still runs, with no saving claimed.
- Holds where lanes pause for five minutes to an hour; a longer pause still
  rewrites, and a lane that never waits pays the dearer write.
- Measured on one machine and one account's week, at API prices.
- Reviewers and other short-lived agents stay on the default lifetime: nearly all
  the measured rewrites are in lanes, and a reviewer rarely waits.

## Acceptance Criteria

- **Given** a build run on a host with agent definitions, **when** `implement
  step` hands out the implementer role, **then** the payload names abcd's build
  lane agent to start, and a hand-started lane finds the same agent named on the
  build page.
- **Given** that agent's definition, **when** the audit tests run, **then** its
  tools are the implementer's granted tools plus those a background wait needs,
  no skill tool, and a test fails when the definition and the grant drift apart.
- **Given** that agent's definition, **when** the host starts it, **then** its
  cache writes are one-hour in the live transcript, proved in a real run on the
  plugin as installed; a person's own setting for sub-agent cache lifetime still
  takes precedence, and abcd writes nothing under the host's own settings folder.
- **Given** a lane brief, **when** the lane must wait at the load gate or for a
  preflight, **then** the brief tells it to run the gate's command form as a
  background command and stop until woken, never to poll, and a lane so started
  resumes on the wake.
- **Given** a reviewer or any other agent abcd ships, **when** its definition is
  read, **then** it declares no one-hour cache.
- **Given** one real multi-lane run before and one after, **when** they are
  compared, **then** the record states cache-write tokens after pauses, first-call
  context and the host's own usage figure where it shows one, and the intent is
  not reported as saving anything those figures do not show.

## Open Questions

- Does a plugin-shipped agent file honour the cache-lifetime field? The host's
  reference says it reads the field from sub-agent files and names only a command
  line form as ignoring it; no source says so of plugin agents, so the live
  criterion above decides, and a refusal there sends the intent back.
- Is a lane woken cleanly when started in the foreground, or must the host start
  every lane in the background for the wake to reach it?
- How much of the 90M of unclassified polling is waiting a background command can
  replace, against waiting on a forge check, which has no command to leave behind?

## Parked

Parked 2026-11-04 behind itd-2611041636196213. Two reviews of this draft asked for
changes before planning, and it is not ready as written:

- Logs show the larger cost is repeat reloads inside one wait: of 121.7M tokens
  rewritten in general-purpose lanes after a pause, 85.75M (70 percent) repeat within
  one merged wait, 121.36M come from lanes started in the foreground, and 94.46M
  follow a polling command. So stopping polling comes before a longer cache, costs no
  premium, and a guard warning on foreground wait loops could reach every agent. The
  mechanism above, and its 18 percent, are to be rewritten around that.
- The measurements belong in a research note, with 655 and 648 reconciled, and the
  draft needs typed links.
- A wait rule belongs in the agent's own body: a routed runner has no background
  commands, and a lane is forbidden a full preflight, so the criterion naming one is
  wrong. A background command stops with a foreground sub-agent, so the agent needs
  `background: true`, and its turn end may read as finished.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
