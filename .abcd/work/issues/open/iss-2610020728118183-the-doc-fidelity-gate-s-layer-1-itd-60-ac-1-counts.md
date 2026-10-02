---
schema_version: 1
id: "iss-2610020728118183"
slug: "the-doc-fidelity-gate-s-layer-1-itd-60-ac-1-counts"
severity: "minor"
category: "drift"
source: "review-followup"
found_during: "autonomous run A resumed 2026-09-25: fidelity audit itd-60"
origin: researcher-authored
production_mode: hand-written
remedy: "Exclude README.md (the index) from the chapter set the gate reads, or read only the chapters the surfaces index lists as chapters, so a surface the index alone names refuses with 'no brief chapter names'; then give 'abcd spec', 'abcd rules' and 'sota-researcher' a chapter (or a documented home in an existing one) so the gate passes on the merit. Grounds: spc-2609020903498198 scope 'Every verb, sub-verb and agent must be named by a chapter under the brief's 04-surfaces/' and invariant 18, under which only a chapter carries the generated appendix of a surface's flags."
---

The doc-fidelity gate's layer 1 (itd-60, ac-1) counts 04-surfaces/README.md, the surfaces index, as a chapter: ReadInputs lists every regular *.md under 04-surfaces/ (internal/core/docfidelity/store.go:70) and names() accepts a code span in any of them, so at main 7fb52a6b5 the verb 'abcd spec', its sub-verb 'abcd spec close', the verb 'abcd rules' and the agent 'sota-researcher' pass the coverage floor on the README alone (abcd docs fidelity --json: coverage rows with chapter README.md), while no numbered chapter names them and no generated appendix (invariant 18) states their flags. A one-row index entry satisfies the floor for any new surface, which is the placeholder the spec's 'named by a chapter under 04-surfaces/' exists to refuse.
