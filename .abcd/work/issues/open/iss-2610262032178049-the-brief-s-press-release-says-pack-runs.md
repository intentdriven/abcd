---
schema_version: 1
id: "iss-2610262032178049"
slug: "the-brief-s-press-release-says-pack-runs"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 247 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/01-product/01-press-release.md:23"
remedy: "none (filed automatically)"
related_intents: [itd-88]
finding_ref: cfk-c7ac8bc08ed5da98
---

The brief's press release says pack runs three passes, including targeted chat retrieval over transcripts; itd-88 says Pass B ships only as a declared exemption

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/01-product/01-press-release.md:23` — "runs three passes (settled artefacts → targeted chat retrieval → distil/compose/audit) over a project's specs, ADRs, transcripts, oracle reviews, and curated memory"
- End B: `.abcd/development/intents/shipped/itd-88-lifeboat-coverage-experiment.md:46` — "there is no corpus. It ships as a **declared exemption** in `_provenance.json`, never a silent gap, until the transcript store has data."

The brief's press release scopes `/abcd:disembark pack` as three passes (settled artefacts → targeted chat retrieval → distil/compose/audit) over specs, ADRs and transcripts, with no staged marker. itd-88 says the transcript-mining pass (Pass B) has no corpus and ships as a declared exemption in `_provenance.json` until the transcript store has data. The pack cannot both run the chat-retrieval pass and exempt it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 247.
