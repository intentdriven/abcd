---
schema_version: 1
id: "iss-257"
slug: "citation-metadata-and-references"
severity: "minor"
category: "documentation"
source: "agent-observation"
found_during: "v0.5.1 derived cut reconciliation"
found_at: "CITATION.cff"
resolution: "Delivered whole in the references-baseline change: the root CITATION.cff (CFF 1.2.0) powering the forge's cite-this-repository box, the References & sources section of ACKNOWLEDGEMENTS.md (curated, alphabetically ordered primary sources), canonical CSL-JSON metadata at .abcd/development/research/references.csl.json, and the documented on-demand .bib export for LaTeX toolchains. This record is the derived changelog's citable home for the capability; the hand-written Unreleased entry it replaces is removed in the same cut."
impact: additive
---

abcd is citable and its sources are recorded: a root CITATION.cff (CFF 1.2.0) powers the forge's cite-this-repository box; the References & sources section of ACKNOWLEDGEMENTS.md records the academic literature the design record draws on as a curated, alphabetically ordered list of primary sources; canonical CSL-JSON metadata lives in .abcd/development/research/references.csl.json with a documented on-demand render path. Filed retroactively: the capability shipped with a hand-written Unreleased entry and no record, which the derived release flow refuses — this record is the entry's citable home.