---
schema_version: 1
id: "iss-2610301943552138"
slug: "abcd-records-nothing-about-what-each"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "planning itd-2610230756259215"
origin: researcher-authored
production_mode: hand-written
found_at: "agents/question-drafter.md"
remedy: "Record, for every question abcd asks, the tokens it cost: the question-writing agent's usage for that question (input, output and cache tokens, the number of checks and drafts, and the model, as the host reports them for the sub-agent), stored with the question's answer in the interview's answers record, and summed per interview; report it beside the answers, and fold the per-question figures into the session and spec totals of itd-2609292107351737 when that draft is widened into a tokenomics intent."
refines: [itd-2609292107351737]
---

abcd records nothing about what each question it puts to a person costs in tokens. A question passes through the question-writing agent, which may draft and check it several times before it passes (up to five checks to a part under itd-2610230756259215), so one question can cost far more than the text the person sees, and no record says how much: a single drafter run in an interview on 2026-10-30 used between 58 and 73 thousand tokens, and one on 2026-10-23 used about 174 thousand while counting rows by hand. Without the count, nobody can tell whether the writer's loop is worth its cost, compare drafting strategies, or see what an interview costs as a whole. The product thinker asked for the tokens of each question to be recorded, as part of a wider account of what abcd's work costs in tokens (a tokenomics intent), whose nearest home is the draft itd-2609292107351737, which measures a development session's token cost against its spec's predicted size class.
