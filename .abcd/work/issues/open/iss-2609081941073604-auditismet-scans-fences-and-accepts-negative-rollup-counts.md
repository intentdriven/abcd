---
schema_version: 1
id: "iss-2609081941073604"
slug: "auditismet-scans-fences-and-accepts-negative-rollup-counts"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "bughunt-triage"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/site/compose.go"
---

auditIsMet says in its comment that it reads the Audit Notes rollup, but the body greps every raw line of the intent file for the substring Acceptance rollup:, fenced blocks and frontmatter included, and parses the count with fmt.Sscanf %d, which accepts negatives. A fenced NOT_MET -1 can therefore cancel a real NOT_MET 1 whenever MET is already above zero, featuring an intent whose acceptance is not met. A lone negative does not feature (notMet == -1), newestMetIntent still requires the shipped/ bucket and a written press release, and site check does not re-grade the rollup. strings.Cut is a substring match, not a section walker; the same package already skips fences when splitting headings (Sections). Fix: scan only the Audit Notes section, fence-aware, reusing Sections, and refuse n < 0. Refusing negatives alone is not enough: a fenced Acceptance rollup: MET 1 with no NOT_MET part can still flip a concerns-only rollup that already has notMet == 0. Detector: an intent whose real rollup is NOT_MET 1 plus a fenced Acceptance rollup: NOT_MET -1 must not be featured, while the same intent carrying only the honest NOT_MET 0 / MET 1 rollup in Audit Notes still is, given the other newestMetIntent conditions. Independent of the releaseOf substring match. Reported as GitHub issue 625 against ec7f40d6.
