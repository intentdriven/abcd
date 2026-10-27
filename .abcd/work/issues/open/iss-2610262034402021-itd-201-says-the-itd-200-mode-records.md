---
schema_version: 1
id: "iss-2610262034402021"
slug: "itd-201-says-the-itd-200-mode-records"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 514 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-201-every-question-abcd-s-agents-put-to-a.md:65"
remedy: "none (filed automatically)"
related_intents: [itd-201]
finding_ref: cfk-6bc9280146934b0c
---

itd-201 says the itd-200 mode records which hat the human wears so it is not asked again; the board chapter says answering any abcd question resets the mode to managed.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-201-every-question-abcd-s-agents-put-to-a.md:65` — "the itd-200 mode records the answer so the next question does not ask again"
- End B: `.abcd/development/brief/04-surfaces/08-abcd.md:178` — "finds the marker, resets the state to managed, clears the marker"

itd-201's decision has the agent ask which hat the human wears and rely on the itd-200 mode to record the answer so the next question does not ask again. 08-abcd states that when abcd's own question is admitted the state is set from the chip's role, and when the answer comes back the hook finds the marker and resets the state to managed. The mode holds only managed or whom a question waits on, and returns to managed after every answer, so it cannot carry the hat answer forward to the next question.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 514.
