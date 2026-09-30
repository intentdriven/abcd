---
schema_version: 1
id: "iss-2609300916451435"
slug: "a-repository-guard-json-entry-s-why-and-successor-are"
severity: "minor"
category: "security"
source: "review-followup"
found_during: "autonomous run 2026-09-23"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard/guard.go"
remedy: "Cap why and successor each at a documented byte bound in guard.Validate, chosen from the bundled registry's longest entry with headroom, so an over-bound entry refuses the file loudly; and have the injection budget's trailer name the file whose rules filled it (guard.json or the layer's rules.json) instead of always .abcd/rules.json. Grounds: the review probe of feat/guard-teach-repo-entries reproduced the drop and the misattribution."
---

A repository guard.json entry's why and successor are unbounded: the SHELL teaching plane injects both word for word, so one committed entry with a 240 KB why pushed every other domain, bundled hazards included, out of the 64 KiB injection budget; guard.Validate checked only that they are non-empty, 03-configuration.md's 'about a hundred tokens' per entry was a hope rather than a bound, and the truncation trailer always blamed .abcd/rules.json even when guard.json filled the budget.
