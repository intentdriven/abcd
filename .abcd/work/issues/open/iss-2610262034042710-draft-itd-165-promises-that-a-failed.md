---
schema_version: 1
id: "iss-2610262034042710"
slug: "draft-itd-165-promises-that-a-failed"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 467 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-165-an-intent-that-fails-its-own-fidelity.md:12"
remedy: "Supersede or re-scope itd-165 with `abcd intent reclassify`: the audit ingest already captures a major bug for a NOT_MET criterion and a minor inconsistency for an INCONCLUSIVE one, so the draft's premise that no record is created and that an inconclusive verdict files nothing is false; keep only a gate or ratchet remainder if one is wanted."
related_intents: [itd-165]
finding_ref: cfk-789e69b5eb7d3ad8
---

draft itd-165 promises that a failed fidelity verdict becomes visible work; the intent surface's audit ingest already files an issue for every failed or undecided check

A scope leakage (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-165-an-intent-that-fails-its-own-fidelity.md:12` — "A failed fidelity verdict becomes work somebody can see"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:330` — "That issue is captured through the ledger's one filer"

itd-165 claims, as a capability still to be planned, that a failed fidelity verdict becomes work somebody can see. The intent surface says an ingested verdict judging any criterion NOT_MET or INCONCLUSIVE flags the block and captures an issue carrying the check through the ledger's one filer. The same ground is covered twice, once as shipped and once as a draft.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 467.
