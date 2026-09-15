---
id: itd-2609020625402599
slug: the-scribe-s-context-is-assembled-and-its-output-is-ingested
spec_id: spc-2609020626045177
kind: standalone
suggested_kind: null
reclassification_history: []
builds_on: [itd-188, itd-183, itd-180, itd-185]
severity: minor
impact: additive
origin: researcher-authored
production_mode: dictated-and-formatted
---

# The scribe's context is assembled and its output is ingested by a verb, and the record can show that no session held both a reading and the ledger

Typed links: `builds_on` [itd-188](../shipped/itd-188-machine-assistance-in-maintaining-the-ledger-without-any-con.md) (the scribe definition), [itd-183](../shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md) (the assembler and manifest idiom), [itd-180](../shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md) (the disposition validator), [itd-185](../shipped/itd-185-one-ingest-verb-validates-every-cold-reading-output-includin.md) (the ingest idiom); `refines` [itd-188](../shipped/itd-188-machine-assistance-in-maintaining-the-ledger-without-any-con.md) (the protocol becomes a verb).

## Press Release

> **The scribe is a verb, not a protocol.** `abcd scribe assemble` builds the scribe's context from the ledger's allow list and nothing else, and writes a manifest of what it passed. `abcd scribe ingest --scribe-json <path>` validates what the scribe returned, transcribes dispositions, admissions and surprises through the same validators the capture verbs use, and refuses a payload that authors anything. Every assembly, reading or scribe, carries a per-run stamp naming its kind and its run, and the session store gains a check that reports a retained transcript carrying both stamps of one run. Until now the scribe's inverse access rule was held by its definition alone; now it is held by construction, and a reader can check it.

> "The reading session and the ledger session were always meant to be two sessions," said an AI/agent researcher who maintains the ledger with machine help. "I want the tool to build the scribe's context so I cannot accidentally hand it the tree, and I want the transcript store to be able to say whether any session held both."

## Why This Matters

[itd-188](../shipped/itd-188-machine-assistance-in-maintaining-the-ledger-without-any-con.md) specifies the scribe as a definition with an inverse access rule, hand-run until an ingest verb lands, with two acceptance criteria: the scribe's assembled context contains ledger content and no shipped-tree material, and each reading run and each scribe run is a distinct retained session with no session holding both. Iteration 1 shipped the definition with the access rule stated in its allow list, and its own text says "There is no ingest verb". The fidelity verdict found both criteria met by declaration rather than by mechanism: no assembler runs for the scribe, and session retention cannot show that no session held both.

The scribe exists so that machine assistance in maintaining the ledger remains available without any context holding both ledger content and a reading. That property is the mirror of the read block, and the read block is held by an assembler, a manifest and an eval. The scribe deserves the same three things, or its half of the wall is an assertion.

## Decisions flagged for the maintainer

Both were adopted by the maintainer on 2026-09-02 as [adr-2609021016275803](../../decisions/adrs/2609021016275803-no-session-holds-both-a-reading-and-the-ledger-and-a-per-run.md), which states the mechanism behind brief invariant 15 and amends the invariant to name the stamp and the check.

- **The invariant exists, and the ADR states its mechanism.** Brief invariant 15 states that no session holds both a reading and the ledger; the ADR states the per-run context stamp and the session check as the mechanism, and the invariant's amendment names them. The per-run stamp is a change to the reading bundle's shape and moves the assembler version, as itd-199 did for the bundle's selector block.
- **A new top-level verb.** `scribe` is a verb of its own rather than a sub-verb of `reading`, because the two contexts must never share a front door. The surface coverage and index gates require its plugin page, its brief surface chapter row and its entry in the released surface snapshot, all in scope below.

## What's In Scope

