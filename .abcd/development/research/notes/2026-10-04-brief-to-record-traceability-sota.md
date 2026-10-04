# Brief-to-record traceability: state of the art (2026-10-04)

A state-of-the-art pass run for the product thinker's working hypothesis of 2026-10-04 (recorded on itd-61, itd-142 and itd-143, and in the decision log): the brief is the target design, written in present tense, and abcd deterministically traces each section to the work that realises it. Feeds itd-2610040817105016.

## Finding

The hypothesis is settled practice. Amazon's PR/FAQ writes the product "as though it already exists"; OpenSpec keeps current truth as SHALL statements and proposed change as deltas; requirements tools keep "delivered" out of the prose and in per-unit links and metadata. What abcd lacks is the mechanics: no brief section carries a stable identifier, and no intent names the brief section it realises.

## Recommendations, ranked

1. A stable identifier per brief section, independent of its heading text (an explicit marker, since auto-generated anchors change on rename and some hosts strip `{#id}`). The product thinker's choice: granularity (chapter, heading, paragraph); start at heading.
2. Intents carry a `realises` link to a section, stamped with a fingerprint of the section's text at link time, as Doorstop does ("the link fingerprint is used to detect when a parent item is changed"). DOORS and Polarion call the result a suspect link.
3. A deterministic coverage report per section (shipped, planned, drafts; unclaimed sections; suspect links), in the shape of OpenFastTrace's ok / uncovered / outdated. Folder membership already is abcd's status, so the counts need no new state.
4. The brief stays present tense; status lives in the links, never in the prose. The product thinker's choice: one document with a status overlay (Amazon) or a vision plus change folders (OpenSpec).
5. On a brief edit: the suspect set is deterministic (fingerprints); the classification (new intent, issue, contradiction owing a decision, no impact) is a model's proposal the product thinker confirms, "no impact" included. No published baseline covers classifying design-document edits against work items. The nearest figures, as corrected on 2026-10-04 against the primary sources (see "Correction" below): automated change-impact analysis reaches 86 to 96 per cent recall on one dataset with 33 impacted requirements, with precision under 50 per cent (ProReFiCIA); pairwise contradiction detection reaches about 65 per cent recall for a logic-and-model hybrid and 0 per cent for the model alone (ALICE); conflict detection on natural data reaches F1 22 to 55. So a model's "no impact" cannot be trusted alone. Spec Kit's `analyze` is the auditable shape: read-only, a table of id, category, severity, location and recommendation, never applied automatically.
6. Write-back is mechanical: closing a spec re-stamps the brief link in the same act, because the main critique of living specifications is the upkeep nobody does.

## Correction (2026-10-04)

The first version of this note misstated two figures; a downstream lab session checked them against the primary sources.

- ProReFiCIA (Etezadi, Abualhaija, Arora, Briand; arXiv 2511.00262, ACM TOSEM 2026): 85.7 per cent recall, 95.8 per cent with retrieval, is fully AUTOMATED recall against ground truth, not recall "with human review"; the human appears only as the reviewer the cost metric assumes (3.0 to 3.4 per cent of requirements per change). One satellite dataset: 192 requirements, 11 change rationales, 33 impacted. Recall is macro-averaged per rationale (84.8 and 90.9 per cent pooled). Precision is unreported; from the paper's counts it is about 48 and 46 per cent. A flag-everything baseline reaches 86.3 per cent recall at 99 per cent cost, so a recall figure needs its flag rate beside it.
- ALICE (Gärtner and Göhlich, Automated Software Engineering 31:49, 2024): the recall is that of a formal-logic and GPT-3 hybrid; GPT-3 alone scored 0 per cent. The task is binary pairwise contradiction over 1,071 controlled-language requirement pairs with 26 contradictions (2.4 per cent). The confusion matrix gives 17 of 26, 65.4 per cent (Wilson 95 per cent interval 46 to 81), against the paper's own "60 per cent"; the 99 per cent accuracy sits on a 97.6 per cent always-no baseline. "One in three missed" matches the matrix (9 of 26), not the 60 per cent.
- Context: on synthetic contradictions GPT-4o reaches F1 96.8 (RWTH, Volkenandt and Rumpe, 2026); on natural conflict data zero-shot NLI reaches F1 22 to 55 and ChatGPT F1 9 to 30 (Fazelnia et al., RE 2024); contradictions across three or more statements are a documented blind spot; trace-link recovery between requirement levels reaches a best average F1 of 0.451 (Hey et al., REFSQ 2025). None of the vendor or practitioner sources listed below publishes an accuracy figure.

## Version 1 and later

Version 1: heading-level identifiers; `realises` with a fingerprint; the coverage report; a lint listing suspect links; an assessment verb that proposes classifications for the product thinker to confirm. Later: paragraph identifiers, code-to-brief traces, formal contradiction checks, links across repositories.

## Risks

Suspicion fatigue (normalise whitespace before fingerprinting; one confirmed "re-stamp all, no impact" act); the recall ceiling (the assessment is a net, not a proof, and says so); the one-off migration of the brief's 111 files to identifiers.

## Not adopted

Requirements tools as dependencies (own formats or runtimes; borrow the fingerprint only); a revision number inside the identifier; specification-as-source code generation; future tense or status prefixes in brief prose; vendor "bidirectional auto-sync" claims without method.

## Sources

Doorstop item reference (github.com/doorstop-dev/doorstop, docs/reference/item.md); OpenFastTrace (github.com/itsallcode/openfasttrace); OpenSpec (github.com/Fission-AI/OpenSpec); OSLC RM change propagation (archive.open-services.net); ProReFiCIA (arXiv 2511.00262); ALICE (TU Berlin depositonce); RWTH NLI for requirements consistency; GitHub Spec Kit `analyze` template; Thoughtworks Technology Radar, spec-driven development (November 2025); Augment, "What spec-driven development gets wrong"; Chromium documentation best practices; Amazon PR/FAQ; CommonMark anchors discussion; sphinx-needs changelog; DO-178C traceability (Parasoft).
