---
schema_version: 1
id: "iss-2610262032256736"
slug: "launch-names-its-docs-lint-gate-row-the"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 267 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/04-launch.md:329"
remedy: "none (filed automatically)"
finding_ref: cfk-b7e32d9d008e16d8
---

Launch names its docs-lint gate row the documentation audit; the intent chapter keeps audit and lint as separate verbs, the conformance check being lint

A terminology drift (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/04-launch.md:329` — "**The documentation audit** is the docs-lint engine over the repository's configured doc roots"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:534` — "The top-level conformance check is `abcd lint`, with `/abcd:audit` reserved for itd-16's hash-chain fidelity checks"

The intent chapter's statement of the verb register says the top-level conformance check is abcd lint, the audit name is reserved for other checks, and each verb means one thing. Launch calls its docs-lint engine row the documentation audit, and its acceptance speaks of a documentation-auditor warning, using audit for a lint-bucket conformance check.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 267.
