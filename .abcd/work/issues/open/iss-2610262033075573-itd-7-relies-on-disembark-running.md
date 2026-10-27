---
schema_version: 1
id: "iss-2610262033075573"
slug: "itd-7-relies-on-disembark-running"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 362 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/02-disembark.md:11"
remedy: "none (filed automatically)"
related_intents: [itd-7]
finding_ref: cfk-75a5ceac395beff7
---

itd-7 relies on disembark running dev-sync before the pack and writing a workspace file into the repo; the disembark chapter has no such step and never writes to the source

A sequencing impossibility (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/04-surfaces/02-disembark.md:11` — "writes to the source, and a test fingerprints the source tree before and after"
- End B: `.abcd/development/intents/planned/itd-7-rp-workspace-portability.md:38` — "dev-sync runs as a pre-pack step (already in brief). Workspace state goes into the lifeboat naturally."

itd-7 says dev-sync runs as a pre-pack step of disembark, that this step is already in the brief, and (in its press release) that it writes `.abcd/rp/workspace.json` into the repository so the workspace reaches the lifeboat. The disembark chapter says the packer never writes to the source, which a before-and-after fingerprint of the source tree proves. It describes packing as one deterministic pass (inventory, destination gate, secret scan, write, voyage line) whose output paths come from the brief-section mapping, with no pre-pack step and no place in the lifeboat for a workspace file. As written, disembark cannot provide the step itd-7 depends on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 362.
