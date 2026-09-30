---
schema_version: 1
id: "iss-2609300916465620"
slug: "notewithheld-compares-a-rules-json-override-of-a-guardrail"
severity: "minor"
category: "inconsistency"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/rules/rules.go"
remedy: "Compare against the base withRepoShellDomain built in Load, before any rules.json layer, and reword the note so a withheld entry may be one the repository's guard.json declares. Grounds: review probe P6b of feat/guard-teach-repo-entries."
---

noteWithheld compares a rules.json override of a guardrail domain against defaultRuleSet rather than the base Load built, so a SHELL override that omits a lesson taught from the repository's own .abcd/guard.json never names it (the note says WITHHOLDS 17 of its 17 and leaves the repository entry out), and the note calls every withheld entry one 'abcd ships'.
