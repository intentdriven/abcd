---
schema_version: 1
id: "iss-2610100626327722"
slug: "when-an-abcd-question-is-refused-for-another-fault-the"
severity: "minor"
category: "ux"
source: "user-observation"
found_during: "the verb-split sign-off interview, 2026-10-10"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/surface/cli/guard_question.go"
remedy: "In a refusal for another fault, list a rows overrun after the refusing parts, on a line saying it does not refuse on its own and the question would be shown, with a test that a mode or layout refusal carrying a rows overrun says so."
---

When an abcd question is refused for another fault, the refusal lists the question's rows overrun among the parts to fix ('fix each and ask again'), although the rows limit alone never refuses (the question gate's rowsOnly, iss-2610070637562567). The person and the agent read it as a cause of the refusal, and the agent cuts a question the gate would have shown; seen twice in one interview on 2026-10-10.
