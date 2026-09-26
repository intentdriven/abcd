---
id: itd-2609251624540864
slug: a-lab-runs-one-procedure-from-intention-to-discard-and-every
spec_id: null
kind: discipline
kind_notes: "A method every lab inherits, with no user moment of its own: it fixes how a lab is run, recorded and closed, whatever the lab studies. The `abcd lab` verb family (itd-2609212137128014) mechanises the parts a machine can check; the rest is this record."
suggested_kind: discipline
reclassification_history: []
builds_on: [itd-2609212137128014, itd-193]
severity: minor
impact: additive
origin: researcher-authored
production_mode: hand-written
---

# A lab runs one procedure from intention to discard

## The rule

A lab is a throwaway world pinned at one commit, run to answer one question. It
runs one procedure, in six stages, and every amendment the hand-run labs earned
is part of it. Each rule below was forced by a failure in a lab; the procedure
changes only by a lab recording an amendment and the next lab reading it.

### The lifecycle

1. **Intention.** Before any mutation, the lab writes its question, its
   hypothesis and what would show it wrong, its measures, its STOP conditions,
   its sources and the pin, and reads every earlier lab's amendments. The
   intention is the pre-mutation contract the harvest is scored against: a
   constraint refined mid-lab is recorded beside it as an operating constraint,
   and the intention itself is not edited.
2. **Snapshot.** The world is a standalone clone detached at the pin, with no
   remote and no path back to the checkout it came from. The preflight runs
   here, before anything mutates, and so does every pinning it requires:
   pinning is a mutation, and it belongs to this stage.
3. **Mutate.** The lab's work, logged as it lands. Every probe writes its record
   before anything cites it, and every mutation session ends with a commit
   inside the snapshot, so the world survives the session.
4. **Harvest.** The harvest is assembled in the lifeboat's section shape —
   intention, method, what worked, what is open, candidates, coverage — after a
   refutation pass over every candidate and a retraction sweep that passes. The
   secret and privacy scan runs over it before anything is filed, and its
   output is kept as an artefact rather than asserted in prose.
5. **Record.** Knowledge enters the repository only through ceremony: a capture
   that names the lab in its `found_during` and the pin in its text, an intent
   or decision record that cites the lab, a decisions-log line. Lab filings are
   merged by a person, never automatically.
6. **Discard.** The session ends, not the home: the snapshot is archived as a
   bundle and its live tree deleted, and everything else stays.

### Isolation

- **The lab is self-contained.** It drives its work with its own binary, built
  from the snapshot, and runs under its own HOME, so the tools' own caches and
  stores land inside the lab. That is not harness isolation, so the preflight is
  required: it enumerates what a session in the snapshot loads — configuration,
  plugins, hooks — pins the snapshot's own copies, and refuses any path by which
  operator-level state reaches the lab world. An operator-level installation is
  never used inside a lab.
- **The gates stay armed.** The world keeps its identity and name gates; a repair
  adapts the snapshot's configuration, never its authorship or its gates. The
  preflight states the snapshot's expected hook and gate state, and lists every
  gate the world carries with its expected behaviour, so a refusal mid-lab is
  classified in seconds rather than investigated.
- **The lab's identity is a property of the world**, not an authorship claim: the
  snapshot's identity record is re-pinned to the lab, and the identity gate's
  refusal of anything else is expected friction.
- **Every delegated agent gets the same fence**: the snapshot only, never the
  real checkout, and the probe-record discipline. A sub-agent's boundary is the
  lab's boundary.
- **Host limits are registered up front.** A behaviour the host filesystem or
  scanner makes unreproducible — case-folding, an account name the secret scan
  refuses — is probed and recorded in the preflight, not discovered mid-run.
- **The working directory is asserted** under the lab home before any verb runs.

### Two binaries

The work binary is built once from the pristine snapshot, its vintage verified
equal to the pin, and never rebuilt; every claim about the world as it is cites
only it. Changes are tested with a second binary, rebuilt per mutation, and each
rebuild is one recorded line: the command, the source commit and the binary's
digest. A rebuild of the work binary mid-lab is a provenance event and is
recorded as one; the staleness gate firing against a mutated tree is expected,
and its clearance is recorded rather than passed silently.

### Evidence

- **No claim outlives its input.** Every probe writes its input, its command
  line, its exit status and its two output streams before the harvest may cite
  it, and names the exact artefact it observed — the binary's vintage, the
  record id, the file's digest — because the record is the repeat only when the
  identity is explicit. Preserved records are what make a review overthrow
  cheap.
- **The private tier is inspected by shape, never by value.** When a probe's
  input is private-tier content, its record holds a redaction note — shape,
  size, digest and the command line, marked `redacted-private` — and never the
  values. Shape includes structure: fence markers and field counts are in scope.
