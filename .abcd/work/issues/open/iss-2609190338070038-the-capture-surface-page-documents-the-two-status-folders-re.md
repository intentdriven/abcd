---
schema_version: 1
id: "iss-2609190338070038"
slug: "the-capture-surface-page-documents-the-two-status-folders-re"
severity: "minor"
category: "documentation"
source: "agent-observation"
found_during: "Gropius autonomous sweep, session gropiusllm-66, relayed to abcd-17 on 2026-09-19"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/capture.md"
---

The capture surface page documents the two-status-folders refusal but not the convention that avoids it: a record is resolved on the branch that carries it. commands/capture.md explains why an id in open/ and resolved/ at once is refused and how the merge artefact arises, and stops there; the rule that prevents it (never re-add a record to the default branch after a branch was cut from it; resolve it on the branch that carries it) lives only in this repository's AGENTS.md, which a managed repository does not inherit. In the Gropius sweep of 2026-09-19 (session gropiusllm-66, forty lanes) a record captured on an unmerged branch could not be resolved from another branch without producing exactly that duplicate once both landed, so lanes merged each other's branches to resolve. Wanted, either: one paragraph on the capture page stating the convention for a managed repository, or a resolve that tolerates a record absent from this checkout when told where it is (the session's --expect-on-branch), which is the ledger-visibility capability itd-2609091416295622 already weighs. This record is the documentation half.
