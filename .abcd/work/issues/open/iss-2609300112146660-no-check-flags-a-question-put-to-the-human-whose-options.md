---
schema_version: 1
id: "iss-2609300112146660"
slug: "no-check-flags-a-question-put-to-the-human-whose-options"
severity: "minor"
category: "process"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
remedy: "Flag a question-tool call whose options carry no description or preview, at the point abcd already sees it (the guard's question-tool hook, which is scoped to the host's question tools) or as a transcript eval over recorded sessions; grounds as for iss-2609291925134691: GOV.UK Design System, Radios (each option carries its own short hint) and Nielsen Norman Group, 'Placeholders in Form Fields Are Harmful' (guidance must stay visible while answering); which surface carries the check is internal mechanism."
related_issues: ["iss-2609291925134691"]
---

No check flags a question put to the human whose options carry no example. GRILL rule 11 and the planning interview page (commands/intent.md) say the example that makes a question answerable goes in the question text and each option's preview carries its own concrete example, but nothing verifies a question against that, so the rule holds only while an agent remembers it. This is the eval or lint the remedy of iss-2609291925134691 floated and that record's fix did not build.
