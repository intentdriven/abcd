---
schema_version: 1
id: "iss-2611211631298699"
slug: "the-guard-s-rule-for-a-name-search"
severity: "minor"
category: "observation"
source: "agent-observation"
found_during: "review-of-guard-coverage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/guard"
remedy: "Treat any process listing that reaches the stop command as a name search unless it lists only named pids"
---

The guard's rule for a name-search process stop recognises a process listing filtered through grep or awk before the stop, but not through sed, perl, cut or other filters.
