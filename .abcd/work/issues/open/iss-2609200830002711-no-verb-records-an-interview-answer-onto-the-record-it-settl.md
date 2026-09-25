---
schema_version: 1
id: "iss-2609200830002711"
slug: "no-verb-records-an-interview-answer-onto-the-record-it-settl"
severity: "minor"
category: "future-work-seed"
source: "agent-finding"
found_during: "Gropius sub-agent-lane experiment, session gropiusllm-2b, relayed to abcd-17 on 2026-09-20"
origin: researcher-authored
production_mode: hand-written
found_at: "commands/intent.md"
deferred_after: "v0.10.0"
deferral_reason: "ruling owed to the product thinker (away; run A 2026-09-25): Plan an intent answer write path that records interview answers on the record and in the decision log?"
---

No verb records an interview answer onto the record it settles. The planning interview on the intent surface page resolves open questions, the mechanism claim, the scope conditions and every acceptance criterion one at a time with the human, and step 9 says "edit the draft file to the confirmed content": each answer is prose the session writes by hand into the draft, and a repository that also logs decisions writes it a second time into its decision log, two writes that drift. Only the grounds step has a verb (intent ready --grounds). Relayed from the Gropius session gropiusllm-2b on 2026-09-20, running a one-decision-at-a-time interview over held drafts and captures with the answers fanned out to sub-agent lanes: every lane appended the answer to the draft and to DECISIONS.md by hand, and nothing downstream can read the answers as data. Wanted: a write path of the shape abcd intent answer <itd-N> <question-ref> "<text>" that records the answer on the record (under the open question it settles, or as the Mechanism or a Scope Conditions bullet by target) and, where the repository keeps a decision log, appends the dated line there in the same write, so the planning verb can read what was answered and what is still owed. Intent-shaped (a capability); the routing is the product thinker's.
