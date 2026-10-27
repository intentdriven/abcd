---
schema_version: 1
id: "iss-2610262045199145"
slug: "itd-46-says-quoted-text-alone-creates-a"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1267 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:20"
remedy: "none (filed automatically)"
related_intents: [itd-46, itd-82]
finding_ref: cfk-4e40104d42a8d32d
---

itd-46 says quoted text alone creates a capture; itd-82 requires a remedy field on every new issue

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-46-abcd-intent-quoted-text-create-symmetric.md:20` — "Quotes mean create. That's the whole rule for both commands."
- End B: `.abcd/development/intents/planned/itd-82-drain-ledger-triage.md:44` — "and requires it of every new issue (ruling BX3 of 2026-09-29)"

itd-46 makes quoted text the whole create path for capture and intent: quotes mean create, and that is the whole rule for both commands. itd-82 has capture require the remedy field of every new issue (ruling BX3). Quoted text with no remedy can no longer create an issue, so quotes alone are no longer the whole rule for capture.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1267.
