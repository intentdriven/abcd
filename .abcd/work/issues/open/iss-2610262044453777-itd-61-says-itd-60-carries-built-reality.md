---
schema_version: 1
id: "iss-2610262044453777"
slug: "itd-61-says-itd-60-carries-built-reality"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1227 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-61-brief-change-derivation.md:44"
remedy: "none (filed automatically)"
related_intents: [itd-61, itd-60]
finding_ref: cfk-890a06ca52b5bf84
---

itd-61 says itd-60 carries built reality into the brief and the public docs; shipped itd-60 covers the brief only and leaves the public docs to a later rung

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-61-brief-change-derivation.md:44` — "Carrying built reality into the brief and the public docs is [[itd-60-doc-fidelity-anti-drift]]"
- End B: `.abcd/development/intents/shipped/itd-60-doc-fidelity-anti-drift.md:144` — "The public docs — a later rung, reported and never refusing."

itd-61 leaves the forward direction out of its own scope, public docs included, because it says itd-60 covers it. Shipped itd-60 covers the brief only. A lagging public-doc sentence is reported and never refuses, and drafting public-doc changes was moved to a later rung. So neither document carries built reality into the public docs, although itd-61 says one does.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1227.
