---
schema_version: 1
id: "iss-2610262035127167"
slug: "itd-94-says-autonomous-drain-runs-are"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 555 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-94-an-intent-that-is-not-planned-cannot-be.md:25"
remedy: "In itd-94's press release (line 25) replace 'Autonomous drain runs get the same gate for free' with 'Autonomous runs of intents (the run protocol's step 0 and abcd build's ready check) get the same gate for free'; leave the same words in the ingested Audit Notes receipt, which is the verdict of record."
related_intents: [itd-94]
finding_ref: cfk-ee296f65d70bbebc
---

itd-94 says autonomous drain runs are held by the intent readiness gate; the intent chapter says the drain verb's lanes are issues

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-94-an-intent-that-is-not-planned-cannot-be.md:25` — "Autonomous drain runs get the same gate for free: exit 1 is a journaled SKIP, never a guess."
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:278` — "a run of the top-level drain verb never does, since its lanes are issues and an issue has no criteria."

itd-94 says autonomous drain runs get the implement-readiness gate for free, so exit 1 becomes a journaled SKIP. The intent chapter says the lanes of the top-level drain verb are issues, which have no acceptance criteria. An intent readiness gate has no intent to judge in a run whose lanes are issues, so the two documents disagree about what drain runs work on.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 555.
