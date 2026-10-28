---
schema_version: 1
id: "iss-2610230851446558"
slug: "17-guard-md-says-a-broken-guard-never"
severity: "minor"
category: "documentation"
source: "review-followup"
found_during: "v0.15.2 release review"
origin: researcher-authored
production_mode: hand-written
found_at: ".abcd/development/brief/04-surfaces/17-guard.md"
remedy: "State the legacy-folder exception in the guard chapter's fail-open paragraph, with its reason and the rename that clears it."
---

17-guard.md says a broken guard never bricks a session, but hooks/pre-tool-use.sh exits 2 (blocking) on every shell and question call while a legacy ~/.abcd folder exists, until it is moved.
