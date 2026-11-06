---
schema_version: 1
id: "iss-2611060809453708"
slug: "04-surfaces-11-history-md-250-says-what"
severity: "minor"
category: "inconsistency"
source: "user-observation"
found_during: "the v0.16.2 release gates"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/11-history.md"
remedy: "Scope the sentence to the notices and list what else the hook prints, saying why none of it is read from a tracked file."
---

`04-surfaces/11-history.md:250` says what the session-start hook prints on its output stream is one fixed sentence and a count. Stdout also carries the `## AWAY` block (`internal/surface/cli/cli.go:2166`), the inbox greeting (cli.go:2323), the update-news line, a second constant line when commit checks are off (cli.go:2381) and the checklist (`internal/surface/cli/hook_checklist.go:39-44`). `04-surfaces/29-report.md:109` and `04-surfaces/40-checklist.md:71-75` describe some of these, so the chapters disagree. The reason the sentence gives holds for each: none of them carries text from a tracked file.
