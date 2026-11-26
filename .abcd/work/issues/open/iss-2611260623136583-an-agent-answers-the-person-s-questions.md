---
schema_version: 1
id: "iss-2611260623136583"
slug: "an-agent-answers-the-person-s-questions"
severity: "minor"
category: "process"
source: "user-observation"
found_during: "a question about running a build from a plain terminal, 2026-11-26"
origin: researcher-authored
production_mode: hand-written
found_at: "AGENTS.md"
remedy: "Add a rule to the rules loader's bundled OPINIONS domain, recalling on question words about the project's state (where are we, is it possible, planned, already, status, who is building): a question about the project's state, plans or capabilities is answered from the record read in that turn (abcd status, abcd <id>, the intents, issues and decisions folders, the command pages, abcd peers), each claim citing the record id or page it rests on, and a claim the record does not settle is said to be unsettled rather than answered from memory. Prove it with a rules test that the domain recalls on such a prompt and injects the rule."
---

An agent answers the person's questions about the project (where a capability stands, whether something is planned or already built, who is working on what) from its own memory or the conversation instead of reading the record first, so an answer can be stale or wrong while sounding certain. The person asked on 2026-11-26 that the record always be checked instead: the intents in every bucket, the issues, the decisions, the command pages and what peers and reservations show.
