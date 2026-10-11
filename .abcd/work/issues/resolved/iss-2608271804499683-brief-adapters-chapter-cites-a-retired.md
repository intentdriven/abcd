---
schema_version: 1
id: "iss-2608271804499683"
slug: "brief-adapters-chapter-cites-a-retired"
severity: "nitpick"
category: "observation"
source: "agent-finding"
found_during: "structural consistency review of .abcd/ and docs/ (2026-08-27)"
found_at: ".abcd/development/brief/05-internals/02-adapters.md"
resolution: "brief adapters/agents chapters now cite the dated reviews-charter grammar as source of truth"
impact: internal
resolved_by:
  commit: "a032377d"
---

the brief's adapters chapter cites a retired reviews layout: 02-adapters has the RepoPrompt adapter falling back to flat .abcd/work/reviews/*.md and its adapter table repeats that source, but the reviews charter's grammar is dated subdirectories (<YYYY-MM-DD>-<scope>/*.md) and the flat form does not exist in the tree; 01-agents' review-collator glob has the same problem plus an input the charter excludes from the folder. Correct both citations to the charter grammar and cite reviews/README.md as the source of truth so the brief stops carrying a second layout.