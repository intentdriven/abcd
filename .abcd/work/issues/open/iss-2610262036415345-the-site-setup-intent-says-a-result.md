---
schema_version: 1
id: "iss-2610262036415345"
slug: "the-site-setup-intent-says-a-result"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 666 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2609061543533170-abcd-sets-up-a-managed-repository-s.md:70"
remedy: "none (filed automatically)"
related_intents: [itd-2609061543533170]
finding_ref: cfk-eb96e282b0cfeeae
---

The site-setup intent says a result followed by the error envelope is the repository-wide JSON convention; the update chapter emits a refusal as a single document.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-2609061543533170-abcd-sets-up-a-managed-repository-s.md:70` — "A `--json` run that exits 1 prints two JSON documents, the result and then the CLI's error envelope; that is the repository-wide convention"
- End B: `.abcd/development/brief/04-surfaces/21-update.md:238` — "In the JSON form a refusal of either kind is one document on stdout"

The site-setup intent tells a strict `--json` consumer that a run exiting 1 prints two documents, the result and then the CLI's error envelope, and calls this the convention across the repository. The update chapter says a refusal in JSON is one document on stdout, carrying the envelope's fields beside its own, so that no receipt is followed by a second envelope. A consumer built to the claimed repository-wide convention would misread update's refusals. So the convention is either not repository-wide or update breaks it.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 666.