- **`abcd scribe assemble --run <rdg-N>`**, which requires an ingested run and builds the scribe's context from the ledger's directories as the record declares them (the readings, dispositions, admissions, surprises and reframes stores, and the issue ledger's three status directories), together with the researcher's supplied dispositions text, and writes a manifest of what it passed by path. The run's reading records come from the store, not from a raw reading output supplied twice. It refuses any path outside the allow list by construction, including the shipped tree, the brief, the intents, the specs, the decisions and the session-transcript store.
- **`abcd scribe ingest --scribe-json <path>`**, which validates the scribe's four outputs (records to file, fidelity flags, outstanding items, refusals), writes dispositions, admissions and surprises through the existing validators and stores, and refuses a payload carrying a field the scribe may not author: a ground, a resolution, a disposition the researcher did not supply.
- **Per-run context stamps.** A reading bundle and a scribe context each carry a stamp naming the kind of session, the run and a digest of the context, matched exactly, so a session that merely reads the documentation carries none. `abcd history` gains a check, run by `history list` and by the smoke lane, that reports a retained transcript carrying both stamps of one run by name, reports which runs it saw when none does, and says the property is unobserved when the store holds no transcript that carries any stamp.
- **The scribe's manifest** is parked in the local tier at assembly and promoted beside the run at ingest, inside the read block, so the next reading cannot see it.
- **Surfaces:** the plugin page `commands/scribe.md`, a row in the brief's surface chapter index, the released surface snapshot, and the definition's Delivery section rewritten to name the verb.

## What's Out of Scope

- Any judgement by the verb about the content transcribed. The scribe authors nothing and the verb authors nothing.
- A change to the scribe's allow list. The rule stays the definition's; the verb enforces it.
- Redaction policy. Records written through the capture validators inherit the redaction those verbs already apply.

## Mechanism

We expect an assembler with an allow list to hold the inverse access rule for the same reason the reading assembler holds the read block: positive inclusion excludes by default, and a manifest makes the exclusion checkable. We expect the session-kind stamp to reach a retained transcript because a session reads its bundle or its context through a tool whose result the transcript retains, so the stamp text is in the transcript when the content was. It fails where a host hands a session content the transcript does not retain, which the scope conditions disclose and the check reports as unobservable rather than as clean.

## Scope Conditions

- The scribe's context is assembled from committed ledger content and from supplied text. A scribe that needs uncommitted ledger content is outside this scope. <!-- cond: cond-2609020626046719 -->
- The session check reads the native session-transcript store and can only see what a host retained. Where the host assembles context before anything is retained, the check reports that it cannot observe the property, and the definition's protocol remains the gate. <!-- cond: cond-2609020626048270 -->
- The verb transcribes into the stores that exist. A record family the scribe is asked to write that has no store is refused, never invented. <!-- cond: cond-2609020626049512 -->

## Acceptance Criteria

- **Given** an ingested run and supplied dispositions, **when** `scribe assemble` runs, **then** its context contains the run's reading records from the store, the other ledger content, and the supplied text, its manifest names every path passed, and no path from the shipped tree, the durable record outside the ledger, or the transcript store appears.
- **Given** a scribe payload carrying a disposition the researcher supplied, **when** `scribe ingest` runs, **then** a disposition record exists through the disposition validator and nothing else was written.
- **Given** a scribe payload carrying a ground or a resolution the researcher did not supply, **when** `scribe ingest` runs, **then** it refuses and names the field.
- **Given** a scribe payload naming an item it was given no disposition for, **when** `scribe ingest` runs, **then** the item appears in the outstanding report and no record was written for it.
- **Given** a retained transcript carrying a reading stamp and a scribe stamp of one run, **when** the session check runs, **then** it reports the transcript by name.
- **Given** two retained transcripts, one carrying each stamp of one run, **when** the session check runs, **then** it reports that no retained transcript carries two stamps of one run.
- **Given** an empty store, **when** the session check runs, **then** it reports that the property is unobserved rather than clean.
- **Given** the read-block eval over a repository holding a scribe manifest, **when** it runs, **then** the manifest does not reach any reading.

## Prior Art

- [itd-188](../shipped/itd-188-machine-assistance-in-maintaining-the-ledger-without-any-con.md) and its spec (the definition), [itd-183](../shipped/itd-183-the-cold-reading-sees-exactly-what-the-assembler-passes-posi.md) (the assembler and manifest idiom), [itd-180](../shipped/itd-180-a-cold-reading-s-findings-land-as-reading-records-and-the-re.md) (the disposition validator), the `abcd history` store.
- The cold-reading rulings of 2026-08-28 in the decision log.

## Open Questions

None. The flagged decisions are adopted as adr-2609021016275803.

## Audit Notes

_Empty. Populated by intent-auditor when intent moves to shipped/._

## Grounds

- pursued: we expect an allow-list assembler with a manifest to hold the scribe's inverse access rule by construction as the reading assembler holds the read block, and a session-kind stamp to make the two-sessions property observable in the transcript store; a scribe context reaching shipped-tree material, or a stamp that never lands in a retained transcript, would show it wrong
