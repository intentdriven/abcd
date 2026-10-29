---
schema_version: 1
id: "iss-2610262039504627"
slug: "itd-40-says-identity-and-lineage-live"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 890 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md:441"
remedy: "Edit 03-configuration.md (The history store): replace 'one identity-and-lineage file per repo' with 'one meta.json per repo (identity labels and the corpus pointer)' and say lineage (supersedes, superseded_by, prior names) lives in the registry entry in index.json."
related_intents: [itd-40]
finding_ref: cfk-a8cc9f1f15e3d872
---

itd-40 says identity and lineage live directly in the history `index.json`; 03-configuration puts lineage in a separate file per repository under that key.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/03-configuration.md:441` — "root-commit SHA, and one identity-and-lineage file per repo under that key."
- End B: `.abcd/development/intents/shipped/itd-40-folder-classification.md:42` — "the history `index.json` is the single registry; root-SHA identity and lineage live there directly."

itd-40 says identity is not duplicated because the history `index.json` is the single registry, with identity and lineage held there directly, and it points to 03-configuration. 03-configuration says the store holds a registry keyed on the root-commit SHA plus one identity-and-lineage file per repository under that key. Lineage cannot be both directly in the index and in a separate per-repository file.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 890.
