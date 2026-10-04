# Brief-to-record traceability: state of the art (2026-10-04)

A state-of-the-art pass run for the product thinker's working hypothesis of 2026-10-04 (recorded on itd-61, itd-142 and itd-143, and in the decision log): the brief is the target design, written in present tense, and abcd deterministically traces each section to the work that realises it. Feeds itd-2610040817105016.

## Finding

The hypothesis is settled practice. Amazon's PR/FAQ writes the product "as though it already exists"; OpenSpec keeps current truth as SHALL statements and proposed change as deltas; requirements tools keep "delivered" out of the prose and in per-unit links and metadata. What abcd lacks is the mechanics: no brief section carries a stable identifier, and no intent names the brief section it realises.

## Recommendations, ranked

1. A stable identifier per brief section, independent of its heading text (an explicit marker, since auto-generated anchors change on rename and some hosts strip `{#id}`). The product thinker's choice: granularity (chapter, heading, paragraph); start at heading.
2. Intents carry a `realises` link to a section, stamped with a fingerprint of the section's text at link time, as Doorstop does ("the link fingerprint is used to detect when a parent item is changed"). DOORS and Polarion call the result a suspect link.
3. A deterministic coverage report per section (shipped, planned, drafts; unclaimed sections; suspect links), in the shape of OpenFastTrace's ok / uncovered / outdated. Folder membership already is abcd's status, so the counts need no new state.
4. The brief stays present tense; status lives in the links, never in the prose. The product thinker's choice: one document with a status overlay (Amazon) or a vision plus change folders (OpenSpec).
5. On a brief edit: the suspect set is deterministic (fingerprints); the classification (new intent, issue, contradiction owing a decision, no impact) is a model's proposal the product thinker confirms, "no impact" included. Published recall for impact analysis is about 86 to 96 per cent with human review; contradiction detection about 60 per cent recall, so a model's "no impact" misses roughly one contradiction in three. Spec Kit's `analyze` is the auditable shape: read-only, a table of id, category, severity, location and recommendation, never applied automatically.
6. Write-back is mechanical: closing a spec re-stamps the brief link in the same act, because the main critique of living specifications is the upkeep nobody does.

## Version 1 and later

Version 1: heading-level identifiers; `realises` with a fingerprint; the coverage report; a lint listing suspect links; an assessment verb that proposes classifications for the product thinker to confirm. Later: paragraph identifiers, code-to-brief traces, formal contradiction checks, links across repositories.

## Risks

Suspicion fatigue (normalise whitespace before fingerprinting; one confirmed "re-stamp all, no impact" act); the recall ceiling (the assessment is a net, not a proof, and says so); the one-off migration of the brief's 111 files to identifiers.

## Not adopted

Requirements tools as dependencies (own formats or runtimes; borrow the fingerprint only); a revision number inside the identifier; specification-as-source code generation; future tense or status prefixes in brief prose; vendor "bidirectional auto-sync" claims without method.

## Sources

Doorstop item reference (github.com/doorstop-dev/doorstop, docs/reference/item.md); OpenFastTrace (github.com/itsallcode/openfasttrace); OpenSpec (github.com/Fission-AI/OpenSpec); OSLC RM change propagation (archive.open-services.net); ProReFiCIA (arXiv 2511.00262); ALICE (TU Berlin depositonce); RWTH NLI for requirements consistency; GitHub Spec Kit `analyze` template; Thoughtworks Technology Radar, spec-driven development (November 2025); Augment, "What spec-driven development gets wrong"; Chromium documentation best practices; Amazon PR/FAQ; CommonMark anchors discussion; sphinx-needs changelog; DO-178C traceability (Parasoft).