- **Prove from logs, not claims.** A finding carries its command and its result;
  a hypothesis is labelled one, and an unresolved candidate is labelled
  unresolved. Open issues a lab's hypothesis rests on are reproduced at the pin
  before the procedure built on them is trusted.
- **Real-session smoke is a mandatory stage.** An offline suite can pass while
  the thing it tests cannot be loaded by the real host; only a live session
  shows it.
- **A fail-closed gate is attacked beyond the happy path**: its tests cover the
  working-directory and fault axes as well as the payload.
- **Concurrent windows are stated before they run.** Each window records its base
  commit, every worktree's branch tip and the binary digest beforehand, declares
  whether its durable writes are restored or committed afterwards, and merges
  into a named lab branch, never the snapshot's base.
- **Absence is proved with an instrument that can see it**: a byte-exact search
  of the file, with the command recorded, never a lossy text extraction.
- **Cost is recorded as the runner reports it and labelled so.** A figure the
  lab could not measure is not claimed.

### Findings and corrections

- **A candidate is filed as a question** while its premise is unverified, and no
  candidate enters the harvest before a refutation pass: one source or record
  read whose purpose is to kill it.
- **A retraction is a sweep, not an edit.** Every retracted or corrected claim is
  proved absent from the lab's own documents by a search for the pattern — not
  the instance — before it may be marked applied.
- **A review round ends on mechanical criteria**: every retraction proved absent,
  and every candidate carrying a source or record citation that could refute it.
  Deeper questions after that belong to the next lab's intention.
- **A snapshot commit message says what its diff does**, within the repository's
  conventional prefixes.
- **A handover carries values, not pointers** — pins, digests, commit ids inline
  — and names the gates the next stage owes. A recovering session re-verifies
  against the source records, not the handover's summary of them.

### Halt and record

A STOP condition or a gate refusal halts the lab. It is recorded as a finding,
with the gate, the command and the reproduction, and the lab stops there rather
than adapting around it — no retry under another shape, no weakened gate, no
falsified input. A halt lifts only when the refusing gate passes again, and the
finding stays in the harvest, where it leads.

### Where a lab's material lives

Evidence lives at the operator level, in the machine-scoped lab store keyed on
the repository's root commit, because it must outlive any one checkout and a lab
can span several. Knowledge enters the repository only through the ceremony
above. The local ephemeral tier holds pointers only: the handover a filing
session picks up. Snapshots and bundles, transcripts and anything derived from
them without redaction, private-tier values, and absolute local paths never
enter the repository.

## Why

Three hand-run labs, run in sequence against one repository, took review from
four rounds with three application failures, to one round with one defect, to a
verdict of ship with no finding, while the model and the harness stayed the
same. The variable that moved was this procedure. Each rule above was forced by a
concrete failure in a lab — a correction asserted and never applied, a suite
that passed while the thing it tested could not load, a contamination the
preflight would have caught — and was written down as an amendment the next lab
read. A rule held only in the head of whoever ran the last lab is forgotten under
pressure; a record is not.

## The gate

- **Given** a lab, **when** it mutates anything, **then** its intention, its
  snapshot at the pin and its passing preflight already exist.
- **Given** a claim in a harvest, **when** it is read, **then** it cites a probe
  record or an evidence file in the lab that could refute it.
- **Given** a correction, **when** it is marked applied, **then** a sweep for
  its pattern across the lab's documents finds no instance.
- **Given** a gate refusal or a STOP condition during a lab, **when** it occurs,
  **then** the lab halts and records it as a finding rather than adapting
  around it.
- **Given** a lab's knowledge, **when** it enters the repository, **then** it
  enters through capture or a record, citing the lab, and no evidence enters
  with it.

## Fit

The discipline family is the right home: this fixes the method of every lab and
has no user moment of its own. It builds on
[itd-193](itd-193-a-verifier-works-on-a-copy-no-agent-mutates-a-live-worktree.md),
whose rule a lab applies at the scale of a whole world: the lab mutates a copy,
never a live checkout.

## Staging

The mechanised rungs are the `abcd lab` verb family
([itd-2609212137128014](../shipped/itd-2609212137128014-abcd-lab-mechanises-the-lab-conventions-three-hand-run.md)):
the store and the snapshot at the pin, the preflight's harness-isolation and
dual-binary checks, probe-record scaffolding, the retraction sweep, the harvest's
citation check and its section shape, and the halt a gate refusal holds. The
rest of this record — the refutation pass, the real-session smoke stage, the
concurrent-window statements, the handovers, the review exit criteria — is the
documented protocol, and a lab brief that cites this record is its enforcement.
