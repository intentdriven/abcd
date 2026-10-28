---
schema_version: 1
id: "iss-2610262038051992"
slug: "itd-94-says-autonomous-drain-runs"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 771 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-94-an-intent-that-is-not-planned-cannot-be.md:25"
remedy: "none (filed automatically)"
related_intents: [itd-94]
finding_ref: cfk-95376c04c0ff6daf
---

itd-94 says autonomous drain runs inherit the implement-readiness gate, exit 1 a journaled SKIP; 35-drain's lanes are issue runs checked only by the drain rule and the peers check

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-94-an-intent-that-is-not-planned-cannot-be.md:25` — "Autonomous drain runs get the same gate for free: exit 1 is a journaled SKIP, never a guess."
- End B: `.abcd/development/brief/04-surfaces/35-drain.md:561` — "Its checks are the rule above, read the same way, and the peers check"

itd-94 presents the intent implement-readiness gate (`abcd intent ready`) as one autonomous drain runs get for free, a NOT READY exit 1 becoming a journaled SKIP. 35-drain's drain works open issues, each lane being the run `abcd build <iss-N>` starts, whose checks are the drain rule and the peers check; no readiness gate applies to such a lane, and the drain as written has no intent run for itd-94's gate to stop.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 771.
