---
schema_version: 1
id: "iss-2611211025172155"
slug: "ideate-reviewer-sources-taken-on-trust"
severity: "minor"
category: "process"
source: "agent-finding"
found_during: "a /abcd:ideate run in the abcd source checkout, 2026-11-20"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/ideate.md"
remedy: "Reword the leg 3 ask in the page: a reviewer lists only what it opened during the review, a source handed to it in the findings is not its own, and the host records an empty list for a reviewer it saw open nothing; the verdict document gains a field for each reviewer in which the host states whether it saw the reviewer open its sources, and the rendered verdict labels a list the host did not see opened as reported only. Grounds: internal, the leg 1 rule that a claim read at second hand is not a check."
---

The sources of a leg 3 reviewer are recorded on the word of the reviewer. The ideate page asks each reviewer for the sources it consulted, `ideate record` checks their shape and that a record id resolves, and the written verdict lists them under that reviewer. That list is the fix of iss-2610141647308358, which recorded on 2026-10-14 an evaluator that made a single tool call and never opened the repository it judged, and which was resolved by recording what each reviewer consulted (itd-2610141646454347). This record is what that fix left. On 2026-11-20 a reviewer that made no tool call during its review returned a list of sources. The host had seen that it opened nothing and recorded the list as empty; a host that does not look writes into the verdict sources that nobody opened, and the record cannot show the difference. The recorder's own comment says the binary cannot observe how an agent was run (validateAdversarial in internal/core/ideate/record.go), so what can be had is an honest label of who saw what, not a check. iss-2611211024539425 would have the binary print the leg 3 brief; where that is built first, this wording goes into that brief and not into the page alone.
