---
schema_version: 1
id: "iss-2610262032400242"
slug: "universal-patterns-says-the-voyage-store"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 302 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/04-universal-patterns.md:161"
remedy: "none (filed automatically)"
finding_ref: cfk-beeb05409782f6ca
---

universal patterns says the voyage store is tracked under the visibility rule; embark says it is never committed

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/05-internals/04-universal-patterns.md:161` — "Both are tracked under the visibility rule"
- End B: `.abcd/development/brief/04-surfaces/03-embark.md:264` — "root-commit SHA exactly as the history store is, and therefore never committed"

Universal patterns cites embark section 7 and says both the log tier and ~/.abcd.noindex/voyage/< source-root-sha>/ are tracked under the visibility rule. The embark chapter says the voyage store lives at the operator level, keyed on the root commit, and is therefore never committed. A store outside every repository cannot be committed, or ignored, under a repository's visibility rule.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 302.
