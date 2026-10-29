---
schema_version: 1
id: "iss-2610262034293679"
slug: "itd-55-says-abcd-already-stress-tests"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 500 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/drafts/itd-55-first-principles-foundations-auditor.md:17"
remedy: "In itd-55 replace 'the grill' and `/abcd:intent grill` (press release, scope, out of scope, acceptance criterion) with the planning interview (`abcd intent interview`, itd-94) and say where the foundations audit differs from the interview's questioning of vague terms and hidden assumptions (a non-interactive, load-ranked inventory over any reasoned document); also replace its 'fidelity reviewer' wording with `intent-auditor`."
related_intents: [itd-55]
finding_ref: cfk-a8ad5f2b1628ba36
---

itd-55 says abcd already stress-tests acceptance criteria with the grill; the intent chapter says no grill sub-verb ships.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/drafts/itd-55-first-principles-foundations-auditor.md:17` — "abcd already stress-tests acceptance criteria (the grill)"
- End B: `.abcd/development/brief/04-surfaces/05-intent.md:319` — "superseded by itd-94; no grill sub-verb ships"

itd-55 places its foundations auditor next to an existing grill that stress-tests acceptance criteria, and rules out duplicating `/abcd:intent grill`. 05-intent.md says the grill's intent, itd-27, was superseded by itd-94 and that no grill sub-verb ships. itd-55 defines itself against a surface that, according to the intent chapter, does not exist.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 500.
