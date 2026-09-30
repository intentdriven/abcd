---
id: spc-2609211918551301
slug: coherence-aware-grill
intent: itd-42
origin: researcher-authored
production_mode: hand-written
---
# coherence-aware-grill

## Summary

The design record for itd-42 as re-scoped on 2026-09-21: the automated
pre-pass before the planning interview (itd-84's next rung). It reads the
brief's invariants, the principles and a one-line index of every intent, and
writes the coherence questions into the planning brief.

## Scope

1. **`abcd intent prepass <itd-N>`** (CLI) and the same step at the top of
   the plugin page's interview: reads `.abcd/development/brief/02-constraints/
   03-invariants.md`, the principles directory, the draft, and an index the
   verb builds from every intent's id, title and shelf (criteria 1 to 3).
2. **Invariant conflicts**: a host-delegated judgement (adr-25) over the
   draft against each invariant, each finding quoting both lines; the binary
   assembles the input and validates the output shape, the host judges
   (criterion 1).
3. **Overlaps**: the same judgement over the draft against the index,
   naming siblings; each becomes a question with the four answers and an
   optional recommendation-with-reason in prose (criterion 2, decision 2).
4. **The planning brief**: written to
   `.abcd/.work.local/scratch/planning-briefs/<itd-N>.md` in the shape the
   intent page describes (summary-back, decomposition table, questions,
   blocks-planning flags); nothing else is written (criterion 3).
5. **The interview reads it**: the plugin page's interview opens from the
   brief, asks each question, and records the answer as a decision or a
   typed link on the record (criterion 4).
6. **Unanchored concerns** are written as questions marked unanchored
   (criterion 5).

## Out of scope

- Grading into the calibration note (the human's, on confirming the routing).
- The capture-time validator (itd-84's later rung).
- Any write to the draft, the brief or a sibling.

## Approach

`internal/core/intent/prepass.go`: the index builder, the input assembler
(invariants, principles, draft), the brief writer; the judgement is a host
pass with a validated JSON return, the pattern the intent audit and the
cold reading already use. The plugin page calls the verb, hands the host the
input, and writes the brief from the validated return.

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 invariant conflicts quoted | scope 1, 2 |
| 2 overlaps as four-answer questions, recommendation in prose | scope 3 |
| 3 reads the four inputs, writes the brief only | scope 1, 4 |
| 4 the interview asks and records | scope 5 |
| 5 unanchored concerns marked | scope 6 |
