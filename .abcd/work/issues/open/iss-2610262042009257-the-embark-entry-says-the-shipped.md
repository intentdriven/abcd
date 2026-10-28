---
schema_version: 1
id: "iss-2610262042009257"
slug: "the-embark-entry-says-the-shipped"
severity: "minor"
category: "inconsistency"
source: "agent-finding"
found_during: "abcd intent consistency, finding 1039 of .abcd/work/reviews/2026-10-26-consistency/00-summary.md"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/glossary/interview/embark.md:26"
remedy: "none (filed automatically)"
related_intents: [itd-2610030810350727]
finding_ref: cfk-bfd39cec1d4ae499
---

The embark entry says the shipped interviews ask one thing at a time, not in rounds; the question-layout intent lets up to four tabbed questions share a screen.

A premise contradiction (severity minor) found by the cross-document consistency pass.

- End A: `.abcd/development/brief/glossary/interview/embark.md:26` — "Both forms ask one thing at a time, not in rounds."
- End B: `.abcd/development/intents/shipped/itd-2610030810350727-every-question-abcd-puts-to-a-person.md:103` — "use tabs for parts of one thing, such as the criteria of one feature or the paragraphs of one text, up to four per screen, each short"

The interview-context embark entry says both shipped forms of the planning interview ask one thing at a time, not in rounds. The shipped question-layout intent rules that parts of one thing, such as the criteria of one feature, are put as tabs, up to four per screen. It says this narrows the GRILL rule's 'one at a time' to dependent questions. The glossary therefore describes the shipped interviews by a rule the intent amended.

Evidence: `.abcd/work/reviews/2026-10-26-consistency/00-summary.md`, finding 1039.
