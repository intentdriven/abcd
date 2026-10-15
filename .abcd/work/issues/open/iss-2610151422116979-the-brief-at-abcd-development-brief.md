---
schema_version: 1
id: "iss-2610151422116979"
slug: "the-brief-at-abcd-development-brief"
severity: "minor"
category: "drift"
source: "agent-finding"
found_during: "v0.14.2 release review, iss35 brief-surface crosscheck, content commit 87513ff33"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/05-internals/03-configuration.md"
remedy: "Rewrite .abcd/development/brief/05-internals/03-configuration.md:542 to match the shipped surface: Further config reads go through layered.Load(layered.Config, ...), which includes ~/.abcd.noindex/config.json: oracle.review/oracle.rp (internal/core/oracle/review.go), interview.list (internal/core/layered/interview.go, machine layer only),…"
duplicates: [iss-2610151422111534]
---

The brief at .abcd/development/brief/05-internals/03-configuration.md:542 says: The machine layer config.json's readers are the provider adapter's and the filing-time match's `match` keys ... every other config read resolves the repo-scope `.abcd/config.json` alone. The shipped surface differs: Further config reads go through layered.Load(layered.Config, ...), which includes ~/.abcd.noindex/config.json: oracle.review/oracle.rp (internal/core/oracle/review.go), interview.list (internal/core/layered/interview.go, machine layer only), reservation.github_username (internal/core/claim/holder.go, machine layer only) and consistency.max_request_bytes (internal/core/intent/consistency_config.go); pace.go and runner/config.go call the layered reader too. The chapter's own interview and reviewer-route sections also say those keys are read from the machine layer. (v0.14.2 release crosscheck, false-claim, raised by opus in a34.)
