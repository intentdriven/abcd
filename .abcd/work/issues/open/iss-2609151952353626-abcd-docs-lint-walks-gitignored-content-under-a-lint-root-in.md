---
schema_version: 1
id: "iss-2609151952353626"
slug: "abcd-docs-lint-walks-gitignored-content-under-a-lint-root-in"
severity: "minor"
category: "bug"
source: "agent-finding"
found_during: "peer session report 2026-09-15 (a teaching-repo session; second instance of the class behind iss-2609151150180583)"
origin: researcher-authored
production_mode: hand-written
found_at: "internal/core/lint/lint.go"
---

abcd docs lint walks gitignored content under a lint root. In a repository whose lint roots include a delivery folder, a shipped fetch script writes assessment data and a cached git clone of the abcd repository under that folder, which .gitignore excludes; the lint read the whole cached clone (5,391 findings, 94 blockers, all inside the ignored folder, none in the repository's own prose). exempt_paths covers the content-authoring families only, so with the folder exempted ten links_resolve blockers still fire from inside the cache, whose record files link to paths that resolve only in the abcd checkout. A gitignored path is by definition not the repository's documentation; the lint could ask git check-ignore once per root and prune ignored directories, the way the site publisher already does, and name what it pruned. Workaround the adopter took: the fetch script keeps its clone cache outside every lint root. Observed on the v0.8.0 plugin binary. Second instance of the class behind iss-2609151150180583 (exempt_paths cannot excuse links_resolve).
