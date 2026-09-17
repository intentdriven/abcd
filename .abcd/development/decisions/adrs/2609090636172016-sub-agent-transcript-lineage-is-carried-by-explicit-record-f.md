---
id: adr-2609090636172016
slug: sub-agent-transcript-lineage-is-carried-by-explicit-record-f
status: accepted
date: 2026-09-09
supersedes: null
superseded_by: null
related_intents: [itd-2609090559376002]
related_rfcs: []
related_adrs: [adr-29]
---

# ADR-2609090636172016: Sub-agent transcript lineage is carried by explicit record fields, not a composite session id

## Context

adr-29 established the native transcript corpus and the record it stores: a
session id, the repository's root commit, a capture stamp, a source kind, a
content hash, a path and the redaction counts. It was written for one caller,
the session itself, and that caller needs no lineage: a session has no parent.

Sub-agent transcripts break that assumption. A sub-agent's transcript belongs to
the session that spawned it, and at greater depth to another sub-agent, so the
record has to answer two questions the schema cannot express: which session does
this belong to, and what kind of agent produced it.

The constraint was already locked by use rather than by design. Operators
capturing sub-agent transcripts by hand had only one writable identifier, so
they encoded the parent into it as `<parent-prefix>--agent-<agent-id>`. That
convention is in the store now: 176 of 267 records carry it. `Read` matches the
session id by exact string, so none of those records can be reached from the
identifier of the session that produced them, the parent prefix is truncated to
whatever the operator pasted, and the agent type was never recorded at all.

The harness supplies what is missing. Its sub-agent completion event carries the
spawning session id, the agent id and the agent type as distinct values; the
per-agent sidecar adds spawn depth and the parent agent. The information exists
at capture time and is being discarded.

## Decision

We will carry sub-agent lineage in explicit record fields, and the session id
will mean one thing only: the session a transcript belongs to.

A record gains fields naming the agent, its parent agent, its type, its spawn
depth, the tool use that spawned it, and where that lineage came from. A
sub-agent record's session id is the full, untruncated id of the spawning
session, which is also what a parent record carries, so a reader holding a
session id reaches the whole session by matching one field. Records already
written under the composite convention are migrated onto the fields, recovering
the untruncated parent from the transcript body the record already stores.

The new fields are externally supplied text and pass through the same redaction
gate as the transcript body. There is no field on this record that the scanner
does not see.

## Alternatives Considered

- **Continue the composite session id.** Costs nothing to adopt, since it is
  already in use, and needs no migration. Rejected: it is the status quo whose
  failure prompted this. It makes the identifier mean two things, keeps every
  existing sub-agent record unreachable from its session, and cannot carry the
  agent type at all. The convention would also harden with each capture, since
  every reader would have to learn to parse it.
- **Explicit fields on the record (chosen).** Each value means one thing, the
  session id keeps its existing meaning, and a reader filters rather than parses.
  Costs a schema version, a migration of 176 records, and the widening of the
  idempotency key so that two sub-agents of one session are not mistaken for
  duplicates. Chosen because the store exists to be read back, and an identifier
  that must be parsed to be understood is not readable.
- **A separate sidecar index mapping agents to sessions.** Leaves the record
  untouched and keeps the migration out of the store. Rejected: it puts lineage
  somewhere the record is not, so a record copied, exported or reconstructed
  without its index loses its meaning, and it introduces a second source of truth
  that can drift from the records it describes.
- **A separate record family for sub-agent transcripts.** Clean separation, no
  schema change to the existing family. Rejected: a sub-agent transcript is the
  same artefact captured through a different door, and splitting the family would
  duplicate the redaction path, the idempotency rule and every reader.

## Consequences

Easier: a session can be reconstructed by matching one field, an audit can ask
what a session's reviewers concluded, and the corpus can be grouped by agent type
without parsing identifiers. The reconstruction and telemetry work in
itd-2609090559376002 depends on exactly this and needs no further schema change.

Harder: the record carries a schema version and readers must admit both shapes,
since records written before this decision keep theirs until migrated. The
migration is a write over existing records and is therefore report-by-default,
applying only when asked.

New obligations: the idempotency key must include the agent id, or two sub-agents
of one session that produced byte-identical transcripts would collapse into one
record. Migration must not rewrite the content hash or the filename, both of
which existing readers and the deduplication path depend on. Any future field
added to this record is externally supplied until proven otherwise, and passes
through the redaction gate with the rest.
