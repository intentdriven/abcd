---
schema_version: 1
id: "iss-2610262033578037"
slug: "itd-190-makes-a-present-but-empty"
severity: "major"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 456 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/disciplines/itd-190-the-claim-recording-gradient-an-intent-s.md:24"
remedy: "none (filed automatically)"
related_intents: [itd-190]
finding_ref: cfk-82c0658081edca7a
---

itd-190 makes a present-but-empty Mechanism section a readiness-gate fault; the intent template says a blank section passes the readiness gate with the nullity recorded

A premise contradiction (severity major) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/disciplines/itd-190-the-claim-recording-gradient-an-intent-s.md:24` — "Prompted, nullable — declined with the nullity token; a present-but-empty section is a gate fault"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:411` — "A blank section passes the readiness gate with the nullity recorded"

itd-190 separates three byte states for a claim section (absent; present but empty, which is a gate fault; the nullity token, which is a claim declined) and says they are never collapsed. The intent surface's template says a blank Mechanism section passes the readiness gate with the nullity recorded, which treats empty as the nullity. One gate cannot both fault and pass the same empty section.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 456.
