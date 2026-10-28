---
schema_version: 1
id: "iss-2610141647315978"
slug: "abcd-lab-harvest-refuses-probe-citations"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "lab lab-261014142726-9c4d759 (a hand-run of /abcd:ideate on one external write-up, 2026-10-14)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lab/harvest.go"
remedy: "Accept either form in findings.md (strip a state/probes/<name>/record.md path to its probe name) and say which in the scaffold and the harvest help."
---

abcd lab harvest refuses probe citations given by path although its own help says each finding cites its probe records by path: findings citing state/probes/<name>/record.md are refused as 'incomplete: the record', because the parser looks the text up as a bare probe name; an earlier lab of the same day cites by path throughout (lab finding F-13).
