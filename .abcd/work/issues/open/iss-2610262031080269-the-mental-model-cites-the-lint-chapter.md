---
schema_version: 1
id: "iss-2610262031080269"
slug: "the-mental-model-cites-the-lint-chapter"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 91 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/03-mental-model.md:103"
remedy: "none (filed automatically)"
finding_ref: cfk-cb9d86b370657f57
---

The mental model cites the lint chapter for the reserved code VR001, but the lint chapter says the engine has no numbered code catalogue

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/03-mental-model.md:103` — "plus the vocabulary-registration requirement (`VR001`, [`05-internals/06-lint.md`] (../05-internals/06-lint.md))"
- End B: `.abcd/development/brief/05-internals/06-lint.md:14` — "The engine carries no numbered code catalogue: rules are named, not numbered"

The mental model points at `05-internals/06-lint.md` for the vocabulary-registration check, reserved as `VR001`. The lint contract says the engine carries no numbered code catalogue: rules are named, not numbered, and are defined only in the two JSON configs. The page the mental model cites has no place for a numbered code, reserved or not.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 91.
