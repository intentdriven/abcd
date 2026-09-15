# Open Questions

> **Status: LIVE.** Open architectural questions are recorded here as they arise — from unresolved review threads, RFCs, and ledger items. The brief is the project's current state ([adr-5](../../decisions/adrs/0005-brief-is-current-state.md)), and an unresolved question is part of that state: a question the team is carrying belongs here now. Lifeboat extraction *reads* this file and grounds what the record can prove; it is a reader, never the sole populator.

## Purpose

This file lists architectural questions that the team did NOT resolve — left open deliberately or simply unfinished. The next agent reading this file should treat these as legitimate degrees of freedom: things the original team punted on, where a fresh design pass is permitted (or expected).

## Format

For each entry:

```markdown
## <Question>

- **Status:** open (not resolved)
- **What's at stake:** <consequences of resolving one way vs another>
- **Current best guess:** <if any — clearly labelled as guess, not decision>
- **Source:** <RFC / review thread / issue ledger entry>
```

## Why this is separate from `02-what-didnt.md`

`what-didnt.md` records *settled* dead ends: approaches that were tried, failed, and abandoned with prejudice. `open-questions.md` records *unsettled* design questions: things the original team didn't try at all, or tried inconclusively, or deferred. The next agent reads them differently — closed-with-prejudice vs open-for-fresh-thought.

## Related sources during build

- **[`.abcd/development/decisions/adrs/`](../../decisions/adrs/)** — Architecture Decision Records. Open questions that get resolved promote into ADRs.
- **`.abcd/development/roadmap/rfcs/`** — Request for Comments. Multi-stakeholder discussion artefacts (open / resolved-yes / resolved-no).
- **`.abcd/work/issues/`** ledger entries flagged as open questions or future-work seeds.

## Where does a hold-register-shaped record live?

- **Status:** open (not resolved)
- **What's at stake:** capture's triage routes (defect fix / promote / brief fix / wontfix) force frame-level unease into artefact-level fixes. A hold route — non-articulation recorded as data, carrying axes and exiting by articulation — needs a home: a new ledger state, a brief section, or a record family of its own; each choice shapes what automated reviewers may read.
- **Current best guess:** none recorded yet for the home — candidate RFC or intent, decomposed before filing. **Narrowed meanwhile:** the disposition record (cold-reading workstream) reserves the two-axis hold field — frame-location crossed with MoSCoW priority — present in its schema and unpopulated, so whatever home is chosen inherits one taxonomy rather than reconciling two; a hold is directional, carrying an `exit_condition`, and exits by articulation, never by silent expiry.
- **Source:** ledger seed `iss-2608220750029991` (hold route missing from capture triage); the reserved field lands with the detection-and-disposition-records intent.
