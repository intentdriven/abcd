---
id: itd-2611042148411751
slug: an-agent-that-needs-a-new-tool-asks-by
spec_id: null
kind: null
suggested_kind: null
reclassification_history: []
builds_on: [itd-2611041815429986]
severity: minor
refines: [itd-2611041815429986]
impact: additive
origin: researcher-authored
production_mode: hand-written
blocked_by: [itd-2611041815429986]
---

# An agent asks for a new tool by proposing a change to the list

## Press Release

> An agent that needs a new tool asks by proposing a change to the project's list of tools, and a person approving that change approves the name. A person still says where the tool lives on each computer, so an approved name runs nothing until they do. Agents a host session starts can reach a registered tool through one command abcd gives them. "An agent asked for a formatter overnight," said Frank, an SRE. "In the morning I read a one-line change, merged it, told abcd where the formatter lives, and the item carried on from where it had stopped."

## Why This Matters

itd-2611041815429986 stops an item that needs an unlisted tool and leaves the rest to
a person: nothing yet carries the agent's request, and an agent a host session starts
cannot reach a registered copy off the host's PATH. Both matter most on the runs a
person is not watching.

The request needs a home that an injected agent cannot abuse. The review of
2026-11-04 found a pull request is that home: it changes only the list, and the rule
of itd-2611041815429986 keeps its approval with a person under any facilitator mode. Approving a name also grants
little, since a person still says where the tool lives
(adr-2611041815445203). The evidence is in
[the research note](../../research/notes/2026-11-04-repository-tools-on-path-sota.md).

## Mechanism

We expect a tricked agent's request to fail harmlessly, because it can only propose a
name and a bounded reason, shown as data, and nothing it proposes runs until a person
registers a file. We expect fewer stalled items, because a request becomes a
change someone can decide in seconds. This would be shown wrong if approved names were
routinely registered unread, or if a host session's agents ignored the command their
brief names.

## Scope Conditions

- Holds through abcd's own merge gate and check. A merge made directly on the forge skips the gate, and what stops it there is that repository's own settings, which abcd does not change.
- Holds as a filter against mistakes where an agent shares the person's account (adr-42). A host session's agent can still run what its account can reach.
- Holds for agents who follow their brief: the wrapper reaches a registered copy only for an agent that uses it.
- Holds for a name. Where a tool lives stays a person's act on each computer, and nothing here can change that.

## Acceptance Criteria

- **Given** a lane's agent needs a tool not on the list, **when** it asks, **then** the asking command of itd-2611041815429986 also proposes a pull request that changes only the list, and the item waits with its work kept.
- **Given** the request's name and reason, **when** abcd shows them, **then** they are held to the list's grammar and length limit and shown as data; what an agent writes elsewhere on the pull request is not something abcd controls, and the person is told to read the diff.
- **Given** the pull request merges, **when** the tool has no registration on a computer, **then** the item stays waiting there and the report names the command a person runs; merging never records a location.
- **Given** a request was declined, **when** an agent asks for the same tool again in that run, **then** it is refused, so asking repeatedly gains nothing.
- **Given** the wrapper and a listed tool's command, **when** it runs, **then** each registered file's fingerprint is checked, the command runs with those copies first on PATH, and its exit code is passed through; a command that is not a listed tool's is refused.
- **Given** the wrapper and a command the shell guard would refuse, **when** an agent runs it through the wrapper, **then** the guard refuses it just as it would bare, and allowing `abcd` does not allow what it wraps.
- **Given** a host session drives a lane and a registered copy is off the host's PATH, **when** the run starts, **then** the start proceeds and the lane's brief tells the agent to run its commands through the wrapper, replacing the refusal of itd-2611041815429986.

## Open Questions

_None open. Who approves a name is settled in decision 6 of itd-2611041815429986._

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._
