---
id: spc-2609211751376504
slug: reflect-command
intent: itd-24
origin: researcher-authored
production_mode: hand-written
---
# reflect-command

## Summary

**Re-seeded on a release on 2026-09-29** (itd-24 decision 5, the product
thinker's ruling AD): a retrospective starts from the intents a tag shipped.
This spec was first written at the phase grain and re-read on 2026-09-21 as
the release (adr-2609212115255771); the scope below is now written at the
release grain, so no reading rule is needed.

The design record for itd-24, from the product thinker's interviews of
2026-09-21 and 2026-09-29 (decisions 1 to 5 on the intent). One host-run
interview, `/abcd:reflect <release-tag>`, seeded from what the release
actually shipped, written to the durable record tier, packed by the lifeboat
and surfaced on embark.

## Scope

1. **The seed**: the intents the tag shipped, read the way the release cut
   reads them: those that reached `shipped/` between the previous release tag
   and this one, less any whose `shipped_in` names another release, plus any
   in `shipped/` now whose `shipped_in` names this one. Amended on 2026-09-30
   under ruling AD ("seed from A RELEASE (the intents a tag shipped)"): the
   first wording read membership from `shipped_in` alone, but no cut writes
   that stamp, so it found nothing for any release cut the ordinary way; the
   stamp now moves a record between releases rather than defining membership.
   For each, the `## Audit Notes` the intent auditor wrote (per-criterion
   verdicts, honoured / diverged / missing) and its `impact`; and the
   changelog section the cut composed for the tag. The seed says so when a
   shipped intent carries no audit notes (criteria 1, 2).
2. **The interview**: five sections in order (went well, could improve,
   lessons, decisions, metrics), one question at a time through the host's
   question tool under the GRILL rules; the metrics section is computed
   (intents shipped, audit-note severity distribution, the tag's date and the
   previous tag's), not asked (criterion 1).
3. **The thin-answer rule**: an answer under a declared floor (one clause, or
   a restatement of the section heading) is met with one follow-up question
   before anything is written (criterion 4).
4. **Refusals and warnings**: a tag that shipped no intent refuses and
   writes nothing; a release with intents whose `target_release` names it
   still unshipped warns, lists them, and asks before proceeding; a shipped
   intent with no audit notes is offered `abcd intent audit <itd-N>` first,
   and the interview continues either way (criteria 2, 3, 7).
5. **The output**: `.abcd/development/retrospectives/<release-tag>/README.md`,
   frontmatter naming the tag, the intents, the date and the seed's
   provenance (which audits fed it), five sections, links to the changelog
   section and each intent; never a copy of the audit notes (criterion 1).
6. **The nudge**: one line at the end of `launch ship`, when the cut is
   written, that a retrospective for the release is owed and the command to
   run; it is printed once (the retrospective's absence is not re-announced)
   and gates nothing (criterion 8, decision 1).
7. **The lifeboat**: `disembark` packs `.abcd/development/retrospectives/`
   whole; `embark` ranks the packed retrospectives' lessons against the new
   voyage's brief by term overlap with the brief's framing chapter and shows
   the top three with the rest as a list, asking which apply (criteria 5, 6,
   decision 2).

## Out of scope

- An audit of its own: the seed is the per-intent audit.
- Per-intent or per-spec retrospectives.
- A date or tag range as the seed, and phase documents as the seed (the
  alternatives decision 5 did not take).
- Editing a retrospective after it is written; a second run on the same tag
  refuses naming the existing file.

## Approach

`internal/core/reflect` holds the seed builder (reads the tag and the
intents it shipped, per scope 1), the metrics, the thin-answer floor and
the writer; the interview itself is host-run from `commands/reflect.md`,
which renders the seed and the five sections and calls `abcd reflect write
<release-tag> --answers <file>` with the answers. The nudge is one line in
`launch ship`. Lifeboat packing extends the record-family list `disembark`
already carries; the embark ranking scores with the canonical term-overlap
primitive, `internal/core/record/match` (spc-2609212141417782), declared a
heuristic.

## How the criteria are satisfied

| Criterion | Where |
| --- | --- |
| 1 seeded interview and the file | scope 1, 2, 5 |
| 2 missing audit offered first | scope 4 |
| 3 empty release refuses | scope 4 |
| 4 thin answer gets a follow-up | scope 3 |
| 5 lifeboat packs them | scope 7 |
| 6 embark shows a ranked few | scope 7 |
| 7 open work warns and asks | scope 4 |
| 8 nudge once at the cut | scope 6 |
