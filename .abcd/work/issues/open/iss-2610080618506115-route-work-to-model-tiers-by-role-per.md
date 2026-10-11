---
schema_version: 1
id: "iss-2610080618506115"
slug: "route-work-to-model-tiers-by-role-per"
severity: "minor"
category: "future-work-seed"
source: "user-observation"
found_during: "the guard-refusal session of 2026-10-07"
origin: researcher-authored
production_mode: hand-written
found_at: "model routing (internal/core/oracle)"
refines: [itd-2609170822093401]
remedy: "Measure first: in a lab, or by keeping Opus live and running Haiku on the same tasks in the shadow for a later evaluation, establish which tasks the quick and default tiers handle as well as the thinking tier; then add per-task routing over role-named tiers (thinking, default, quick) that each model family translates into its own models."
---

Route work to model tiers by role, per task as well as per agent: thinking (Opus today), default (Sonnet) and quick (Haiku), so that, for example, a summary written for the product thinker goes to the quick tier rather than the thinking tier. The tier names are what abcd stores; each model family maps them to its own models, so a move to another vendor changes the mapping and nothing else. This refines the planned per-agent tier table (itd-2609170822093401), which has economy and frontier but no middle tier and no per-task routing. Before a cheaper tier takes over any task it should earn it: keep the thinking tier doing the live work and run the quick tier on the same task in the shadow, recording both outputs, so a later evaluation can show where the quick tier is good enough.
