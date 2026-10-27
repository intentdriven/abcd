---
schema_version: 1
id: "iss-2610262031058243"
slug: "the-press-release-puts-abcd-run"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 84 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:27"
remedy: "none (filed automatically)"
finding_ref: cfk-496ba1cb9cff6fc8
---

The press release puts `/abcd:run` operator-internal wiring in scope, but the surfaces index lists every registered verb and `run` is not among them

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:27` — "operator-internal command wiring (e.g. `/abcd:run`, the itd-29 autonomous-run operator surface"
- End B: `.abcd/development/brief/04-surfaces/README.md:269` — "so the two tables together name every verb the binary registers apart from the framework's own `help`"

The press release's in-scope plumbing names operator-internal command wiring, giving `/abcd:run` (the itd-29 autonomous-run surface with status/pause/resume/preflight) as its example. The surfaces index says its two tables together name every verb the binary registers apart from the framework's `help`. Its operator-internal table holds only rules, spec, hook, completion and statusline. A `run` verb cannot be delivered scope and also missing from a register that claims to be complete.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 84.
