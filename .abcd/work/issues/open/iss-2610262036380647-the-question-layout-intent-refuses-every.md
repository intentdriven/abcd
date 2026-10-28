---
schema_version: 1
id: "iss-2610262036380647"
slug: "the-question-layout-intent-refuses-every"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 663 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/intents/shipped/itd-2610030810350727-every-question-abcd-puts-to-a-person.md:32"
remedy: "none (filed automatically)"
related_intents: [itd-2610030810350727]
finding_ref: cfk-ac83840b3b505e4f
---

The question-layout intent refuses every question that breaks the asking rules; the guard admits a question whose only fault is going over the rows limit.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/intents/shipped/itd-2610030810350727-every-question-abcd-puts-to-a-person.md:32` — "A question that breaks these rules is refused before it reaches you, naming the part and the limit"
- End B: `.abcd/development/brief/04-surfaces/17-guard.md:153` — "The rows limit is the one finding that does not refuse on its own"

The intent promises that a question breaking its rules is refused before it reaches the person, and that such a question reads at 80 columns without scrolling, with height handled by tabs. The guard chapter makes the rows limit the one finding that does not refuse on its own. A question whose only findings are tabs over the rows limit is admitted and marked open, and the agent is told only after it returns. So an over-tall question does reach the person, which the intent says cannot happen.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 663.
