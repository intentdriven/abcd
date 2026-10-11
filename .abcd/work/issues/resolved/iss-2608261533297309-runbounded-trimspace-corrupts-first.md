---
schema_version: 1
id: "iss-2608261533297309"
slug: "runbounded-trimspace-corrupts-first"
severity: "nitpick"
category: "bug"
source: "agent-observation"
found_during: "bughunt-a round 9"
found_at: "internal/gitutil/repo.go"
resolution: "runBounded trims the trailing side only; NUL-list first-entry fidelity pinned by a leading-space fixture"
impact: fix
resolved_by:
  commit: "89603e7c"
---

gitutil.runBounded returns strings.TrimSpace over the whole capture buffer, and lifeboat's pathIsIgnored NUL-splits that string from ls-files -z to build the not-ignored set. The -z form exists so whitespace in filenames cannot desync the list, but the shared trim strips leading whitespace off the first entry, so a repo whose first-sorting path begins with space/tab/CR/newline gets that one file silently classified ignored and dropped from the evidence walk — the quiet-evidence-loss shape the adapter's own contract forbids. Verified: -z disables quoting, the trailing NUL survives (NUL is not IsSpace), only the leading side is damaged; TrackedFiles bypasses the trim via raw Output. One-path blast radius, rare shape. Acceptance: NUL-list consumers receive the raw buffer (or a trailing-only trim) with a watched-fail test on a leading-whitespace name.