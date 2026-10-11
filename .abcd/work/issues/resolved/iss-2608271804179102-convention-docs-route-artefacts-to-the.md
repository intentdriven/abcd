---
schema_version: 1
id: "iss-2608271804179102"
slug: "convention-docs-route-artefacts-to-the"
severity: "minor"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/work/reviews/README.md"
resolution: "retired .abcd/logbook/ routes repointed to .work.local/logs/ in the reviews charter and research-notes README"
impact: internal
resolved_by:
  commit: "a032377d"
---

two convention documents route artefacts to the retired .abcd/logbook/ tier: the reviews charter sends per-invocation artifacts (oracle audits, grill reports, disembark audits) there and the research-notes README sends run logs and ephemeral acceptance output there, but .abcd/logbook/ does not exist, is not tracked, and is not gitignored — the sanctioned destination is .abcd/.work.local/logs/. Repoint both clauses in one sweep, citing the iss-56 adjudication; this is the deferred half of iss-73's own resolution note.