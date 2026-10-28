---
schema_version: 1
id: "iss-2610262031060442"
slug: "the-press-release-lists-golden-test"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 86 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:27"
remedy: "none (filed automatically)"
finding_ref: cfk-c4ca8f72231bafc5
---

The press release lists golden-test fixtures, a periodic SOTA audit and a self-improvement pre-flight as in-scope plumbing, but the prompt-quality chapter says those layers are staged

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:27` — "a prompt-quality stack with golden-test fixtures, structural lint, periodic SOTA audit, prompt-version frontmatter, self-improvement pre-flight"
- End B: `.abcd/development/brief/05-internals/05-prompt-quality.md:19` — "Layer C ships and is armed as a blocker; layers B and D are staged"

The press release lists, among the plumbing that makes abcd possible, a prompt-quality stack with golden-test fixtures, a periodic SOTA audit and a self-improvement pre-flight. None of them carries a staged marker. The prompt-quality chapter says only layer C (the linter) ships. Layer B (golden tests) and layer D (the SOTA audit) are staged, and the pre-flight is staged too. The brief's truth rule requires a staged claim to say so, so the two pages give different answers to what exists.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 86.
