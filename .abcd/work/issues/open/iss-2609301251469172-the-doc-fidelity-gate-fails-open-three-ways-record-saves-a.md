---
schema_version: 1
id: "iss-2609301251469172"
slug: "the-doc-fidelity-gate-fails-open-three-ways-record-saves-a"
severity: "major"
category: "security"
source: "impl-review"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/docfidelity/store.go"
remedy: "Refuse at Record a PROMOTE whose failing list names a brief sentence (the mirror of the HOLD-with-no-sentence refusal) and have Judge refuse a match carrying one; drop the backlog mechanism, since the spec (spc-2609020903498198) has no backlog, judges the brief against the binary and never the tag (the legitimate lead), and the file is absent with every surface named; refuse a sentence or replacement carrying a line break or longer than a fixed bound at Record and at Apply. Grounds: the review-docFidelity probes, each reproduced as a failing test first; shown wrong if any probe still saves or passes."
---

the doc-fidelity gate fails open three ways: Record saves a PROMOTE whose failing list names a false brief sentence and Judge lets that match proceed; a new undocumented surface listed in doc-fidelity-backlog.json passes layer 1 as backlog with no chapter, and nothing enforces the claim that the list only shrinks; and Record accepts a multi-line or unbounded sentence, which Apply then replaces across several lines of a chapter
