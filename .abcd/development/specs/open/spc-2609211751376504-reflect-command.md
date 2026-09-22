---
id: spc-2609211751376504
slug: reflect-command
intent: itd-24
origin: researcher-authored
production_mode: hand-written
---
# reflect-command

## Summary

**Re-read on 2026-09-21 (adr-2609212115255771): the unit is the release, not the phase.** Wherever this spec says phase document, read the release tag and the changelog section the cut composed; the seed is the intents whose `shipped_in` names the release, with their audit notes; the nudge is one line at the end of `launch ship`; the open-work warning names intents with `target_release` at that version still unshipped; the output path is `.abcd/development/retrospectives/<tag>/README.md`.

The design record for itd-24, from the product thinker's interview of
2026-09-21 (decisions 1 to 3 on the intent). One host-run interview,
`/abcd:reflect <phase-id>`, seeded from what the phase actually shipped,
written to the durable record tier, packed by the lifeboat and surfaced on
embark.

## Scope

1. **The seed**: the phase document under `.abcd/development/roadmap/phases/`
   names the intents it bundled; for each in `shipped/`, the `## Audit Notes`
   the intent auditor wrote (per-criterion verdicts, honoured / diverged /
   missing) and the `impact` and `shipped_in` stamps are the seed. The
   phase-fidelity report this record first named does not exist; the
   per-intent audit is the audit there is, and the seed says so when a
   shipped intent carries no audit notes (criteria 1, 2).
2. **The interview**: five sections in order (went well, could improve,
   lessons, decisions, metrics), one question at a time through the host's
   question tool under the GRILL rules; the metrics section is computed
   (intents shipped, audit-note severity distribution, first and last
   `shipped_in`), not asked (criterion 1).
3. **The thin-answer rule**: an answer under a declared floor (one clause, or
   a restatement of the section heading) is met with one follow-up question
   before anything is written (criterion 4).
4. **Refusals and warnings**: a phase document naming no shipped intent
   refuses and writes nothing; a phase with a named intent still in
   `planned/` warns, lists them, and asks before proceeding; a shipped intent
   with no audit notes is offered `abcd intent audit <itd-N>` first, and the
   interview continues either way (criteria 2, 3, 7).
5. **The output**: `.abcd/development/retrospectives/<phase-id>/README.md`,
   frontmatter naming the phase, the intents, the date and the seed's
   provenance (which audits fed it), five sections, links to the phase
   document and each intent; never a copy of the audit notes (criterion 1).
6. **The nudge**: `spec close` that ships the last planned intent a phase
   document names prints one line that a retrospective is owed and the
   command to run; it is printed once (the retrospective's absence is not
   re-announced) and gates nothing (criterion 8, decision 1).
7. **The lifeboat**: `disembark` packs `.abcd/development/retrospectives/`
   whole; `embark` ranks the packed retrospectives' lessons against the new
   voyage's brief by term overlap with the brief's framing chapter and shows
   the top three with the rest as a list, asking which apply (criteria 5, 6,
   decision 2).

## Out of scope

- A phase-level fidelity audit: the seed is the per-intent audit.
- Per-intent or per-spec retrospectives.
- Editing a retrospective after it is written; a second run on the same phase
  refuses naming the existing file.

## Approach

`internal/core/reflect` holds the seed builder (reads the phase document and
the shipped intents), the metrics, the thin-answer floor and the writer; the
interview itself is host-run from `commands/reflect.md`, which renders the
seed and the five sections and calls `abcd reflect write <phase-id>
--answers <file>` with the answers. The nudge is one line in `spec close`'s
ship path. Lifeboat packing extends the record-family list `disembark`
already carries; the embark ranking is a small term-overlap score in
`internal/core/lifeboat`, declared a heuristic.

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 seeded interview and the file | scope 1, 2, 5 |
| 2 missing audit offered first | scope 4 |
| 3 empty phase refuses | scope 4 |
| 4 thin answer gets a follow-up | scope 3 |
| 5 lifeboat packs them | scope 7 |
| 6 embark shows a ranked few | scope 7 |
| 7 open work warns and asks | scope 4 |
| 8 nudge once at the last close | scope 6 |
