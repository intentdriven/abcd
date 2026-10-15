---
schema_version: 1
id: "iss-2610151422085155"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/22-site.md"
remedy: "Rewrite .abcd/development/brief/04-surfaces/22-site.md:103-105 to match the shipped surface: The switchable set does match these seven (internal/core/site/pages.go PageNames). But the build writes more routes than the list names: `contributors/` always (internal/core/site/explorer.go:206, and the attribution gate reads it), and `record/foundations/`,…"
---

The brief at .abcd/development/brief/04-surfaces/22-site.md:103-105 says: "Every repository gets the same pages: the landing page, the record explorer, one page per record, the relationship graph, the timeline, the glossary and the status page." The shipped surface differs: The switchable set does match these seven (internal/core/site/pages.go PageNames). But the build writes more routes than the list names: `contributors/` always (internal/core/site/explorer.go:206, and the attribution gate reads it), and `record/foundations/`, `record/development/` and `references/` whenever their inputs exist (explorer.go:214-227). The chapter itself refers to the references and contributors pages at lines 189-191, but the page set it enumerates names none of the four. Repositories do not all get the same pages either: health, foundations, development, references and glossary each depend on their input. (v0.14.2 release crosscheck, false-claim, raised by opus in a17.)
