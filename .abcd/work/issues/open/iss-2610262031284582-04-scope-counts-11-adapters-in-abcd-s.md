---
schema_version: 1
id: "iss-2610262031284582"
slug: "04-scope-counts-11-adapters-in-abcd-s"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 133 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/04-scope.md:32"
remedy: "none (filed automatically)"
finding_ref: cfk-efda7bffde7eed24
---

04-scope counts 11 adapters in abcd's plumbing; the build sequence says the scanner is the one capability seam standing beside four further adapter packages, the other seams being planned.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/04-scope.md:32` — "11 adapters, harness shim, prompt-quality stack, hooks"
- End B: `.abcd/development/brief/06-delivery/01-build-sequence.md:35` — "Four further adapter packages stand outside the capability seams"

04-scope lists the plumbing as eighteen agents and '11 adapters, harness shim, prompt-quality stack, hooks'. The build sequence says the scanner is the one capability seam standing, 'Four further adapter packages stand outside the capability seams' (openaiapi, hosting, repoprompt and tailscale), and the oracle seam's package and the history, spec and run seams are planned rather than present. A count of eleven and an enumeration of five cannot both describe the current plumbing.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 133.
