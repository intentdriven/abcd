---
schema_version: 1
id: "iss-2610262039572342"
slug: "itd-34-says-its-subtype-deferral-mirrors"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 898 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/06-lint.md:14"
remedy: "none (filed automatically)"
related_intents: [itd-34]
finding_ref: cfk-5b1c212fef9eb237
---

itd-34 says its subtype deferral mirrors the lint-code namespace pattern in 06-lint; 06-lint says the engine has no code catalogue.

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/06-lint.md:14` — "The engine carries no numbered code catalogue: rules are named, not numbered"
- End B: `.abcd/development/intents/shipped/itd-34-three-intent-kinds.md:109` — "This mirrors the lint-code namespace pattern in"

itd-34 justifies deferring a closed subtype enum by pointing to the lint-code namespace pattern in 05-internals/06-lint.md, where categories are added as intents introduce them. 06-lint says the engine has no numbered code catalogue, that rules are named rather than numbered, and that any literal list belongs in a generated region. The pattern itd-34 relies on is not in the document it cites.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 898.
