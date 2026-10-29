---
schema_version: 1
id: "iss-2610272213410875"
slug: "the-layered-configuration-reader"
severity: "minor"
category: "bug"
source: "impl-review"
found_during: "review of merge-gate step 1 (itd-2610150602547271), 2026-10-27"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/layered/layered.go"
remedy: "Decide how a top-level near-miss is caught: a registry of every top-level key the repository and machine files may hold (ahoy's keys plus each claimant's namespace), refusing any other, or a near-miss check by edit distance against the claimed namespaces; ground the choice in how strict-config readers (e.g. JSON Schema additionalProperties, Cargo's unused-key warnings) handle it, and name the file and the nearest claimed namespace in the refusal."
---

The layered configuration reader silently ignores a misspelt namespace at a file's top level, such as facilitater for facilitator, or one padded with whitespace: only a claimed namespace's own spelling, its case twins and its dotted form are refused, because nothing claims a file's top level. So a person who writes {"facilitater":{"mode":"human"}} in the machine file is outvoted by a repository's auto with no word, and the same holds for every claimant's namespace. Found by the review pair of the merge gate's step 1 on 2026-10-27.
