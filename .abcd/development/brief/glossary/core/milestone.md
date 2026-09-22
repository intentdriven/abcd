---
term: milestone
bounded_context: core
definition: A planned end condition for a stretch of work; retired: the checkpoint is the derived release plus each intent's acceptance criteria.
aliases: []
forbidden_synonyms: []
status: superseded
introduced_in: adr-9
starts_when: null
ends_when: null
not_to_be_confused_with: core/record-families
versions: null
---
<!-- Adapted from mattpocock/skills (MIT). See README Acknowledgements. -->

# milestone

> **Superseded on 2026-09-21 (adr-2609212115255771).** The word never had an entry of its own; it lived as a forbidden synonym of phase. Its job, a checkable end condition, is done twice over: each intent's acceptance criteria say when that work is done, and the derived release says what shipped. An intent that must land by a cut says so with `target_release` (itd-2609212103572513), which the cut reports and carries forward. See [record-families](record-families.md).
