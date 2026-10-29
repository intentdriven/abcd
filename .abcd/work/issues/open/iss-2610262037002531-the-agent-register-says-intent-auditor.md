---
schema_version: 1
id: "iss-2610262037002531"
slug: "the-agent-register-says-intent-auditor"
severity: "nitpick"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 690 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/README.md:417"
remedy: "In the agent register, list intent-auditor as serving `intent` and `build`, since the build loop composes its request and validates its verdict."
finding_ref: cfk-6aecefdab4925277
---

The agent register says intent-auditor serves `intent` alone, while the build loop composes its request and validates its verdict

A premise contradiction (severity nitpick) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/README.md:417` — "`intent` | [`05-intent.md`] (05-intent.md) | Audits a shipping intent's promise against the delivered code"
- End B: `.abcd/development/brief/04-surfaces/34-build.md:697` — "the auditor's fidelity verdict checked against the request (its receipt, both provenance hashes, every criterion and scope condition)"

README's Serves column names the verb that builds an agent's input or validates what it returns. It lists only `intent` for intent-auditor, while it lists `build` and `drain` for the two reviewers. 34-build has the validate stage hand the closing lane to an intent-auditor and check its fidelity verdict against the request before the loop records it. By the column's own definition, `build` serves the auditor as well.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 690.
