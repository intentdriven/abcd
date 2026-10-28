---
schema_version: 1
id: "iss-2610262032133109"
slug: "itd-48-runs-both-roles-through-a-cli"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 239 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:108"
remedy: "none (filed automatically)"
related_intents: [itd-48]
finding_ref: cfk-f6e087024c69edeb
---

itd-48 runs both roles through a CLI oracle in headless Ralph mode; the dependencies page says the autonomous run is not a Ralph port

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-48-intent-fidelity-reviewer-roles-2-3.md:108` — "Both roles call `_build_cli_oracle()` from itd-47's extended version, so they run in headless Ralph mode."
- End B: `.abcd/development/brief/02-constraints/02-dependencies.md:45` — "Not a Ralph port; the native loop is the always-available fallback."

itd-48 says both roles call `_build_cli_oracle()` from itd-47 so that they run in headless Ralph mode. 02-dependencies describes the autonomous run as a pluggable seam with a native loop as the always-available fallback, and says plainly that it is not a Ralph port. itd-48 depends on a run mode that the constraints chapter rules out.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 239.
